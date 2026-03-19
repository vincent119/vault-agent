# 設定（Config）— 繁體中文

## 讀取優先序

Vault-Agent 的設定讀取優先序為：

1. **OS 環境變數**
2. `configs/config.yaml`
3. 預設值

範例請見：

- `configs/config.sample.yaml`
- `configs/.env.example`

## 設定欄位

### `app`

- `app.name`：服務名稱（用於 telemetry service name）
- `app.port`：HTTP server port（預設 8080）
- `app.env`：`dev` / `prod`（影響 log 預設輸出格式）

### `log`

對應 `github.com/vincent119/zlogger` 的 `Config`：

- `log.level`：`debug/info/warn/error/fatal`
- `log.format`：`console/json`
- `log.outputs`：`[console]` 或 `["file"]` 或混用
- `log.log_path`：file output 目錄
- `log.file_name`：file output 檔名（空則依日期）
- `log.add_caller`：是否顯示 caller
- `log.add_stacktrace`：是否輸出 stacktrace
- `log.development`：開發模式
- `log.color_enabled`：console 顏色

> `app.env=prod` 時會覆寫部分行為（強制 `format=json`、`development=false`、`color_enabled=false`），避免 production console/color/development 設定誤用。

### `vault`

Vault KV v2 設定與認證（二擇一）：

- `vault.address`：Vault URL
- `vault.mount_path`：KV v2 mount（預設 `secret`）
- **Token（本機/dev）**
  - `vault.token`
- **Kubernetes Auth（K8s）**
  - `vault.auth_k8s_path`：Vault 內 K8s auth mount path（例：`devops-eks`）
  - `vault.auth_k8s_role`：Vault role（例：`vault-agent`）

### `aws`

- `aws.region`：AWS region

### `k8s`

- `k8s.kubeconfig`
  - 空字串：**只嘗試 In-Cluster Config**（不會預設讀 `~/.kube/config`）
  - 非空：使用指定 kubeconfig 檔案路徑
- `k8s.namespace`：限制 Sync Worker 掃描 namespace（空字串表示全部）

### `sync`

- `sync.interval_seconds`：Sync Worker 週期秒數（預設 60）

### `telemetry`

- `telemetry.enabled`：開關
- `telemetry.otlp_endpoint`：空字串 => stdout exporter；非空 => OTLP exporter（依 `otlp_transport` 選 gRPC 或 HTTP）
- `telemetry.otlp_transport`：`grpc`（預設，port 4317）或 `http`（port 4318）
- `telemetry.otlp_compression`：`none` 或 `gzip`
- `telemetry.otlp_headers`：認證等 header，格式 `key1=value1,key2=value2`
- `telemetry.otlp_basic_auth`：對 OTLP 收集器 Basic 認證，格式 `user:password`（gRPC/HTTP 皆支援）

### `metrics`

- `metrics.enabled`：開關
- `metrics.basic_auth`：非空時為 `user:password`，對 `/metrics` 啟用 HTTP Basic Auth

### `tls`

若要對外提供 HTTPS webhook：

- `tls.cert_file`
- `tls.key_file`
