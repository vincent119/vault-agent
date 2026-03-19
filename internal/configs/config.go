// Package configs 提供 Config 結構體，用於載入系統設定。
package configs

import (
	"fmt"
	"strings"

	"github.com/spf13/viper"
)

// Config 包含所有系統設定。
// 優先順序：OS Env > config.yaml > 預設值
type Config struct {
	App       AppConfig       `mapstructure:"app"`
	Log       LogConfig       `mapstructure:"log"`
	Vault     VaultConfig     `mapstructure:"vault"`
	AWS       AWSConfig       `mapstructure:"aws"`
	K8s       K8sConfig       `mapstructure:"k8s"`
	Sync      SyncConfig      `mapstructure:"sync"`
	Telemetry TelemetryConfig `mapstructure:"telemetry"`
	Metrics   MetricsConfig   `mapstructure:"metrics"`
	TLS       TLSConfig       `mapstructure:"tls"`
}

type AppConfig struct {
	Name string `mapstructure:"name"`
	Port int    `mapstructure:"port"`
	Env  string `mapstructure:"env"`
}

// LogConfig 日誌設定，對應 zlogger 的 Config（Level、Format、Outputs 等）。
// 可由 config.yaml 的 log 區塊或環境變數 LOG_* 載入。
type LogConfig struct {
	Level         string   `mapstructure:"level"`          // debug, info, warn, error, fatal
	Format        string   `mapstructure:"format"`        // json, console
	Outputs       []string `mapstructure:"outputs"`        // console, file
	LogPath       string   `mapstructure:"log_path"`       // 檔案輸出目錄
	FileName      string   `mapstructure:"file_name"`      // 檔案名稱（空則依日期）
	AddCaller     bool     `mapstructure:"add_caller"`     // 是否顯示 caller
	AddStacktrace bool     `mapstructure:"add_stacktrace"` // 是否在 error 時輸出 stack
	Development   bool     `mapstructure:"development"`   // 開發模式（較詳細）
	ColorEnabled  bool     `mapstructure:"color_enabled"` // console 是否上色
}

// Addr 回傳 HTTP 監聽地址字串，如 ":8080"
func (a AppConfig) Addr() string {
	port := a.Port
	if port == 0 {
		port = 8080
	}
	return fmt.Sprintf(":%d", port)
}

type VaultConfig struct {
	Address   string `mapstructure:"address"`
	Token     string `mapstructure:"token"` // 靜態 token；與 Kubernetes 認證二擇一
	MountPath string `mapstructure:"mount_path"` // KV v2 mount 名稱，預設 "secret"
	// Kubernetes 認證：當 Token 為空時使用。與 vault-py 對齊：auth_path 即 Vault 內 K8s auth mount 路徑（如 im-devops-eks）
	AuthK8sPath string `mapstructure:"auth_k8s_path"` // 例：im-devops-eks、kubernetes
	AuthK8sRole string `mapstructure:"auth_k8s_role"` // 例：vault-py、vault-agent
}

type AWSConfig struct {
	Region string `mapstructure:"region"`
}

// K8sConfig 與 Kubernetes 相關的設定
type K8sConfig struct {
	// Kubeconfig 本機開發用路徑（Pod 內部署時留空，使用 In-Cluster Config）
	Kubeconfig string `mapstructure:"kubeconfig"`
	// Namespace 限定 Sync Worker 掃描的命名空間；空字串表示掃描全部
	Namespace string `mapstructure:"namespace"`
}

// SyncConfig 背景同步 Worker 相關設定
type SyncConfig struct {
	// IntervalSeconds 每次同步輪詢的間隔秒數
	IntervalSeconds int `mapstructure:"interval_seconds"`
}

// TelemetryConfig OpenTelemetry 追蹤相關設定
type TelemetryConfig struct {
	// Enabled 為 true 時才初始化 Tracer
	Enabled bool `mapstructure:"enabled"`
	// OTLPEndpoint 為空時 Tracer 輸出至 stdout；非空時依 OTLPTransport 送 gRPC 或 HTTP
	OTLPEndpoint string `mapstructure:"otlp_endpoint"`
	// OTLPTransport 傳輸協定："grpc"（預設，port 4317）或 "http"（port 4318）
	OTLPTransport string `mapstructure:"otlp_transport"`
	// OTLPCompression 壓縮："none"（預設）或 "gzip"。對應 [configgrpc](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc) 的 compression
	OTLPCompression string `mapstructure:"otlp_compression"`
	// OTLPHeaders 認證或自訂 header，格式 "key1=value1,key2=value2"（與 OTEL_EXPORTER_OTLP_HEADERS 一致）；可用於 Authorization 等
	OTLPHeaders string `mapstructure:"otlp_headers"`
	// OTLPBasicAuth 對 OTLP 收集器使用 HTTP Basic 認證（gRPC/HTTP 皆支援），格式 "user:password"；對應 [configauth](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configauth) Client Basic Auth
	OTLPBasicAuth string `mapstructure:"otlp_basic_auth"`
}

