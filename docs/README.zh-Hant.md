# Vault-Agent 文件（繁體中文）

本文件為 Vault-Agent 的正式說明入口（繁中與英文分檔）。
若你要英文版，請見 [`docs/README.en.md`](README.en.md)。

## 目錄

- [Vault-Agent 文件（繁體中文）](#vault-agent-文件繁體中文)
  - [目錄](#目錄)
  - [延伸文件](#延伸文件)
  - [1. 概覽](#1-概覽)
  - [2. 端點](#2-端點)
  - [3. Pod 注入（Annotation / Label）](#3-pod-注入annotation--label)
    - [3.1 Label（Webhook 過濾）](#31-labelwebhook-過濾)
    - [3.2 Annotation（注入設定）](#32-annotation注入設定)
  - [4. 背景同步（Sync Worker）](#4-背景同步sync-worker)
  - [5. 設定（Config）](#5-設定config)
    - [5.1 重要設定摘要](#51-重要設定摘要)
  - [6. Vault 認證](#6-vault-認證)
  - [7. Metrics / Telemetry](#7-metrics--telemetry)
    - [7.1 Metrics](#71-metrics)
    - [7.2 Telemetry（Tracing）](#72-telemetrytracing)
  - [8. 本機開發](#8-本機開發)
    - [8.1 指令](#81-指令)
  - [9. Kubernetes 部署（Kustomize）](#9-kubernetes-部署kustomize)
  - [10. Docker](#10-docker)

## 延伸文件

- 設定：[`docs/config.zh-Hant.md`](config.zh-Hant.md)
- Annotation / Label：[`docs/annotations.zh-Hant.md`](annotations.zh-Hant.md)
- 可觀測性：[`docs/observability.zh-Hant.md`](observability.zh-Hant.md)
- 部署：[`docs/deploy.zh-Hant.md`](deploy.zh-Hant.md)
- Vault Server 設定（K8s Auth/Policy/Role）：[`docs/vault-server-setup.zh-Hant.md`](vault-server-setup.zh-Hant.md)
- AWS Secrets Manager：[`docs/aws-secrets-manager.zh-Hant.md`](aws-secrets-manager.zh-Hant.md)
- **架構圖與流程圖**（Vault 認證與取值）：[`docs/architecture-diagrams.zh-Hant.md`](architecture-diagrams.zh-Hant.md)

---

## 1. 概覽

Vault-Agent 是一個 Kubernetes 同步控制器，提供：

- **Mutating Webhook**：`POST /mutate` 解析 AdmissionReview v1，回傳 JSON Patch 注入 Pod env
- **背景同步 Worker**：定期掃描 Kubernetes Secret 並從 Vault / AWS Secrets Manager 回寫更新
- **可觀測性**：`/metrics`、可選 tracing、結構化日誌（zlogger）

---

## 2. 端點

- **`POST /mutate`**
  - K8s Mutating Webhook endpoint
  - 回傳 JSON Patch（可能為空 `[]`）
  - 會回寫 `X-Request-ID`，並把 request_id 注入到 request context，用於串接日誌
- **`GET /healthz`**：健康檢查（200 = OK）
- **`GET /metrics`**：Prometheus metrics（可開關、可選 Basic Auth）

---

## 3. Pod 注入（Annotation / Label）

Webhook 只會處理符合 `objectSelector` 的 Pod（見 `deployments/kustomize/base/webhook.yaml`），並且 **仍需 annotation 明確 opt-in**。

### 3.1 Label（Webhook 過濾）

- `inject-vault-agent: "true"`

### 3.2 Annotation（注入設定）

- `inject-vault-agent: "true"`：啟用注入（必填）
- `inject-vault-agent.backend`：`vault`（預設）或 `aws`
- `inject-vault-agent.path`：Vault path（KV v2 的 path）或 AWS secret name（必填）
- `inject-vault-agent.keys`：JSON array 字串（可選），例如 `["DB_USER","DB_PASS"]`
  - 空/未填：回傳該路徑下所有 key-value

---

## 4. 背景同步（Sync Worker）

當程式能連到 Kubernetes（In-Cluster 或提供 `k8s.kubeconfig`）時：

- 會定期掃描 **帶有 label `inject-vault-agent=true`** 的 Secret
- 依 Secret 的 annotations 解析 backend/path/keys
- 從對應 backend 拉取資料並覆寫 `Secret.Data`
- 僅在有差異時才 Update（避免不必要的 API 呼叫）

> 若你只需要 webhook，可不提供 K8s 連線設定，Sync Worker 會自動停用。

---

## 5. 設定（Config）

設定讀取優先序：**OS Env > `configs/config.yaml` > 預設值**
範例請見 `configs/config.sample.yaml` 與 `configs/.env.example`。

### 5.1 重要設定摘要

- **`app`**：服務名稱、port、env
- **`log`**：對應 `zlogger.Config`（level/format/outputs...）
- **`vault`**：Vault 地址、KV v2 mount、Token 或 K8s Auth（二擇一）
- **`aws`**：region
- **`k8s.kubeconfig`**：
  - **空字串**：只嘗試 In-Cluster（不會預設讀 `~/.kube/config`）
  - **非空**：使用指定 kubeconfig 路徑
- **`telemetry.enabled`**：tracing 開關
- **`metrics.enabled`** / **`metrics.basic_auth`**：metrics 開關與 Basic Auth

---

## 6. Vault 認證

Vault 支援兩種方式（二擇一）：

1. **靜態 Token（本機 / dev）**：設定 `vault.token`
2. **Kubernetes Auth（K8s 部署）**：
   - `vault.auth_k8s_path`：Vault 內 K8s auth mount path（例如 `im-devops-eks`）
   - `vault.auth_k8s_role`：Vault role（例如 `vault-agent`）

---

## 7. Metrics / Telemetry

### 7.1 Metrics

- `metrics.enabled: true` 才會暴露 `/metrics`
- `metrics.basic_auth` 非空（格式 `user:password`）則對 `/metrics` 啟用 Basic Auth

### 7.2 Telemetry（Tracing）

- `telemetry.enabled: true` 才初始化 tracer
- `telemetry.otlp_endpoint`：
  - 空字串：stdout exporter（開發用）
  - 非空：OTLP gRPC exporter

---

## 8. 本機開發

### 8.1 指令

```bash
make run
make test
```

> 若要在本機跑 Sync Worker，請提供 `k8s.kubeconfig`；否則只會啟動 HTTP endpoints。

---

## 9. Kubernetes 部署（Kustomize）

baseline manifests 在 `deployments/kustomize/base/`：

```bash
kubectl apply -k deployments/kustomize/base
```

重點：

- Webhook `failurePolicy: Ignore`（避免 webhook 失效卡住 Pod 部署）
- `objectSelector` label 過濾 `inject-vault-agent=true`
- Deployment 預設以 non-root（UID/GID 10000）執行

---

## 10. Docker

本專案使用 multi-stage build，並以固定 digest 的 Chainguard 映像降低 CVE 風險，詳見 `Dockerfile`。
