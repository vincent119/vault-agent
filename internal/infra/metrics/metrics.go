// Package metrics 提供 Prometheus 指標定義，用於監控 Vault Agent 的運行狀態。
package metrics

import (
	"encoding/base64"
	"net/http"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// MutateRequestsTotal 紀錄 Webhook 收到請求的次數
	MutateRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vault_agent_mutate_requests_total",
			Help: "Total number of mutating webhook requests received",
		},
		[]string{"status"}, // 例: success, error
	)

	// SyncErrorsTotal 紀錄背景同步時發生的錯誤次數
	SyncErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "vault_agent_sync_errors_total",
			Help: "Total number of background sync errors",
		},
		[]string{"source"}, // 例: vault, aws
	)

	// SecretFetchDuration 紀錄從外部機密系統拉取資料的耗時
	SecretFetchDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "vault_agent_secret_fetch_duration_seconds",
			Help:    "Duration of fetching secrets from backends",
			Buckets: prometheus.DefBuckets,
		},
		[]string{"backend"}, // 例: vault, aws
	)
)

// InitMetrics 預先初始化所有已知 label 組合，確保指標在首次請求前即出現於 /metrics。
func InitMetrics() {
	// 預先建立 MutateRequestsTotal 的 label 組合，避免 webhook 未觸發時指標不存在
	MutateRequestsTotal.WithLabelValues("success").Add(0)
	MutateRequestsTotal.WithLabelValues("error").Add(0)

	// 預先建立 SyncErrorsTotal 的 label 組合
	SyncErrorsTotal.WithLabelValues("vault").Add(0)
	SyncErrorsTotal.WithLabelValues("aws").Add(0)
}

// WrapWithBasicAuth 若 basicAuth 非空（格式為 "user:password"），則用 Basic Auth 保護 next；空字串則直接回傳 next。
func WrapWithBasicAuth(next http.Handler, basicAuth string) http.Handler {
	if basicAuth == "" {
		return next
	}
	expected := "Basic " + base64.StdEncoding.EncodeToString([]byte(basicAuth))
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		const prefix = "Basic "
		s := r.Header.Get("Authorization")
		if s == "" || !strings.HasPrefix(s, prefix) {
			w.Header().Set("WWW-Authenticate", `Basic realm="metrics"`)
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		got := strings.TrimSpace(s)
		if got != expected {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}
