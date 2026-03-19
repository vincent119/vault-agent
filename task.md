# Vault-Agent 開發與重構任務清單

本清單依據 `ImplmentPlan.md` 所述的無 DI、純 Go 標準庫的架構進行展開。

- [x] **Phase 1: 專案基礎建設與初始化**
  - [x] 建立基本的資料夾結構 (`cmd/vault-agent/`, `internal/`, `configs/`, `deployments/kustomize/`, `docs/`)
  - [x] 建立 `go.mod` (指定 Go 1.25.6) 與 `go.sum`
  - [x] 建立基礎檔案：`.gitignore`、`Makefile`、`README.md`
  - [x] 建立並配置設定檔模組 (`viper` + `godotenv`)，支援 OS Env 優先與 YAML 作為備援，範例 `.env.example`, `configs/config.sample.yaml`

- [x] **Phase 2: 基礎設施封裝 (Infra Layer - Global)**
  - [x] 整合並配置 `vincent119/zlogger` 日誌 (修正為正確 API `NewDevelopment`/`NewProduction`)
  - [x] 實作 Prometheus HTTP endpoint (`/metrics`) 與基本指標定義 (Counter, Histogram)
  - [x] OTLP Traces 追蹤配置初始化，支援 stdout fallback 與 OTLP gRPC 兩種模式

- [x] **Phase 3: 領域核心與抽象定義 (Domain Layer)**
  - [x] 定義 `SecretFetcher` 介面 (取得遠端機密資料)
  - [x] 定義 Domain Errors (`ErrSecretNotFound`, `ErrSecretFetchFailed`)
  - [x] 定義 `SecretRef` 資料結構 (backend, path, keys)

- [x] **Phase 4: 基礎設施實作 (Infra Layer - Implementation)**
  - [x] Kubernetes Client 初始化 (支援 In-Cluster 與本機 kubeconfig 兩種模式)
  - [x] 實作 HashiCorp Vault 的 `SecretFetcher` Client (KV v2)
  - [x] Vault **Kubernetes 認證**：Config 新增 `vault.auth_k8s_path`、`vault.auth_k8s_role`；Infra 新增 `NewVaultClientWithK8sAuth`；main 依設定擇一使用 Token 或 K8s Auth（與 vault-py 一致）
  - [x] 實作 AWS Secrets Manager 的 `SecretFetcher` Client (JSON 格式解析)
  - [x] Kubernetes Secret 資源讀寫的 Repository 封裝 (Get/Update/ListByLabel)

- [x] **Phase 5: 應用流程實作 (Application Layer - Use Cases)**
  - [x] 實作 `MutateUseCase`: 解析 K8s `AdmissionReview`、從機密後端取資料、建構 JSON Patch 注入 Pod Env
  - [x] 實作 `SyncWorkerUseCase`: 定時輪詢、比對差異後更新 K8s Secret，監聽 `ctx.Done()` 安全退出

- [x] **Phase 6: 交付介面層 (Delivery Layer)**
  - [x] 實作 Webhook HTTP Handler (`/mutate`) 解析與回應，傳遞 context
  - [x] 串聯 `MutateUseCase` 到 handler，失敗時回傳合法的 Allowed:false AdmissionResponse

- [x] **Phase 7: 系統整合與進入點 (`cmd/vault-agent/main.go`)**
  - [x] 載入設定 (Config)，初始化全域 Logger
  - [x] 手動組裝各層 (VaultClient, AWSClient, K8sRepo, MutateUC, SyncWorker, WebhookHandler)
  - [x] 建立 `http.Server` 並掛載路由 (`/mutate`, `/metrics`, `/healthz`)
  - [x] 啟動 Sync Worker goroutine 與 HTTP Server goroutine
  - [x] 透過 `signal.NotifyContext` 實作 SIGTERM/SIGINT 優雅關機 (30s timeout)

- [x] **Phase 8: 部署、建置與文件**
  - [x] 撰寫 Multi-stage `Dockerfile` (golang:1.25 + distroless 最小映像)
  - [x] 建立 `deployments/kustomize/base/` 目錄：`deployment.yaml`, `service.yaml`, `rbac.yaml`, `webhook.yaml`, `kustomization.yaml`
  - [ ] 撰寫 `deployments/kustomize/overlays/prod/` 生產環境覆寫設定
  - [ ] 補充文件：API 說明、Annotation 格式、安裝指引於 `docs/` 或 `README.md`