// MetricsConfig Prometheus 指標相關設定
type MetricsConfig struct {
	// Enabled 為 true 時才暴露 /metrics
	Enabled  bool   `mapstructure:"enabled"`
	BasicAuth string `mapstructure:"basic_auth"` // 非空時為 "user:password"，對 /metrics 啟用 Basic Auth
}

// TLSConfig Webhook Server TLS 憑證設定
type TLSConfig struct {
	CertFile string `mapstructure:"cert_file"`
	KeyFile  string `mapstructure:"key_file"`
}

func LoadConfig() (*Config, error) {
	v := viper.New()
	v.SetConfigName("config")
	v.SetConfigType("yaml")
	v.AddConfigPath("./configs")
	v.AddConfigPath("../configs")
	v.AddConfigPath("../../configs")

	v.AutomaticEnv()
	v.SetEnvKeyReplacer(strings.NewReplacer(".", "_"))

	// 明確 BindEnv：Viper 的 AutomaticEnv 對巢狀 key 在 Unmarshal 路徑下無法自動對應，
	// 必須顯式宣告每個 key 對應的環境變數名稱。
	_ = v.BindEnv("app.name", "APP_NAME")
	_ = v.BindEnv("app.env", "APP_ENV")
	_ = v.BindEnv("app.port", "APP_PORT")
	_ = v.BindEnv("log.level", "LOG_LEVEL")
	_ = v.BindEnv("log.format", "LOG_FORMAT")
	_ = v.BindEnv("log.outputs", "LOG_OUTPUTS")
	_ = v.BindEnv("log.log_path", "LOG_LOG_PATH")
	_ = v.BindEnv("log.file_name", "LOG_FILE_NAME")
	_ = v.BindEnv("log.add_caller", "LOG_ADD_CALLER")
	_ = v.BindEnv("log.add_stacktrace", "LOG_ADD_STACKTRACE")
	_ = v.BindEnv("log.development", "LOG_DEVELOPMENT")
	_ = v.BindEnv("log.color_enabled", "LOG_COLOR_ENABLED")
	_ = v.BindEnv("vault.address", "VAULT_ADDRESS")
	_ = v.BindEnv("vault.token", "VAULT_TOKEN")
	_ = v.BindEnv("vault.mount_path", "VAULT_MOUNT_PATH")
	_ = v.BindEnv("vault.auth_k8s_path", "VAULT_AUTH_K8S_PATH")
	_ = v.BindEnv("vault.auth_k8s_role", "VAULT_AUTH_K8S_ROLE")
	_ = v.BindEnv("aws.region", "AWS_REGION")
	_ = v.BindEnv("k8s.kubeconfig", "K8S_KUBECONFIG")
	_ = v.BindEnv("k8s.namespace", "K8S_NAMESPACE")
	_ = v.BindEnv("sync.interval_seconds", "SYNC_INTERVAL_SECONDS")
	_ = v.BindEnv("telemetry.enabled", "TELEMETRY_ENABLED")
	_ = v.BindEnv("telemetry.otlp_endpoint", "TELEMETRY_OTLP_ENDPOINT")
	_ = v.BindEnv("telemetry.otlp_transport", "TELEMETRY_OTLP_TRANSPORT")
	_ = v.BindEnv("telemetry.otlp_compression", "TELEMETRY_OTLP_COMPRESSION")
	_ = v.BindEnv("telemetry.otlp_headers", "TELEMETRY_OTLP_HEADERS")
	_ = v.BindEnv("telemetry.otlp_basic_auth", "TELEMETRY_OTLP_BASIC_AUTH")
	_ = v.BindEnv("metrics.enabled", "METRICS_ENABLED")
	_ = v.BindEnv("metrics.basic_auth", "METRICS_BASIC_AUTH")
	_ = v.BindEnv("tls.cert_file", "TLS_CERT_FILE")
	_ = v.BindEnv("tls.key_file", "TLS_KEY_FILE")

	// 設定預設值須在 ReadInConfig 之前，確保邏輯明確
	v.SetDefault("app.port", 8080)
	v.SetDefault("app.env", "dev")
	v.SetDefault("log.level", "info")
	v.SetDefault("log.format", "console")
	v.SetDefault("log.outputs", []string{"console"})
	v.SetDefault("log.add_caller", true)
	v.SetDefault("log.development", true)
	v.SetDefault("log.color_enabled", true)
	v.SetDefault("sync.interval_seconds", 60)
	v.SetDefault("vault.mount_path", "secret")
	v.SetDefault("telemetry.enabled", true)
	v.SetDefault("telemetry.otlp_transport", "grpc")
	v.SetDefault("telemetry.otlp_compression", "none")
	v.SetDefault("metrics.enabled", true)

	// 嘗試讀取 config 檔案；找不到時允許只依賴 OS / K8s Env 繼續執行
	if err := v.ReadInConfig(); err != nil {
		if _, ok := err.(viper.ConfigFileNotFoundError); !ok {
			return nil, fmt.Errorf("read config: %w", err)
		}
	}

	var cfg Config
	if err := v.Unmarshal(&cfg); err != nil {
		return nil, fmt.Errorf("unmarshal config: %w", err)
	}
	return &cfg, nil
}
