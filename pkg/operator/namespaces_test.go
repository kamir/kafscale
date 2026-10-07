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

package operator

import (
	"reflect"
	"testing"
)

func TestWatchNamespaces(t *testing.T) {
	cases := []struct {
		name string
		env  string
		want []string
	}{
		{name: "unset means all namespaces", env: "", want: nil},
		{name: "only separators and blanks", env: " , ,", want: nil},
		{name: "single namespace", env: "kafscale", want: []string{"kafscale"}},
		{name: "several namespaces keep their order", env: "team-b,team-a", want: []string{"team-b", "team-a"}},
		{name: "surrounding whitespace is trimmed", env: " team-a , team-b ", want: []string{"team-a", "team-b"}},
		{name: "duplicates are dropped", env: "team-a,team-b,team-a", want: []string{"team-a", "team-b"}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			t.Setenv(WatchNamespacesEnv, tc.env)
			if got := WatchNamespaces(); !reflect.DeepEqual(got, tc.want) {
				t.Fatalf("WatchNamespaces() = %v, want %v", got, tc.want)
			}
		})
	}
}

func TestCacheOptionsUnrestricted(t *testing.T) {
	if got := CacheOptions(nil); got.DefaultNamespaces != nil {
		t.Fatalf("expected a cluster-wide cache, got namespaces %v", got.DefaultNamespaces)
	}
}

func TestCacheOptionsRestricted(t *testing.T) {
	got := CacheOptions([]string{"team-a", "team-b"})
	if len(got.DefaultNamespaces) != 2 {
		t.Fatalf("expected 2 namespaces in the cache, got %v", got.DefaultNamespaces)
	}
	for _, ns := range []string{"team-a", "team-b"} {
		if _, ok := got.DefaultNamespaces[ns]; !ok {
			t.Fatalf("namespace %q missing from the cache configuration", ns)
		}
	}
}
