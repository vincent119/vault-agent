# Vault-Agent Docs (English)

This is the formal documentation entrypoint for Vault-Agent.
For Traditional Chinese, see [`docs/README.zh-Hant.md`](README.zh-Hant.md).

## Table of Contents

- [Vault-Agent Docs (English)](#vault-agent-docs-english)
  - [Table of Contents](#table-of-contents)
  - [Additional docs](#additional-docs)
  - [1. Overview](#1-overview)
  - [2. Endpoints](#2-endpoints)
  - [3. Pod Injection (Labels / Annotations)](#3-pod-injection-labels--annotations)
    - [3.1 Label (Webhook filter)](#31-label-webhook-filter)
    - [3.2 Annotations (Injection config)](#32-annotations-injection-config)
  - [4. Background Sync (Sync Worker)](#4-background-sync-sync-worker)
  - [5. Configuration](#5-configuration)
  - [6. Vault Authentication](#6-vault-authentication)
  - [7. Metrics / Telemetry](#7-metrics--telemetry)
    - [7.1 Metrics](#71-metrics)
    - [7.2 Telemetry (Tracing)](#72-telemetry-tracing)
  - [8. Local Development](#8-local-development)
  - [9. Kubernetes Deployment (Kustomize)](#9-kubernetes-deployment-kustomize)
  - [10. Docker](#10-docker)

## Additional docs

- Configuration: [`docs/config.en.md`](config.en.md)
- Labels / Annotations: [`docs/annotations.en.md`](annotations.en.md)
- Observability: [`docs/observability.en.md`](observability.en.md)
- Deployment: [`docs/deploy.en.md`](deploy.en.md)
- Vault server setup (K8s Auth/Policy/Role): [`docs/vault-server-setup.en.md`](vault-server-setup.en.md)
- AWS Secrets Manager: [`docs/aws-secrets-manager.en.md`](aws-secrets-manager.en.md)
- **Architecture & flow diagrams** (Vault auth & fetch): [`docs/architecture-diagrams.en.md`](architecture-diagrams.en.md)

---

## 1. Overview

Vault-Agent is a Kubernetes controller that provides:

- A **mutating webhook** (`POST /mutate`) that returns JSON Patch to inject Pod env
- A **background sync worker** to reconcile Kubernetes Secrets from Vault / AWS Secrets Manager
- Observability via **Prometheus metrics**, optional **OTLP tracing**, and structured logs (zlogger)

---

## 2. Endpoints

- **`POST /mutate`**
  - Kubernetes mutating webhook endpoint (AdmissionReview v1)
  - Returns JSON Patch (may be empty `[]`)
  - Adds `X-Request-ID` header and injects request_id into context for log correlation
- **`GET /healthz`**: health probe (200 = OK)
- **`GET /metrics`**: Prometheus metrics (toggleable, optional Basic Auth)

---

## 3. Pod Injection (Labels / Annotations)

The webhook only processes Pods matched by `objectSelector` (see `deployments/kustomize/base/webhook.yaml`), and it also requires explicit opt-in via annotations.

### 3.1 Label (Webhook filter)

- `inject-vault-agent: "true"`

### 3.2 Annotations (Injection config)

- `inject-vault-agent: "true"`: enable injection (required)
- `inject-vault-agent.backend`: `vault` (default) or `aws`
- `inject-vault-agent.path`: Vault KV v2 path or AWS secret name (required)
- `inject-vault-agent.keys`: JSON array string (optional), e.g. `["DB_USER","DB_PASS"]`
  - empty / omitted: return all key-value pairs from the secret

---

## 4. Background Sync (Sync Worker)

When Vault-Agent can connect to Kubernetes (in-cluster, or `k8s.kubeconfig` is provided), it will:

- periodically list Secrets labeled with `inject-vault-agent=true`
- parse backend/path/keys from Secret annotations
- fetch data from the backend and update `Secret.Data`
- only call Update when data differs

If you only need the webhook, you can run without Kubernetes connectivity and the sync worker will be disabled automatically.

---

## 5. Configuration

Load order: **OS env > `configs/config.yaml` > defaults**
See `configs/config.sample.yaml` and `configs/.env.example`.

Key points:

- **`k8s.kubeconfig`**:
  - empty: in-cluster only (**no implicit `~/.kube/config`**)
  - non-empty: use the given kubeconfig path
- **`log`** maps to `zlogger.Config`
- **`metrics.enabled`** and `metrics.basic_auth` control `/metrics`
- **`telemetry.enabled`** and `telemetry.otlp_endpoint` control tracing

---

## 6. Vault Authentication

Choose one of:

1. **Static token** (local/dev): `vault.token`
2. **Kubernetes auth** (in cluster):
   - `vault.auth_k8s_path` (e.g. `im-devops-eks`)
   - `vault.auth_k8s_role` (e.g. `vault-agent`)

---

## 7. Metrics / Telemetry

### 7.1 Metrics

- `/metrics` is exposed only when `metrics.enabled: true`
- if `metrics.basic_auth` is non-empty (`user:password`), `/metrics` requires HTTP Basic Auth

### 7.2 Telemetry (Tracing)

- tracer initializes only when `telemetry.enabled: true`
- `telemetry.otlp_endpoint`:
  - empty: stdout exporter
  - non-empty: OTLP gRPC exporter

---

## 8. Local Development

```bash
make run
make test
```

To enable the sync worker locally, provide `k8s.kubeconfig`; otherwise only HTTP endpoints will run.

---

## 9. Kubernetes Deployment (Kustomize)

Baseline manifests live in `deployments/kustomize/base/`:

```bash
kubectl apply -k deployments/kustomize/base
```

Notes:

- webhook `failurePolicy: Ignore` to avoid blocking Pod creation
- `objectSelector` filters Pods by `inject-vault-agent=true`
- runs as non-root by default (UID/GID 10000)

---

## 10. Docker

Multi-stage build with pinned Chainguard image digests. See `Dockerfile`.
