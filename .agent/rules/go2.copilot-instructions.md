---
trigger: always_on
---

#### 健康檢查
- **必須**實作 [gRPC Health Checking Protocol](https://github.com/grpc/grpc/blob/master/doc/health-checking.md)
- 搭配 Kubernetes liveness/readiness probes 使用 `grpc_health_probe`

#### Deadline 與 Context
- Server 端必須尊重 client 傳入的 deadline
- 長時間操作需定期檢查 `ctx.Done()`
- 禁止忽略 context cancellation

#### 錯誤代碼對應
| Domain Error | gRPC Status |
|--------------|-------------|
| NotFound | `codes.NotFound` |
| ValidationError | `codes.InvalidArgument` |
| Unauthorized | `codes.Unauthenticated` |
| Forbidden | `codes.PermissionDenied` |
| Conflict | `codes.AlreadyExists` |
| Internal | `codes.Internal` |

#### GracefulStop 範例
```go
// 收到關機訊號後
grpcServer.GracefulStop() // 等待進行中請求完成
// 若超時則強制停止
// grpcServer.Stop()
```

### 日誌與可觀測性
- 使用**結構化日誌**（如 `zap`）；固定欄位：`trace_id`, `span_id`, `req_id`, `subsystem`。
- `logger.Error("msg", zap.Error(err))` 報告；避免把錯誤訊息再字串化拼接。
- 指標/追蹤採 OpenTelemetry；HTTP/DB 客戶端優先用已 instrument 的實作。
- **Context 傳遞**：所有跨函式呼叫（特別是跨邊界的 DB/HTTP 呼叫）必須傳遞 `ctx`，以確保 Trace ID 能正確串接。

#### Prometheus Metrics 規範
- **核心原則**：
  - **Counter**：**僅能增長 (Increment)**，不可減少。適用於：請求總數、錯誤總數、任務完成次數。
  - **Gauge**：可增減。適用於：當前記憶體用量、Goroutine 數量、Queue 長度。
  - **Histogram**：數值分佈統計。適用於：請求延遲 (Latency)、Payload 大小。
- **命名慣例**：
  - 使用蛇形命名法 (Snake Case)：`http_requests_total`
  - **必須**包含單位後綴：`_seconds` (延遲), `_bytes` (大小), `_total` (計數)
- **Label 規範**：
  - **禁止**高基數 (High Cardinality) 值（如 `user_id`, `email`, `trace_id`），避免搞垮 Prometheus。
  - 必備 Label：`service` (服務名), `env` (環境), `code` (錯誤碼/狀態碼)。
- **程式碼範例**：
  ```go
  // Counter: 僅能 Inc
  requestsTotal.WithLabelValues("200", "GET").Inc()

  // Histogram: 觀測耗時
  timer := prometheus.NewTimer(requestDuration)
  defer timer.ObserveDuration()
  ```

### 時間與時區
- **內部以 UTC 儲存與運算**；輸出呈現再格式化。
- JSON 時間使用 RFC3339（必要時 `time.RFC3339Nano`）。

### 安全性
- 僅用標準 `crypto/*`；禁自製密碼學。
- 外部輸入需驗證與長度限制；避免正則 ReDoS。
- 檔案 I/O 使用 `fs.FS` 與限制型讀取；防 Zip Slip。
- 納入 `gosec`（或等價 analyzer）於 CI；敏感資訊不得進日誌。

### 依賴與模組
- 模組遵循 **SemVer**；破壞性改動於 major path（`/v2`）。
- 嚴格釘版：`go.mod` 使用最小相依原則；避免 transitive 泄漏。
- 移除依賴需跑 `go mod tidy` 並附影響說明。

### Database Migration（資料庫遷移）

#### 工具選擇
- 推薦 [`golang-migrate/migrate`](https://github.com/golang-migrate/migrate) 或 [`pressly/goose`](https://github.com/pressly/goose)
- 選擇後**全專案統一**，禁止混用

#### 命名慣例
```
migrations/
├── 20260108120000_create_users_table.up.sql
├── 20260108120000_create_users_table.down.sql
├── 20260108130000_add_email_index.up.sql
└── 20260108130000_add_email_index.down.sql
```
- 格式：`YYYYMMDDHHMMSS_<description>.<up|down>.sql`
- 描述使用**蛇形命名法**（snake_case）

#### 版本控制
- Migration 檔案**必須**納入 Git
- **禁止**修改已執行的 migration（新增新檔案修正）
- 復原（down）**必須**與 up 對應，確保可回退

#### CI/CD 整合
- Migration 應在**應用啟動前**執行（init container 或 pre-deploy hook）
- 禁止在應用程式 `main()` 中執行 migration（避免多副本競爭）

#### 最佳實務
- 大型表變更使用 **pt-online-schema-change** 或 **gh-ost**（避免鎖表）
- 新增 NOT NULL 欄位需先加入預設值，再移除預設值

### 依賴注入 (Dependency Injection)
- **Infrastructure 層** (如 Database, Cache) 與 **Application 層** (Use Cases) 的依賴關係需透過 DI 容器組裝。
- 建議在 `cmd/` 或 `internal/<service>/di.go` 中統一管理依賴。
- **禁止**在業務邏輯層中手動 `New` 具體的 Infrastructure 實作。
- 本專案 DI 推薦 fx；若採 wire，需提供產生器與 CI 產生檔一致性規範（go generate / wire_gen.go）。

#### DI 測試情境

##### Interface 設計原則
- Repository / Service 皆以 **interface** 暴露；實作為 private struct
- interface 定義於 Domain 或 Application 層，實作於 Infra 層

##### Mock 生成
- 推薦 [`uber-go/mock`](https://github.com/uber-go/mock)（原 gomock）或 [`mockery`](https://github.com/vektra/mockery)
- 統一 Mock 檔案置於 `internal/mocks/` 或 `mocks/` 套件中
- **必須**使用 `go generate` 自動生成，指令範例：
  ```go
  //go:generate mockgen -source=repository.go -destination=../../mocks/repository_mock.go -package=mocks
  type Repository interface { ... }
  ```

##### 測試範例
```go
// 使用 mock 測試 UseCase
func TestCreateOrder(t *testing.T) {
    // Arrange
    mockRepo := mocks.NewMockOrderRepository(t)
    mockRepo.EXPECT().
        Save(mock.Anything, mock.AnythingOfType("*domain.Order")).
        Return(nil)

    uc := application.NewCreateOrderUseCase(mockRepo)

    // Act
    err := uc.Execute(t.Context(), input)

    // Assert
    require.NoError(t, err)
}
```

##### fx 測試模式
```go
func TestIntegration(t *testing.T) {
    app := fxtest.New(t,
        fx.Provide(NewTestDB),       // 測試用 DB
        fx.Provide(NewOrderRepo),
        fx.Provide(NewCreateOrderUseCase),
        fx.Invoke(func(uc *CreateOrderUseCase) {
            // 執行測試
        }),
    )
    app.RequireStart()
    defer app.RequireStop()
}
```

### Configuration（設定管理）

#### 優先順序（由高至低）
1. **環境變數**（`APP_DATABASE_HOST`）
2. **設定檔**（`config.yaml`）
3. **預設值**（程式碼內建）

#### 結構化配置範例
```go
type Config struct {
    Server   ServerConfig   `mapstructure:"server"`
    Database DatabaseConfig `mapstructure:"database"`
    Redis    RedisConfig    `mapstructure:"redis"`
}

func LoadConfig() (*Config, error) {
    viper.SetConfigName("config")
    viper.AddConfigPath("./configs")
    viper.AutomaticEnv()
    viper.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

    if err := viper.ReadInConfig(); err != nil {
        return nil, fmt.Errorf("read config: %w", err)
    }

    var cfg Config
    if err := viper.Unmarshal(&cfg); err != nil {
        return nil, fmt.Errorf("unmarshal config: %w", err)
    }
    return &cfg, nil
}
```

#### 敏感資訊處理
- **禁止**將 secrets（密碼、API Key）放入設定檔或程式碼
- 使用環境變數或外部 Secret Manager：
  - HashiCorp Vault
  - Infisical
  - Kubernetes Secrets（搭配 External Secrets Operator）

#### 啟動驗證
- 必要設定缺失時，應於啟動階段 `log.Fatal` 終止
- 使用 `validator` 套件驗證結構欄位

#### 範例檔案
- 提供 `config.sample.yaml`（不含真實資料）
- `.env.example` 列出所有環境變數與說明

### 目錄結構
```bash
.
├── cmd/
│   └── <app>/main.go          # 進入點：只負責載入 configs、初始化 internal/infra (DB/Redis) 並啟動依賴注入
├── api/                       # 對外契約：OpenAPI (Swagger)、Protobuf 定義與產生的程式碼
├── configs/                   # 設定檔：config.yaml, env.example
├── internal/                  # 核心邏輯：外部無法 import
│   ├── <service>/             # 業務服務【Bounded Context】
│   │   ├── domain/            # 領域層：Entity, VO, Repository Interface (純業務，禁 SQL/JSON)
│   │   ├── application/       # 應用層：Use Case 流程編排，依賴 domain interface
│   │   ├── infra/             # 實作層：Repository Impl (SQL/GORM), JWT 實作, 外部 API 呼叫
│   │   ├── delivery/          # 介面層：HTTP Handlers, gRPC Services, DTO 定義
│   │   └── di.go              # 依賴注入 (Google Wire or Fx)
│   ├── infra/                 # 【全域基礎設施】
│   │   ├── database/          # DB 連線池初始化 (MySQL, Postgres)
│   │   ├── cache/             # Redis, Memcached 客戶端
│   │   └── logger/            # 結構化日誌 (Zap/Slog) 配置
│   └── pkg/                   # 【Shared Kernel】跨 Bounded Context 的共用領域抽象（例：Money、DomainError
│                              # 僅限「跨 Bounded Context 皆成立」的抽象
│                              # 禁止放入特定 service 的規則或流程
├── pkg/                       # 【通用工具】uuid, retry, stringutils (完全不含業務邏輯)
├── migrations/                # Database Migration 檔案（詳見 Migration 規範）
├── scripts/                   # 腳本：DB Migration, Makefile 輔助腳本
├── deployments/               # Kubernetes、Helm Chart 與部署相關檔案
│   ├── helm/                  # Helm charts
│   └── kustomization/         # Kustomize overlays
├── docs/                      # swagger.yaml, 架構設計文件
├── documents/                 # 專案文件
│   ├──  en/                   # 專案相關文件（需求規格、設計文件、SOP）
│   └──  zh/                   # 專案相關文件（需求規格、設計文件、SOP）
├── test/                      # 整合測試與測試資料 (fixtures) 黑箱 / 整合測試，禁止直接測 domain 私有狀態
├── Dockerfile                 # Multi-stage build
├── Makefile                   # 常用指令 (make wire, make test, make lint)
├── .dockerignore              # Docker build 忽略清單（排除編譯輸出、暫存檔與測試資料）
├── .gitignore                 # Git 忽略清單（node_modules、vendor、log、tmp 等）
├── .golangci.yml              # 靜態分析與 Linter 設定（統一風格與品質門檻）
├── README.md                  # 專案說明：目的、架構、建置步驟、測試與部署指引
├── LICENSE                    # 授權條款；內部專案可標註版權與使用限制
├── go.mod                     # Go 模組定義與依賴版本管理
└── go.sum                     # 依賴模組驗證雜湊清單（確保可重建性）
```

### 架構總覽（Architecture Overview）

#### 本專案採用 領域驅動設計（Domain-Driven Design, DDD） 作為核心架構方法，並僅在最外層以 MVC 作為介面實作模式，兩者責任邊界清楚、互不混用。
- 每一項業務能力皆建模為一個獨立的 限界上下文（Bounded Context），並置於 internal/<service>/ 目錄下。
- 業務規則集中於 領域層（Domain Layer），與任何框架或基礎設施實作完全解耦。
- MVC 僅應用於 交付層（Delivery Layer），用於處理對外介面（HTTP / gRPC）。
- 基礎設施相關關注點（資料庫、快取、日誌等）皆透過 依賴注入（Dependency Injection） 進行解耦與提供。
- MVC 僅作為 Delivery Layer 的實作模式之一，不構成系統核心架構。

#### 此架構能確保系統具備長期可維護性、可測試性，並可在不破壞核心業務模型的前提下，平順演進自單體架構至微服務架構。

### Shared Kernel（共用核心）使用規範

> Shared Kernel 位於 `internal/pkg/`，存放跨 Bounded Context 通用的領域抽象。

#### ✅ 適合放入的內容
| 類型 | 範例 |
|------|------|
| Value Objects | `Money`, `Email`, `PhoneNumber`, `Address` |
| Domain Error | `DomainError`, `ValidationError`, `NotFoundError` |
| 通用介面 | `Clock`, `UUIDGenerator`（用於測試注入） |
| 規格抽象 | `Specification<T>` pattern 基底 |

#### ❌ 禁止放入的內容
| 類型 | 原因 |
|------|------|
| 特定 BC 的 Entity/Aggregate | 造成 BC 間耦合 |
| 業務流程編排（Use Case） | 違反 BC 邊界獨立性 |
| 框架耦合的實作（如 GORM Model） | 應放 Infra 層 |
| 可變狀態或 Singleton | 難以測試與併行安全 |

#### 變更流程
1. 修改 Shared Kernel 需所有**相依 BC 負責人同意**
2. 變更需附**影響範圍分析**（列出受影響的 BC）
3. **向下相容**變更可直接合併；破壞性變更需升版並遷移計畫

#### 設計原則
- **最小化**：只放真正跨 BC 通用的抽象
- **不可變**：Value Objects 設計為 immutable
- **無副作用**：Shared Kernel 內的邏輯不應有 I/O 或外部依賴

### 設定檔與環境變數
- 使用 `spf13/viper` 管理設定。
- **優先級**：環境變數 (ENV) > 設定檔 (yaml/json) > 預設值。
- 結構定義範例：
  ```go
  type Config struct {
      Server   ServerConfig   `mapstructure:"server"`
      Database DatabaseConfig `mapstructure:"database"`
  }
  ```

### 產生器與 build
- 使用 `//go:build` 標籤管理條件編譯；禁止舊 `+build` 註解。
- 建議使用 **Config/Env** 控制環境行為（12-Factor App 原則），避免 build tags 導致 binary 不一致：
  - 開發環境：開啟詳細錯誤堆疊、Swagger UI（透過 `APP_ENV=dev`）
  - 生產環境：僅輸出 JSON 日誌、關閉 Debug 路由（透過 `APP_ENV=prod`）
- `go generate` 指令須在檔頭註解，並可重複執行（可重入）。
- CGO 預設關閉；開啟需 PR 說明平台/效能/部署影響。

---

## Copilot / Agent 提示模板（Do/Don't）

**Do**
- 僅產生一個 `package` 宣告；imports 分群。
- 立即檢查 `err`，使用 `%w` 包裝。
- 所有公開 API 第一參數 `context.Context`。
- 在邊界複製 slice/map；為 struct 加 `json`/`yaml` tags。
- 撰寫 table-driven 測試 + `t.Helper()`，並加入一個基準測試。
- 撰寫 table-driven 測試 + `t.Helper()`，並加入一個基準測試。

**Don’t**
- 不要保存 `context.Context` 或 `*http.Request` 於 struct。
- 不要在 library 使用 `panic`；不要忽略 `Close()`。
- 不要以 `interface{}` 取代具體型別；不要暴露可變 slice/map。
- 不要在長迴圈內直接 `defer` 造成延後釋放與資源累積；必要時以匿名函式縮小 scope，或顯式 `Close()`。

---
