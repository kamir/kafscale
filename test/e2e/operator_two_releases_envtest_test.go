// Copyright 2025 Alexander Alten (novatechflow), NovaTechflow (novatechflow.com).
// This project is supported and financed by Scalytics, Inc. (www.scalytics.io).
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

//go:build e2e

package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	kafscalev1alpha1 "github.com/KafScale/platform/api/v1alpha1"
	"github.com/KafScale/platform/pkg/operator"
)

// TestOperatorTwoReleasesShareCluster runs two operator processes against one
// API server, the way two Helm releases of the chart would run on one cluster.
// Each process is restricted to its own namespace and configured with its own
// broker image. Each must write its own image into the broker StatefulSet of
// its namespace, and neither may touch the other one.
//
// The broker image comes from the operator environment, so two operators that
// both reconcile a cluster overwrite each other's StatefulSet in a tight loop.
// That is what this test observes when one of the two processes is not
// restricted: the generation of the StatefulSet climbs by dozens per second.
func TestOperatorTwoReleasesShareCluster(t *testing.T) {
	setupTestLogger()
	if !parseBoolEnv("KAFSCALE_E2E") {
		t.Skip("set KAFSCALE_E2E=1 to run operator envtest")
	}
	if !envtestAssetsAvailable() {
		t.Skip("envtest assets missing; set KUBEBUILDER_ASSETS or install setup-envtest")
	}

	const (
		namespaceA  = "release-a"
		namespaceB  = "release-b"
		imageA      = "example.invalid/kafscale-broker:release-a"
		imageB      = "example.invalid/kafscale-broker:release-b"
		clusterName = "shared"
	)

	crdPath := filepath.Join(repoRoot(t), "deploy", "helm", "kafscale", "crds")
	env := &envtest.Environment{
		CRDDirectoryPaths: []string{crdPath},
	}
	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	t.Cleanup(func() {
		if err := env.Stop(); err != nil {
			t.Fatalf("stop envtest: %v", err)
		}
	})

	workDir := t.TempDir()
	kubeconfig := filepath.Join(workDir, "kubeconfig")
	if err := os.WriteFile(kubeconfig, env.KubeConfig, 0o600); err != nil {
		t.Fatalf("write kubeconfig: %v", err)
	}
	binary := filepath.Join(workDir, "kafscale-operator")
	build := exec.Command("go", "build", "-o", binary, "./cmd/operator")
	build.Dir = repoRoot(t)
	if out, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build operator: %v\n%s", err, out)
	}

	scheme := k8sruntime.NewScheme()
	utilruntime.Must(kafscalev1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	direct, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new direct client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	for _, ns := range []string{namespaceA, namespaceB} {
		if err := direct.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}); err != nil {
			t.Fatalf("create namespace %s: %v", ns, err)
		}
	}

	startOperator := func(watch, brokerImage string) {
		t.Helper()
		logs := &bytes.Buffer{}
		cmd := exec.CommandContext(ctx, binary, "--leader-elect=false", "--metrics-bind-address=0")
		cmd.Env = append(os.Environ(),
			"KUBECONFIG="+kubeconfig,
			operator.WatchNamespacesEnv+"="+watch,
			"BROKER_IMAGE="+brokerImage,
			// A reachable etcd keeps a reconcile fast. The API server's own
			// etcd is good enough for the metadata the operator publishes.
			"KAFSCALE_OPERATOR_ETCD_ENDPOINTS="+env.ControlPlane.Etcd.URL.String(),
			"KAFSCALE_OPERATOR_ETCD_SNAPSHOT_SKIP_PREFLIGHT=1",
			"KAFSCALE_OPERATOR_ETCD_SILENCE_LOGS=1",
		)
		cmd.Stdout = logs
		cmd.Stderr = logs
		if err := cmd.Start(); err != nil {
			t.Fatalf("start operator for %q: %v", watch, err)
		}
		t.Cleanup(func() {
			cancel()
			_ = cmd.Wait()
			if t.Failed() {
				t.Logf("operator watching %q:\n%s", watch, logs.String())
			}
		})
	}
	startOperator(namespaceA, imageA)
	startOperator(namespaceB, imageB)

	for _, ns := range []string{namespaceA, namespaceB} {
		cluster := &kafscalev1alpha1.KafscaleCluster{
			ObjectMeta: metav1.ObjectMeta{
				Name:      clusterName,
				Namespace: ns,
			},
			Spec: kafscalev1alpha1.KafscaleClusterSpec{
				Brokers: kafscalev1alpha1.BrokerSpec{},
				S3: kafscalev1alpha1.S3Spec{
					Bucket: "snapshots",
					Region: "us-east-1",
				},
				Etcd: kafscalev1alpha1.EtcdSpec{},
			},
		}
		if err := direct.Create(ctx, cluster); err != nil {
			t.Fatalf("create cluster in %s: %v", ns, err)
		}
	}

	type brokerState struct {
		image      string
		generation int64
	}
	brokerSet := func(ns string) (brokerState, bool) {
		t.Helper()
		var sts appsv1.StatefulSet
		err := direct.Get(ctx, client.ObjectKey{Namespace: ns, Name: clusterName + "-broker"}, &sts)
		if apierrors.IsNotFound(err) {
			return brokerState{}, false
		}
		if err != nil {
			t.Fatalf("read broker StatefulSet in %s: %v", ns, err)
		}
		if len(sts.Spec.Template.Spec.Containers) == 0 {
			return brokerState{}, false
		}
		return brokerState{image: sts.Spec.Template.Spec.Containers[0].Image, generation: sts.Generation}, true
	}

	want := map[string]string{namespaceA: imageA, namespaceB: imageB}

	// Each release creates the broker StatefulSet of its own namespace.
	first := map[string]brokerState{}
	for deadline := time.Now().Add(30 * time.Second); len(first) < len(want) && time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		for ns := range want {
			if _, done := first[ns]; done {
				continue
			}
			if state, found := brokerSet(ns); found {
				first[ns] = state
			}
		}
	}
	if len(first) < len(want) {
		t.Fatalf("expected a broker StatefulSet in both namespaces within 30s, got %d", len(first))
	}

	// Keep watching. Every StatefulSet must carry the image of its own release,
	// and nobody may rewrite it: an unchanged generation shows a single writer.
	for deadline := time.Now().Add(10 * time.Second); time.Now().Before(deadline); time.Sleep(100 * time.Millisecond) {
		for ns, image := range want {
			state, found := brokerSet(ns)
			if !found {
				t.Fatalf("broker StatefulSet in %s disappeared", ns)
			}
			if state.image != image {
				t.Fatalf("broker StatefulSet in %s carries image %q, want %q: another operator release wrote to it", ns, state.image, image)
			}
			if state.generation != first[ns].generation {
				t.Fatalf("broker StatefulSet in %s went from generation %d to %d: more than one operator release writes to it", ns, first[ns].generation, state.generation)
			}
		}
	}
}
