# Vault-K8S 同步控制器 (vault-agent) 重構與功能擴充計畫

## 目標描述

將原有的 Python 專案重新打造為全新的 **Golang** 專案，並且依據要求：

1. **遵循 DDD 架構與專案規範**：嚴格遵守 `cmd/`、`internal/` 等標準目錄層級，全面採用領域驅動設計 (DDD)。
2. **解決資料同步問題 (背景同步)**：不僅是一個 Mutating Webhook，也具備 Controller / Background Worker 的能力，主動輪詢 K8s 上的 Secret 並自動由 Vault 覆蓋更新，確保應用程式永遠能拿到最新配置。
3. **擴充機密來源 (Secret Backend)**：除 HashiCorp Vault 外，新增支援 **AWS Secrets Manager**。
4. **增強可觀測性與維運品質**：整合 `Prometheus` 監控指標、`OTLP Tracing`、統一由 `zlogger` 輸出日誌，並嚴格落實優雅關機 (Graceful Shutdown)。
5. **不使用額外網頁 HTTP 框架**：直接使用標準函式庫 `net/http`。
6. **無 DI 框架**：採純 Go 標準庫與手動組裝方式，不在 `main.go` 中引入 fx/wire 等 DI 框架。

---

## 系統架構設計 (基於 DDD 與專案規範)

依據本專案 Go 開發守則，採標準的 **DDD (Domain-Driven Design)** 與 **Clean Architecture** 結構。依賴關係在 `cmd/vault-agent/main.go` 內**手動組裝**，無使用 fx、wire 等 DI 框架：

```text
vault-agent/
├── cmd/
│   └── vault-agent/
│       └── main.go                 # 程式進入點：載入 config、手動組裝各層、啟動 Webhook/Sync Worker Server
├── configs/
│   ├── config.sample.yaml          # 設定檔範例（不含敏感資訊）
│   └── .env.example                # 環境變數範例
├── internal/
│   ├── configs/                    # 設定載入模組 (viper + godotenv)
│   ├── syncer/                     # Webhook 與 同步 業務服務 (Bounded Context)
│   │   ├── domain/                 # 領域層：SecretFetcher 介面、SecretRef、Domain Errors
│   │   ├── application/            # 應用層：MutateUseCase、SyncWorkerUseCase
│   │   ├── infra/                  # 實作層：Vault Client、AWS Client、K8s Repository
│   │   └── delivery/               # 介面層：Webhook HTTP Handler (`/mutate`)
│   └── infra/                      # 全域基礎設施
│       ├── logger/                 # zlogger 日誌封裝配置
│       ├── telemetry/              # OTLP Traces 追蹤配置
│       └── metrics/                # Prometheus 指標收集器
├── deployments/
│   └── kustomize/
│       ├── base/                   # deployment.yaml, service.yaml, rbac.yaml, webhook.yaml
│       └── overlays/               # 環境覆寫 (如 prod)
├── docs/                           # 文件相關
├── README.md                       # 專案說明文件
├── .gitignore                      # Git 忽略檔案清單
├── Dockerfile                      # Go Multi-stage build (golang:1.25 + distroless)
├── Makefile                        # 常用指令 (make tidy, lint, test 等)
├── go.mod
└── go.sum
```

### 關鍵技術選型

- **HTTP Server**: `net/http`。
- **Kubernetes Client**: `k8s.io/client-go`。
- **機密來源 SDK**: `github.com/hashicorp/vault/api`、`github.com/aws/aws-sdk-go-v2/service/secretsmanager`。
- **設定檔管理**: `github.com/spf13/viper` 與 `github.com/joho/godotenv`。支援優先讀取 OS 環境變數 (供 K8s Deployment 使用) 以及容忍找不到 `config.yaml` 繼續執行的機制。Vault 設定支援 **Token** 或 **Kubernetes 認證**（`vault.auth_k8s_path`、`vault.auth_k8s_role`），二擇一。
- **日誌**: `github.com/vincent119/zlogger`。

---

## 核心流程模組設計

### 1. Delivery 層 (HTTP Webhook & Metrics)

- `/mutate`: K8s AdmissionReview 的進入點。收到請求後，將參數解析並呼叫 Application 層的 UseCase 執行資料拉取與注入，跨層呼叫一律傳遞 `context.Context` 作為首個參數。
- `/metrics`: 提供給 Prometheus 的端點。遵循指標命名規範，如 `http_requests_total`。
- `/healthz`: 健康檢查端點，供 Kubernetes liveness/readiness probes 使用。

### 2. Application 層 (Use Cases)

- **MutateUseCase**: 解析 K8s `AdmissionReview`、從機密後端取資料、建構 JSON Patch 注入 Pod 環境變數。
- **SyncWorkerUseCase**: 定時輪詢、比對差異後更新 K8s Secret。`Run(ctx context.Context)` 監聽 `ctx.Done()` 安全退出。

### 3. Domain 層

- 定義機密提供者抽象如 `SecretFetcher` 介面。
- 定義 `SecretRef` 資料結構 (backend, path, keys)。
- 定義 Domain Errors：`ErrSecretNotFound`、`ErrSecretFetchFailed`。
- 嚴格禁止框架依賴（無 SQL/JSON 操作）。

### 4. Infra 層 (外部系統實作)

- **Vault Client & AWS Client**: 實作 `SecretFetcher` 介面。Client struct 內僅保留連線設定與 `*http.Client`，絕不保存 Request 等可變狀態。Vault 支援兩種認證方式：**靜態 Token**（本機／dev）與 **Kubernetes Auth**（K8s 部署時依 ServiceAccount JWT 向 Vault 取得 token），與 vault-py 的 `VAULT_AUTH_K8S_PATH` / `VAULT_AUTH_K8S_ROLE` 對齊。
- **K8s Repository**: 取代直接操作連線池，封裝 `client-go` 針對 Kubernetes Secret / ConfigMap資源的操作介面。

