<!--
Copyright 2025 Alexander Alten (novatechflow), NovaTechflow (novatechflow.com).
This project is supported and financed by Scalytics, Inc. (www.scalytics.io).

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
-->

# Security Overview

Kafscale is a kubernetes native platform focused on Kafka protocol parity and
operational stability. This document summarizes the current security posture
and the boundaries of what is and is not supported in v1.

## Current Security Posture (v1)

- **Authentication**: none at the Kafka protocol layer. Brokers accept any
  client connection. The console UI supports basic auth via
  `KAFSCALE_UI_USERNAME` / `KAFSCALE_UI_PASSWORD`.
- **Authorization**: none. All broker APIs are unauthenticated and authorized
  implicitly. This includes admin APIs such as CreatePartitions and DeleteGroups.
- **Transport Security**: TLS is optional and must be enabled by operators via
  `KAFSCALE_BROKER_TLS_*` and `KAFSCALE_CONSOLE_TLS_*`. Plaintext is the default.
- **Secrets Handling**: S3 credentials are read from Kubernetes secrets and are
  not written to etcd or source control. The operator projects secrets into pods.
- **Data at Rest**: data is stored in S3 and etcd; encryption at rest depends on
  your infrastructure provider (bucket policies, KMS, disk encryption).
- **Network Trust**: the deployment assumes a private network or cluster-level
  controls (SecurityGroups, NetworkPolicies, ingress rules).

## Operational Guidance

- Deploy brokers and the console behind private networking or VPNs.
- Enable TLS for broker and console endpoints in production.
- Restrict ingress to only trusted clients and operator components.
- Use least-privilege IAM roles for S3 access and restrict etcd endpoints.
- Treat the console as privileged; do not expose it publicly without auth.

## Known Gaps

- No SASL or mTLS authentication for Kafka protocol clients.
- No ACLs or RBAC at the broker layer.
- No multi-tenant isolation.
- Admin APIs are writable without auth; UI is read-only by policy, not enforcement.

## Roadmap

Planned security milestones (order may change as requirements evolve):

- TLS enabled by default in production templates.
- SASL/PLAIN and SASL/SCRAM for Kafka client authentication.
- Authorization / ACL layer for broker admin and data plane APIs.
- Optional mTLS for broker and console endpoints.

## Reporting Security Issues

If you believe you have found a security vulnerability, please follow the
process in `SECURITY.md`.
