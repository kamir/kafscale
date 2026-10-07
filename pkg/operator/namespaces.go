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
	"os"
	"strings"

	"sigs.k8s.io/controller-runtime/pkg/cache"
)

// WatchNamespacesEnv names the namespaces the operator reconciles, as a
// comma-separated list. Unset or empty means every namespace.
const WatchNamespacesEnv = "KAFSCALE_OPERATOR_WATCH_NAMESPACES"

// WatchNamespaces returns the namespaces from WatchNamespacesEnv in the order
// given, without blanks or duplicates. It returns nil when the operator is not
// restricted.
func WatchNamespaces() []string {
	var namespaces []string
	seen := map[string]struct{}{}
	for _, part := range strings.Split(os.Getenv(WatchNamespacesEnv), ",") {
		ns := strings.TrimSpace(part)
		if ns == "" {
			continue
		}
		if _, dup := seen[ns]; dup {
			continue
		}
		seen[ns] = struct{}{}
		namespaces = append(namespaces, ns)
	}
	return namespaces
}

// CacheOptions restricts a manager cache to the given namespaces. With no
// namespaces it returns the zero value, which keeps the cache cluster-wide.
func CacheOptions(namespaces []string) cache.Options {
	if len(namespaces) == 0 {
		return cache.Options{}
	}
	defaults := make(map[string]cache.Config, len(namespaces))
	for _, ns := range namespaces {
		defaults[ns] = cache.Config{}
	}
	return cache.Options{DefaultNamespaces: defaults}
}
