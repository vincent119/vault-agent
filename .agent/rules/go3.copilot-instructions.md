---
trigger: always_on
---

## 實作範例片段

> 具體展示關鍵規則如何落地。

```go
// Package client 提供與遠端服務互動的 HTTP 用戶端。
// 零值不可用，請使用 New 建構子建立。
package client

import (
	"context"          // 佈線取消與逾時的標準機制
	"encoding/json"    // 編解碼輸入/輸出
	"errors"           // 錯誤比對
	"fmt"              // 錯誤包裝與格式化
	"io"               // I/O 介面
	"net/http"         // HTTP 基礎
	"time"             // 逾時與回退間隔
)

// ErrNotFound：對應遠端 404 的語意錯誤（sentinel error）。
var ErrNotFound = errors.New("resource not found") // 小寫開頭，不加標點

// Client 僅保存不可變設定與共享 *http.Client；不保存請求狀態。
type Client struct {
	baseURL    string        // 基底位址（不可含尾斜線）
	httpClient *http.Client  // 可注入以便測試與重用 Transport
}

// New 建立可用的 Client；呼叫者可注入自定 *http.Client。
func New(baseURL string, hc *http.Client) *Client {
	if hc == nil {
		hc = &http.Client{Timeout: 15 * time.Second} // 安全預設
	}
	return &Client{baseURL: baseURL, httpClient: hc}
}

// Resource 對外輸出時含有 json 標籤，零值可用。
type Resource struct {
	ID        string    `json:"id"`
	Name      string    `json:"name,omitempty"`
	UpdatedAt time.Time `json:"updatedAt"` // RFC3339 UTC
}

// Get 透過 context 控制逾時/取消，正確關閉 Body 並轉換語意錯誤。
func (c *Client) Get(ctx context.Context, id string) (Resource, error) {
	var out Resource

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+"/v1/resources/"+id, nil)
	if err != nil {
		return out, fmt.Errorf("new request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return out, fmt.Errorf("do request: %w", err)
	}
	defer resp.Body.Close() // 確保釋放連線

	switch resp.StatusCode {
	case http.StatusOK:
		dec := json.NewDecoder(resp.Body)
		dec.DisallowUnknownFields()
		if err := dec.Decode(&out); err != nil {
			return out, fmt.Errorf("decode: %w", err)
		}
		return out, nil
	case http.StatusNotFound:
		// 將 HTTP 狀態轉換為語意錯誤
		io.Copy(io.Discard, resp.Body) // 盡量讀完以便連線重用
		return out, ErrNotFound
	default:
		b, _ := io.ReadAll(io.LimitReader(resp.Body, 64<<10)) // 限流保護
		return out, fmt.Errorf("unexpected status %d: %s", resp.StatusCode, string(b))
	}
}
```

---

## 常用套件堆疊 (Tech Stack)

### Web 框架
- `net/http` - 標準庫，適合簡單 API
- `github.com/gin-gonic/gin` - 高效能 Web 框架

### 資料庫
- `database/sql` - 標準 SQL 介面
- `github.com/jmoiron/sqlx` - SQL 擴充
- `gorm.io/gorm` - ORM 框架

### 工具類
- `github.com/spf13/cobra` - CLI 框架
- `github.com/spf13/viper` - 設定管理
- `github.com/vincent119/zlogger` - 高效能日誌
- `github.com/go-playground/validator` - Struct 驗證
- `go.uber.org/fx` - 依賴注入 (DI)
- `github.com/prometheus/client_golang` - Prometheus Metrics
- `github.com/redis/go-redis/v9` - Redis Client
- `github.com/vincent119/commons` - 常用工具庫
- `github.com/swaggo/swag` - Swagger 文件產生
- `uber-go/mock` - Mock 生成工具 (gomock)
- `shields.io` - 狀態徽章 (README 用)

---

## Review Checklist
- [ ] 僅一個 `package` 宣告（置頂）
- [ ] 通過 `gofmt -s` / `goimports` / `go vet`
- [ ] 無未使用 imports、無循環依賴
- [ ] `err` 立即檢查並以 `%w` 包裝；跨層以 `errors.Is/As`
- [ ] goroutine / channel 正確收斂；無泄漏
- [ ] I/O 操作安全（含 Close、Pipe、Body 重新可讀）
- [ ] JSON tag 一致、解碼拒絕未知欄位、零值可用
- [ ] 測試含 table-driven、-race、必要 fuzz/bench
- [ ] `go.mod` 依賴釘版；`go mod tidy` 後無不明變更
- [ ] Server/Worker 實作 Graceful Shutdown (監聽 SIGINT/SIGTERM)
- [ ] 使用依賴注入 (DI)，無業務層手動 `New` 實體
- [ ] 跨邊界呼叫有傳遞 `context` (Trace ID)
- [ ] DB Migration 透過版本化腳本管理 (無 AutoMigrate)
- [ ] Domain Event 定義為不可變 struct，包含 EventID 與 OccurredAt
- [ ] 與 Uber / Effective Go 一致或於 PR 註明偏離理由

---

## CI 與工具建議（可直接採用）

### `.gitignore`（建議）
```gitignore
# Go
*.exe
*.test
*.out
coverage.out
vendor/

# IDE
.idea/
.vscode/
*.swp

# 環境與敏感資訊
.env
*.local.yaml
```

### `Makefile`（節選）
```makefile
.PHONY: tidy lint test bench cover fmt vet swagger

tidy:
	go mod tidy

lint:
	golangci-lint run ./...

test:
	go test -race -count=1 ./...

bench:
	go test -run=NONE -bench=. -benchmem ./...

# 執行測試並顯示覆蓋率
cover:
	go test -cover ./...

# 產生覆蓋率報告（HTML）
cover-html:
	go test -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out

# 格式化程式碼
fmt:
	go fmt ./...

# 靜態分析
vet:
	go vet ./...

# 更新 Swagger 文檔
swagger:
	swag init -g cmd/main.go
```

#### 常用測試指令
```bash
# 執行特定測試
go test -run TestFunctionName ./path/to/package

# 執行特定 package 的所有測試
go test -v ./internal/order/...

# 執行測試並產生覆蓋率報告
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

### `.golangci.yml`（節選）
```yaml
run:
  timeout: 5m
linters:
  enable:
    - errcheck
    - gocritic
    - gofumpt
    - govet
    - ineffassign
    - staticcheck
    - unparam
    - prealloc
    - revive
    - gosec
linters-settings:
  gosec:
    excludes:
      - G404
  revive:
    ignore-generated-header: true
issues:
  exclude-use-default: false
```

### PR 模板要點
- 目的與背景（為何要改）
- 變更摘要（做了什麼）
- 風險與復原方案
- 測試證據（覆蓋率、基準、相容性）
- 偏離 Uber/Effective Go 的理由（若有）

---

**建議存放路徑：** `.github/instructions/go.DDD.instructions.md`
此設定將自動套用於所有 Go 檔案（`*.go`, `go.mod`, `go.sum`）。
