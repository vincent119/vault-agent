// Package logger 提供應用程式日誌初始化，依 config 的 log 設定與 env 呼叫 zlogger。
package logger

import (
	"github.com/vincent119/zlogger"
	"go.uber.org/zap"

	"vault-agent/internal/configs"
)

// InitLogger 使用 config 的 log 設定初始化全域日誌；若 log 為空則依 env 使用預設（prod=json/非 dev，其餘=console/dev）。
func InitLogger(env string, logCfg *configs.LogConfig) (*zap.Logger, error) {
	cfg := toZloggerConfig(env, logCfg)
	zlogger.Init(cfg)
	return zlogger.GetLogger(), nil
}

// toZloggerConfig 將 configs.LogConfig 轉成 zlogger.Config，並依 env 套用 prod 覆寫。
func toZloggerConfig(env string, c *configs.LogConfig) *zlogger.Config {
	cfg := &zlogger.Config{
		Level:        "info",
		Format:       "console",
		Outputs:      []string{"console"},
		AddCaller:    true,
		Development:  true,
		ColorEnabled: true,
	}
	if c != nil {
		if c.Level != "" {
			cfg.Level = c.Level
		}
		if c.Format != "" {
			cfg.Format = c.Format
		}
		if len(c.Outputs) > 0 {
			cfg.Outputs = c.Outputs
		}
		cfg.LogPath = c.LogPath
		cfg.FileName = c.FileName
		cfg.AddCaller = c.AddCaller
		cfg.AddStacktrace = c.AddStacktrace
		cfg.Development = c.Development
		cfg.ColorEnabled = c.ColorEnabled
	}
	if env == "prod" {
		cfg.Format = "json"
		cfg.Development = false
		cfg.ColorEnabled = false
	}
	return cfg
}
