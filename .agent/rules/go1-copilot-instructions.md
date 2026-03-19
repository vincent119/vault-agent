---
trigger: always_on
---

# Go 開發與 Copilot / Agent 指南（整合 Uber Guide 版本）

本檔延伸自 `.github/standards/copilot-common.md` 與 `.github/standards/copilot-vocabulary.yaml`，
並 **參照**：

- [Uber Go 風格指南（繁體中文維護版）](https://github.com/ianchen0119/uber_go_guide_tw)
- [Effective Go（官方）](https://go.dev/doc/effective_go#introduction)
- Go 版本目標：**Go 1.25+**（如需更動版本，PR 必述影響）
- Go 版本策略：跟隨 Go 官方 stable（半年一版），升版前需完成 CI、-race、關鍵路徑壓測與相容性驗證。

> 目標：統一格式、用詞與安全實務，確保自動產生或人工撰寫的程式碼皆符合 idiomatic Go，並可直接編譯、部署與維護。

---
## 目錄
- [Copilot / Agent 產生守則](#copilot--agent-產生守則)
- [Go 一般開發規範](#go-一般開發規範整合-uber--effective-go)
- [Domain Events](#domain-events領域事件)
- [優雅關機](#優雅關機graceful-shutdown)
- [目錄結構](#目錄結構)
- [CI 與工具建議](#ci-與工具建議可直接採用)


## Copilot / Agent 產生守則

### 檔案與 package 規範
- 每檔僅 **一行** `package <name>` 宣告（置頂）。
  - 編輯檔案：保留原 package。
  - 新檔案：與資料夾既有 `.go` 同名 package。
- 可執行程式置於 `cmd/<app>/main.go`，library 不得含 `main()`。
- package 名稱：**全小寫、單字、無底線**（Uber）。

### Imports 與工具
- 產出前必可通過 `gofmt -s`（建議 `gofumpt`）、`goimports`、`go vet`。
- 自動清除未用 imports，避免循環依賴。
- 變更 `go.mod` 後提示 `go mod tidy`。
- 縮排：Tab；檔尾留 **單一** 換行；UTF-8（無 BOM）。
- Imports 排序：**標準庫 → 第三方 → 專案內部**；群組以空行分隔。

### 錯誤處理與流程
- 呼叫後**立即**檢查 `err`，採 **early return**。
- 包裝錯誤：`fmt.Errorf("context: %w", err)`；跨層使用 `errors.Is/As`。
- 多重錯誤聚合使用 `errors.Join`（如：驗證列表或 defer Close）。
- 訊息小寫開頭，尾端**不加標點**。
- 僅在**不可恢復初始化**時用 `panic`；避免在 library 使用。
- 禁止「只記錄不回傳」導致錯誤吞沒；**記錄與回傳擇一**，以邏輯層級決定。

### 函式設計
- 函式應該簡短且專注於單一任務（建議不超過 50 行）
- 參數數量盡量控制在 3-4 個以內
- 使用具名回傳值提高可讀性（但避免 naked return）
- `context.Context` 應該作為第一個參數
- `error` 應該作為最後一個回傳值
- 避免 `bool` 參數（改用具名 Option 或拆分函式）
- 多參數時採用 **Functional Options Pattern**（`opts ...Option`）提升擴展性

```go
// ✅ 正確：context 第一、error 最後、使用 Functional Options
func ProcessData(ctx context.Context, data []byte, opts ...Option) (Result, error)

// ✅ Functional Options Pattern 範例
type Option func(*config)

func WithTimeout(d time.Duration) Option {
    return func(c *config) { c.timeout = d }
}

func WithRetries(n int) Option {
    return func(c *config) { c.retries = n }
}

// ❌ 錯誤：bool 參數、context 不在第一個
func ProcessData(data []byte, timeout int, retries int, debug bool, ctx context.Context) (Result, error)
```

### 並行與 I/O 安全 區塊
- 每個 goroutine 需有**退出機制**（`context`、`WaitGroup` 或關閉 channel）。
- Channel 緩衝預設 0 或 1（除非有量測證據）。
- 嚴禁 goroutine 泄漏；資源關閉要落在呼叫點 `defer Close()`。
- 不可重用已讀取的 `req.Body`；需 **clone**：
  ```go
  // 將來源位元組切片拷貝，確保可重播 Body
  buf := bytes.Clone(src)
  req.Body = io.NopCloser(bytes.NewReader(buf))
  req.GetBody = func() (io.ReadCloser, error) {
      return io.NopCloser(bytes.NewReader(buf)), nil
  }
  ```
- `io.Pipe`/multipart 必須**單執行緒順序寫入**；失敗用 `pw.CloseWithError(err)`、成功 `mw.Close()` 再 `pw.Close()`。
- 底層 slice/map 在**邊界（入/出）**時一律複製，避免別名共享。

### HTTP Client 設計
- `Client` 僅存設定（BaseURL、`*http.Client`、headers）；**不得**保存 `*http.Request` 或可變請求狀態。
- 方法介面：
  - 皆接收 `context.Context`。
  - 內部建 `http.Request` → `c.httpClient.Do(req)` → `defer resp.Body.Close()`。
  - 要求**逾時/重試/回退**策略明確，並遵循「Net/HTTP 實務」章節。

### JSON / Struct Tag
- 對外型別欄位加上 `json,yaml,mapstructure` tags；**選填**欄位 `omitempty`。
- 輸入端（decode）預設**拒絕未知欄位**：
  ```go
  dec := json.NewDecoder(r)
  dec.DisallowUnknownFields()
  ```
- 使用 `any` 取代 `interface{}`；但優先具體型別。
- 時間欄位採 **RFC3339**（UTC 優先）；必要時標注本地時區偏差。

### 測試與範例
- 採 **table-driven tests**；子測試用 `t.Run`。
- 使用 `t.Context()` 獲取自動管理的 context （Go 1.24+；本專案目標 Go 1.25+ 故可直接採用）。
- 輔助函式 `t.Helper()`；清理用 `t.Cleanup()`。
- 匯出 API 提供 `example_test.go`。
- 優先標準 `testing`；除非必要不引入 assert 套件。
- **Mocking 策略**：使用 `uber-go/mock` (原 gomock) 針對 interface 生成 mock，統一置於 `internal/mocks` 或同層 `mocks` 套件。
- 需通過：`-race`、單元涵蓋率門檻（預設 80% 可調整；變更需 PR 說明）。
- 提供**基準**與**模糊測試（fuzz）**於關鍵路徑。

### 產出內容要求
- 輸出 **完整可編譯檔案**或明確 **diff**。
- 多檔變更列出：檔名 / 變更摘要 / 風險。
- 新增外部套件需附：`go get <module>@<version>` 與風險評估。

### 詞彙與術語
- 優先 `.github/standards/copilot-vocabulary.yaml`。
- 與現有命名衝突時以 vocabulary 為準。
- 與 Uber/Effective Go 不一致時，PR **必述理由**與替代方案。

---

## Go 一般開發規範（整合 Uber + Effective Go）

### 通用原則
- 清晰優於巧妙；主流程靠左排列；讓 **零值可用**。
- 結構自我說明；註解描述「**為何**」而非「做什麼」。

### 命名慣例
- package：全小寫、單字、無底線；避免 `util`、`common`。
- 變數/函式：小駝峰；匯出名稱首字母大寫。
- 介面以 `-er` 結尾（Reader/Writer）；**小介面**優先。
- 縮略詞大小寫一致：`HTTPServer`、`URLParser`。
- 建構子命名採 `NewType(...)`；必要時 `WithXxx` 選項，但避免過度抽象。
- 常數使用駝峰式（匯出：`MaxRetryCount`；私有：`maxRetryCount`），**禁用全大寫底線**。

### 常數與列舉
- 群組 `const (...)`；**型別化常數**避免魔數。
- Enum 起始值**考慮零值可用性**，必要時保留 `Unknown`。

### 接收者與方法
- 以量測決定**指標/值**接收者（大型結構/需變異 → 指標；小值/不變 → 值）。
- 避免 `init()` 副作用與全域可變狀態。
- 針對可能回傳大量數據的列表方法，優先使用 **Iterators** (`func(yield func(T) bool)`) (Go 1.23+) 取代 Slice 回傳。

```go
// Iterator 範例：避免一次載入全部資料
func (s *Set[T]) Iterator() iter.Seq[T] {
    return func(yield func(T) bool) {
        for _, v := range s.values {
            if !yield(v) {
                return
            }
        }
    }
}
```

### `context` 規範
- 對外 API **第一個參數**為 `ctx context.Context`。
- 禁用 `context.Background()` 直傳至深層；由呼叫者注入。
- 設定逾時/截止於**呼叫邊界**；尊重 `ctx.Done()`。
- 不將 `ctx` 保存於結構體。

### 並行進階
- 以 `errgroup`/`WaitGroup` + `ctx` 收斂；提供**背壓**與**取消**。
- 共享狀態以 `sync.Mutex/RWMutex` 或無鎖結構（經量測）保護。

### Domain Events（領域事件）

#### 定義規範
- Event 為**不可變 struct**，包含：
  - `EventID`：事件唯一識別碼（UUID）
  - `OccurredAt`：事件發生時間（UTC, RFC3339）
  - `AggregateID`：所屬聚合根 ID
  - `EventType`：事件類型字串（格式：`<Aggregate>.<PastTenseVerb>`）
- 事件命名使用**過去式動詞**：`OrderCreated`、`PaymentCompleted`、`UserEmailChanged`

#### 事件結構範例
```go
// DomainEvent 為所有領域事件的基底結構
type DomainEvent struct {
    EventID     string    `json:"eventId"`
    EventType   string    `json:"eventType"`
    AggregateID string    `json:"aggregateId"`
    OccurredAt  time.Time `json:"occurredAt"`
}

// OrderCreated 訂單建立事件
type OrderCreated struct {
    DomainEvent
    CustomerID string  `json:"customerId"`
    TotalAmount float64 `json:"totalAmount"`
}
```

#### 發布模式
| 模式 | 適用場景 | 實作方式 |
|------|----------|----------|
| 同步 | 同一 Bounded Context 內 | Aggregate Root 回傳 `[]DomainEvent` |
| 非同步 | 跨 BC / 外部系統 | Message Queue（NATS、Kafka、RabbitMQ） |

#### Outbox Pattern（推薦）
- **目的**：確保事件與狀態在同一交易中一致
- **流程**：
  1. 業務操作與事件寫入 `outbox` 表於同一 DB Transaction
  2. 背景 Worker 輪詢 `outbox` 表並發佈至 Message Queue
  3. 發佈成功後標記或刪除該筆紀錄
- **優點**：避免分散式交易，保證最終一致性

#### 冪等處理
- Consumer **必須**處理重複事件（網路重試、At-Least-Once 語意）
- 使用 `EventID` 進行去重，可搭配 Redis SET 或資料庫唯一約束
- 設計事件處理邏輯時，確保多次執行結果一致

### 優雅關機（Graceful Shutdown）

- 所有 server、background worker、consumer **必須實作優雅關機**。
- 必須監聽 `SIGINT`、`SIGTERM`，並轉換為 `context.Context` 的取消事件。
- 關機時流程必須遵循以下順序：

  1. 接收系統訊號（`signal.NotifyContext`）
  2. 停止接受新請求（HTTP Server `Shutdown` / gRPC `GracefulStop`）
  3. 等待進行中的請求或任務完成
  4. 在 timeout 到期後強制結束
  5. 關閉所有外部資源（DB、Cache、Queue、Tracer）

- 所有 goroutine 必須能回應 `ctx.Done()` 並自行結束。
- 不得在 goroutine 中忽略取消訊號造成關機卡死。
- 禁止在正常關機流程中使用 `os.Exit()`。
- Kubernetes 環境需搭配 `terminationGracePeriodSeconds` 與 `preStop` hook，
  確保應用層 Shutdown timeout 與 Pod 終止行為一致。
- background worker / queue consumer 必須在收到 ctx.Done() 後停止拉取新任務，
  並完成當前任務或在 timeout 後中止。
- 長時間服務建議提供關機路徑測試（模擬 ctx cancel / SIGTERM）。
- 禁止在 server goroutine 使用 `log.Fatal` / `zlogger.Fatal`（會跳過 defer 與資源收尾）；改為回傳錯誤到 srvErr，由主流程統一處理。
- Shutdown timeout 必須與部署環境一致：
  - Kubernetes：`terminationGracePeriodSeconds >= shutdownTimeout + buffer`（建議 buffer 5~10 秒）。
  - 關機順序固定：Stop accept → Drain → Stop workers/consumers → Close external resources。
- 關機路徑必須可測：至少提供一個「可注入 cancel」的測試入口（例如把 `runHTTPServer` 做成可測函式）。

#### Kubernetes preStop hook 建議
```yaml
# 確保 Pod 從 Service Endpoints 移除後再開始 shutdown
lifecycle:
  preStop:
    exec:
      command: ["sleep", "5"]  # 等待從 endpoints 移除
```

#### HTTP Server 建議實作模式

> 需 Go 1.16+（`signal.NotifyContext`）；本專案目標 Go 1.25+ 故可直接採用。

```go
// 建議：以 signal.NotifyContext 將 OS 訊號轉為可取消的 context，並同時監控 server 異常退出。
// 特性：不論是 SIGTERM/SIGINT 或 ListenAndServe 異常，都會走同一條 graceful shutdown 流程。
func runHTTPServer(
	srv *http.Server,
	shutdownTimeout time.Duration,
	closeResources func(ctx context.Context) error, // 關閉 scheduler/redis/db 等資源（可為 nil）
	logger *zap.Logger,
) error {
	// 1) 訊號轉 ctx
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	// 2) 監控 server 是否異常退出（避免只等訊號，卻漏掉 server 先掛）
	srvErr := make(chan error, 1)

	go func() {
		// ListenAndServe 正常因 Shutdown/Close 退出會回傳 http.ErrServerClosed
		if err := srv.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			srvErr <- err
		}
		close(srvErr) // 關閉 channel 表示 server goroutine 已結束
	}()

	// 3) 等待：訊號 or server 異常
	select {
	case <-ctx.Done(): // 收到關機訊號
	case err := <-srvErr: // 服務異常退出
		if err != nil {
			logger.Error("http server stopped unexpectedly", zap.Error(err))
		}
	}

	// 4) 統一走 graceful shutdown：先停止接新請求，再收尾資源
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()

	if err := srv.Shutdown(shutdownCtx); err != nil {
		// Shutdown 會等待 in-flight request，若卡住要有最後手段
		logger.Error("http server shutdown failed", zap.Error(err))
		_ = srv.Close() // 最後手段：避免卡住（可能中斷連線）
	}

	// 5) 關閉外部資源（scheduler/worker/redis/db...）
	// 注意：若底層 Close() 不吃 context，這裡的 ctx 僅用於「放棄等待」，不能真的中斷卡死的 Close。
	var resErr error
	if closeResources != nil {
		resErr = closeResources(shutdownCtx)
		if resErr != nil {
			logger.Error("close resources failed", zap.Error(resErr))
		}
	}

	logger.Info("server exited")
	return resErr
}
```

> gRPC GracefulStop 範例請參見 [gRPC 規範章節](#grpc-規範)。

### Net/HTTP 實務
- **重用 Transport**，設定逾時：
  ```go
  tr := &http.Transport{
      MaxIdleConns:        100,
      IdleConnTimeout:     90 * time.Second,
      TLSHandshakeTimeout: 10 * time.Second,
      ExpectContinueTimeout: 1 * time.Second,
  }
  c := &http.Client{
      Transport: tr,
      Timeout:   15 * time.Second, // 全域上限；更細粒度以 context 控制
  }
  ```
- 明確重試策略（**僅**冪等方法），具退避與上限；對 5xx/網路錯誤重試，對業務 4xx 不重試。
- 嚴格 `resp.Body.Close()`；讀取前先檢 HTTP 狀態碼。

### API 設計規範

#### 統一 API 回應結構 (JSON Envelope)
```go
type APIResponse[T any] struct {
    Code    int    `json:"code"`              // 業務碼 (非 HTTP 狀態碼)
    Message string `json:"message"`           // 提示訊息
    Data    T      `json:"data,omitempty"`    // 泛型資料 payload
    TraceID string `json:"trace_id,omitempty"`
}
```

### API Versioning（版本管理）

#### 版本策略
- **URL Path 優先**：使用 `/v1/`, `/v2/` 作為版本前綴
- Header 版本（選用）：僅用於次要版本協商（如 `Accept: application/vnd.api.v1+json`）

#### API 文件 (Swagger)
- `main.go` 需定義全域資訊與驗證方式：
  ```go
  // @title           My API
  // @version         1.0
  // @securityDefinitions.apikey ApiKeyAuth
  // @in header
  // @name Authorization
  ```
- API Handler 需附上 Swagger 註解（`swag init` 生成）
  ```go
  // GetUser 取得使用者資訊
  // @Summary 取得使用者詳情
  // @Tags Users
  // @Produce json
  // @Param id path string true "User ID"
  // @Security ApiKeyAuth
  // @Success 200 {object} UserResponse
  // @Router /users/{id} [get]
  ```

#### 版本升級原則
| 變更類型 | 處理方式 |
|----------|----------|
| 新增欄位（向下相容） | 無需升版 |
| 移除/修改欄位 | Major version bump（`/v2/`） |
| 行為變更 | Major version bump |

#### 維護期規範
- 新 Major 版本上線後，舊版本**至少維護 6 個月**
- 棄用通知：回應 Header 加入 `Deprecation: true` 與 `Sunset: <date>`
- Swagger/OpenAPI 文件需標註各版本狀態（active/deprecated）

### gRPC 規範

#### Proto 檔案管理
- 統一放置於 `api/proto/<service>/`
- 使用 [buf](https://buf.build/) 管理 linting、breaking change detection
- 產生的程式碼放入 `api/gen/go/`（不手動編輯）

#### Interceptor 設計
```go
// 建議的 Interceptor 順序（由外至內）
grpc.ChainUnaryInterceptor(
    recovery.UnaryServerInterceptor(),      // Panic 回復
    otelgrpc.UnaryServerInterceptor(),      // OpenTelemetry tracing
    logging.UnaryServerInterceptor(logger), // 結構化日誌
    auth.UnaryServerInterceptor(),          // 認證
)
```