---

## 優雅關機與生命週期 (Graceful Shutdown)

遵循規範，實作優雅關機流程（Shutdown timeout 30 秒）：

- `cmd/vault-agent/main.go` 使用 [`github.com/vincent119/commons/graceful`](https://github.com/vincent119/commons/tree/main/graceful) 統一管理：
  - 監聽 `SIGINT` / `SIGTERM` 並建立 signal context
  - 主要任務（HTTP Server + Sync Worker）在 `ctx.Done()` 時結束並進入 cleanup
  - cleanup（LIFO）中呼叫 `http.Server.Shutdown(shutdownCtx)` 停止接受新請求並 Drain 既有請求
  - Sync Worker 透過同一個 `ctx` 收到取消訊號後自然退出
  - 統一回收與關閉外部資源（如 Tracer shutdown）

---

## 防護機制 (防止 Webhook 拖垮 K8s 叢集)

- **failurePolicy: Ignore**: `MutatingWebhookConfiguration` 設為忽略，確保 agent 出錯時不會卡死無關的 Pod 部署。
- **objectSelector**: 透過標籤過濾 (如 `inject-vault-agent: "true"`)，大幅降低 Webhook 不必要的請求量。

---

## 增強可觀測性與維運品質

本節對應目標 #4，涵蓋監控、追蹤、日誌、健康檢查與優雅關機，便於維運與故障排除。

### 監控指標 (Prometheus)

- **端點**: `GET /metrics`，供 Prometheus scrape。
- **開關**: 設定檔 `metrics.enabled`（或環境變數）控制是否掛載 `/metrics`；未啟用時不暴露端點。
- **可選 Basic Auth**: `metrics.basic_auth` 為非空字串（格式 `user:password`）時，對 `/metrics` 套用 Basic 認證，避免未授權存取。
- **關鍵指標**（定義於 `internal/infra/metrics/`）：
  - `mutate_requests_total`：Webhook 請求計數（label: status=success|error）
  - `secret_fetch_duration_seconds`：從後端拉取機密的耗時直方圖（label: backend=vault|aws）
  - `sync_errors_total`：Sync Worker 同步失敗計數
- **Grafana 儀表板**: 預設範本位於 `docs/Grafana/vault-agent.json`，可匯入 Grafana 做視覺化。

### 分散式追蹤 (OTLP Tracing)

- **開關**: `telemetry.enabled` 為 true 時才初始化 Tracer 並在關機時執行 shutdown。
- **傳輸**: `telemetry.otlp_transport` 可選 **gRPC**（預設，port 4317）或 **HTTP**（port 4318），對應 [OpenTelemetry Collector configgrpc](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc)。
- **Exporter**: 支援 stdout（`otlp_endpoint` 為空）、gRPC、HTTP；gRPC/HTTP 皆支援 **壓縮**（`telemetry.otlp_compression`: `none` | `gzip`）、**認證 header**（`telemetry.otlp_headers`）與 **Basic Auth**（`telemetry.otlp_basic_auth`: `user:password`，對應 [configauth](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configauth) Client Basic Auth）。
- **實作**: `internal/infra/telemetry/` 封裝 OpenTelemetry 初始化；跨層傳遞 `context.Context` 以承載 span。

### 結構化日誌 (zlogger)

- **統一輸出**: 全域使用 `github.com/vincent119/zlogger`，由 `internal/infra/logger` 依設定初始化。
- **設定驅動**: `configs` 的 `log` 區塊（level、format、outputs、file 等）經 `logger.InitLogger(cfg.App.Env, &cfg.Log)` 轉成 zlogger 設定；`app.env=prod` 時強制 JSON、關閉 development 與 color。
- **脈絡欄位**: Delivery 層為每個 Webhook 請求產生 `request_id` 並寫入 `X-Request-ID`，透過 `zlogger.WithRequestID` / `WithComponent` / `WithOperation` 注入 context，Application 與 Infra 層使用 `zlogger.*Context` 輸出，便於依 request_id 串起單一請求的日誌。
- **維運友善**: 關鍵初始化路徑（config 載入、Vault/K8s/AWS 啟用狀態、Sync Worker 開關）在 `main.go` 以 `zlogger.Info` 記錄摘要，不輸出敏感資訊（如 token）。

### 健康檢查

- **端點**: `GET /healthz` 回傳 200，供 Kubernetes liveness / readiness probe 使用。

### 優雅關機 (Graceful Shutdown)

- 詳見上方章節 **優雅關機與生命週期**：使用 `github.com/vincent119/commons/graceful`，30 秒 shutdown timeout，LIFO cleanup（HTTP Server drain、Tracer shutdown），Sync Worker 隨 `ctx.Done()` 安全退出。

---

## 文件相關

- 說明文件採取，中英文雙語，中文在前英文在後

## 實作現況

本計畫已依 `task.md` 完成 Phase 1～7 及 Phase 8 大部分項目，核心功能均已實作。

**待完成項目** (Phase 8 剩餘)：

- 撰寫 `deployments/kustomize/overlays/prod/` 生產環境覆寫設定
- 補充文件：API 說明、Annotation 格式、安裝指引於 `docs/` 或 `README.md`

**已完成（本輪補充）**：

- Vault **Kubernetes 認證**：Config 新增 `auth_k8s_path`、`auth_k8s_role`；Infra 新增 `NewVaultClientWithK8sAuth`；`main` 依設定擇一使用 Token 或 K8s Auth，與 vault-py 部署方式一致。
