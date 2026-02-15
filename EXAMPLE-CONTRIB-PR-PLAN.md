# Example Contribution PR — Review Findings

This document summarizes findings from a code review of the KafScale repository,
focused on the example/demo content added in recent PRs and how it relates to the
core platform architecture.

---

## 1. Repository Structure Overview

The KafScale repository is organized as follows:

| Directory | Purpose |
|-----------|---------|
| `cmd/broker/` | Kafka-compatible broker binary |
| `cmd/proxy/` | Kafka protocol proxy binary |
| `pkg/` | Shared Go packages (protocol, metadata, broker, storage, etc.) |
| `deploy/` | Helm charts, Docker files, demo manifests |
| `addons/` | Processor add-ons (Iceberg, SQL, etc.) |
| `docs/` | Project documentation and release notes |
| `examples/` | Example and demo content |
| `scripts/` | Utility and demo scripts |

---

## 2. Core Architecture: The Proxy Component

The Kafka protocol **proxy** (`cmd/proxy/main.go`) is a central component of KafScale.
It was reviewed in detail. Key findings:

### 2.1 Proxy Struct

```go
type proxy struct {
    addr           string
    advertisedHost string
    advertisedPort int32
    store          metadata.Store
    backends       []string
    logger         *slog.Logger
    rr             uint32          // round-robin counter
    dialTimeout    time.Duration
    ready          uint32
    lastHealthy    int64
    cacheTTL       time.Duration
    cacheMu        sync.RWMutex
    cachedBackends []string
    apiVersions    []protocol.ApiVersion
}
```

### 2.2 What the Proxy Does

- **Intercepts Kafka wire protocol** requests from clients
- **Presents a single logical broker** to clients (rewrites leader/replica IDs to `0`,
  uses a single advertised host/port)
- **Round-robins** data-plane requests (Produce, Fetch, etc.) to actual broker backends
  via `connectBackend()`
- **Handles control-plane requests locally**: ApiVersions, Metadata, FindCoordinator
- **Reads cluster metadata from etcd** (`metadata.Store` interface)
- **Caches backend addresses** with a configurable TTL to survive transient etcd failures
- **Exposes health endpoints** (`/readyz`, `/livez`) for Kubernetes probes

### 2.3 Supported Kafka API Keys (Proxy)

The proxy advertises support for these Kafka protocol API keys:

| API Key | Name | Version Range |
|---------|------|---------------|
| 18 | ApiVersions | 0–4 |
| 3 | Metadata | 0–12 |
| 0 | Produce | 0–9 |
| 1 | Fetch | 11–13 |
| 10 | FindCoordinator | 3 |
| 2 | ListOffsets | 0–4 |
| 11 | JoinGroup | 4 |
| 14 | SyncGroup | 4 |
| 12 | Heartbeat | 4 |
| 13 | LeaveGroup | 4 |
| 8 | OffsetCommit | 3 |
| 9 | OffsetFetch | 5 |
| 15 | DescribeGroups | 5 |
| 16 | ListGroups | 5 |
| 23 | OffsetForLeaderEpoch | 3 |
| 32 | DescribeConfigs | 4 |
| 33 | AlterConfigs | 1 |
| 37 | CreatePartitions | 0–3 |
| 19 | CreateTopics | 0–2 |
| 20 | DeleteTopics | 0–2 |
| 42 | DeleteGroups | 0–2 |

### 2.4 Not Found: "LFSProxy"

A search for `LFSProxy` (case-insensitive, including variations like `lfs.proxy`,
`lfsproxy`) returned **no results**. This identifier does not exist anywhere in the
codebase. The only proxy component is the Kafka protocol proxy described above.

---

## 3. Example Content (PR #10)

The recent PR #10 merged into `main` added example and demo content:

### 3.1 `examples/101_kafscale-dev-guide/`

A developer guide collection with numbered markdown files:

- `01-architecture.md` — Platform architecture overview
- `02-getting-started.md` — Setup and first steps
- `03-configuration.md` — Configuration reference
- `04-scaling.md` — Scaling strategies
- `05-troubleshooting.md` — Common issues and debugging

### 3.2 `examples/E50_JS-kafscale-demo/`

A JavaScript demo application showing KafScale client usage:

- `package.json` / `package-lock.json` — Node.js project with `kafkajs` dependency
- `producer.js` / `consumer.js` — Simple produce/consume examples
- `KAFSCALE-COMPATIBILITY.md` — Documents KafScale's Kafka API compatibility

Key compatibility points documented:
- KafScale exposes a **single broker endpoint** (the proxy)
- Clients see one broker but scaling happens transparently behind the proxy
- Standard `kafkajs` library works without modification

### 3.3 `.gitignore` Update

Added `node_modules/` to `.gitignore` for the JS demo.

---

## 4. Broker Component

The broker (`cmd/broker/main.go`) is the other main binary. It:

- Listens on a configurable Kafka protocol port (default `:19092`)
- Registers itself in etcd for discovery by the proxy
- Handles the actual Kafka storage and protocol operations
- Supports PROXY protocol for extracting real client IPs (via `pkg/broker/proxyproto.go`)
- Has ACL support (`cmd/broker/acl_test.go`)
- Exposes health endpoints for Kubernetes

---

## 5. Add-on Processors

Located under `addons/processors/`:

| Processor | Purpose |
|-----------|---------|
| `iceberg-processor/` | Apache Iceberg table integration |
| `sql-processor/` | SQL query layer over Kafka topics (has its own internal proxy for caching) |

---

## 6. Deployment Artifacts

- **Helm charts**: `deploy/helm/kafscale/` with templates for proxy and operator deployments
- **Docker**: `deploy/docker/proxy.Dockerfile`
- **Demo**: `deploy/demo/nginx-lb.yaml` (nginx load balancer example)
- **Platform script**: `scripts/demo-platform.sh`

---

## 7. Summary

The KafScale platform is a Kafka-compatible streaming system with:

1. A **protocol proxy** that presents a single-broker facade to clients
2. A **scalable broker layer** behind the proxy
3. **etcd-based metadata** for service discovery and cluster coordination
4. **Add-on processors** for Iceberg and SQL workloads
5. **Helm/Docker deployment** tooling for Kubernetes

The example contributions (PR #10) add onboarding material and a working JS demo
that validates the proxy's Kafka compatibility. No `LFSProxy` component exists in
the codebase.
