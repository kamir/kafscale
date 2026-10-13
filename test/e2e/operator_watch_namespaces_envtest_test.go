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
	"context"
	"path/filepath"
	"testing"
	"time"

	appsv1 "k8s.io/api/apps/v1"
	autoscalingv2 "k8s.io/api/autoscaling/v2"
	batchv1 "k8s.io/api/batch/v1"
	corev1 "k8s.io/api/core/v1"
	policyv1 "k8s.io/api/policy/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sruntime "k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"k8s.io/apimachinery/pkg/util/wait"
	"k8s.io/utils/ptr"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/config"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
	metricsserver "sigs.k8s.io/controller-runtime/pkg/metrics/server"

	kafscalev1alpha1 "github.com/KafScale/platform/api/v1alpha1"
	"github.com/KafScale/platform/pkg/operator"
)

// TestOperatorWatchNamespaces runs an operator restricted to one namespace and
// creates the same KafscaleCluster in a second namespace and then in the
// watched one. The operator must leave the second namespace untouched and
// reconcile the watched one.
func TestOperatorWatchNamespaces(t *testing.T) {
	setupTestLogger()
	if !parseBoolEnv("KAFSCALE_E2E") {
		t.Skip("set KAFSCALE_E2E=1 to run operator envtest")
	}
	if !envtestAssetsAvailable() {
		t.Skip("envtest assets missing; set KUBEBUILDER_ASSETS or install setup-envtest")
	}

	t.Setenv("KAFSCALE_OPERATOR_ETCD_ENDPOINTS", "")
	t.Setenv("KAFSCALE_OPERATOR_ETCD_SNAPSHOT_SKIP_PREFLIGHT", "1")
	t.Setenv("KAFSCALE_OPERATOR_ETCD_SILENCE_LOGS", "1")

	const (
		watched   = "watched"
		unwatched = "unwatched"
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

	scheme := k8sruntime.NewScheme()
	utilruntime.Must(kafscalev1alpha1.AddToScheme(scheme))
	utilruntime.Must(corev1.AddToScheme(scheme))
	utilruntime.Must(appsv1.AddToScheme(scheme))
	utilruntime.Must(autoscalingv2.AddToScheme(scheme))
	utilruntime.Must(batchv1.AddToScheme(scheme))
	utilruntime.Must(policyv1.AddToScheme(scheme))

	// The manager client reads through the restricted cache and cannot see the
	// unwatched namespace, so the test talks to the API server directly.
	direct, err := client.New(cfg, client.Options{Scheme: scheme})
	if err != nil {
		t.Fatalf("new direct client: %v", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)

	for _, ns := range []string{watched, unwatched} {
		if err := direct.Create(ctx, &corev1.Namespace{ObjectMeta: metav1.ObjectMeta{Name: ns}}); err != nil {
			t.Fatalf("create namespace %s: %v", ns, err)
		}
	}

	mgr, err := ctrl.NewManager(cfg, ctrl.Options{
		Scheme: scheme,
		Cache:  operator.CacheOptions([]string{watched}),
		Metrics: metricsserver.Options{
			BindAddress: "0",
		},
		Controller: config.Controller{
			SkipNameValidation: ptr.To(true),
		},
	})
	if err != nil {
		t.Fatalf("new manager: %v", err)
	}

	publisher := operator.NewSnapshotPublisher(mgr.GetClient())
	if err := operator.NewClusterReconciler(mgr, publisher).SetupWithManager(mgr); err != nil {
		t.Fatalf("cluster reconciler: %v", err)
	}
	if err := operator.NewTopicReconciler(mgr, publisher).SetupWithManager(mgr); err != nil {
		t.Fatalf("topic reconciler: %v", err)
	}

	go func() {
		if err := mgr.Start(ctx); err != nil {
			t.Errorf("manager error: %v", err)
		}
	}()

	const clusterName = "scoped"
	newCluster := func(ns string) *kafscalev1alpha1.KafscaleCluster {
		return &kafscalev1alpha1.KafscaleCluster{
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
	}
	assertUnwatchedUntouched := func() {
		t.Helper()
		var sets appsv1.StatefulSetList
		if err := direct.List(ctx, &sets, client.InNamespace(unwatched)); err != nil {
			t.Fatalf("list statefulsets in the unwatched namespace: %v", err)
		}
		if len(sets.Items) != 0 {
			t.Fatalf("operator reconciled the unwatched namespace: found StatefulSet %q", sets.Items[0].Name)
		}
		var services corev1.ServiceList
		if err := direct.List(ctx, &services, client.InNamespace(unwatched)); err != nil {
			t.Fatalf("list services in the unwatched namespace: %v", err)
		}
		if len(services.Items) != 0 {
			t.Fatalf("operator reconciled the unwatched namespace: found Service %q", services.Items[0].Name)
		}
	}

	// Phase 1: only the unwatched namespace has a cluster. An operator that sees
	// every namespace creates the managed etcd resources for it within a few
	// seconds. A restricted operator must not react at all.
	if err := direct.Create(ctx, newCluster(unwatched)); err != nil {
		t.Fatalf("create cluster in %s: %v", unwatched, err)
	}
	for deadline := time.Now().Add(15 * time.Second); time.Now().Before(deadline); time.Sleep(250 * time.Millisecond) {
		assertUnwatchedUntouched()
	}

	// Phase 2: the same cluster in the watched namespace is reconciled. This
	// also shows that the operator was running during phase 1.
	if err := direct.Create(ctx, newCluster(watched)); err != nil {
		t.Fatalf("create cluster in %s: %v", watched, err)
	}
	if err := wait.PollUntilContextTimeout(ctx, 200*time.Millisecond, 30*time.Second, true, func(ctx context.Context) (bool, error) {
		return resourceExists(ctx, direct, &appsv1.StatefulSet{}, watched, clusterName+"-etcd") &&
			resourceExists(ctx, direct, &corev1.Service{}, watched, clusterName+"-etcd-client"), nil
	}); err != nil {
		t.Fatalf("expected managed etcd resources in the watched namespace: %v", err)
	}

	assertUnwatchedUntouched()
}
