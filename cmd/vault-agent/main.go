// Package main 提供 Vault Agent 的進入點，使用 github.com/vincent119/commons/graceful 做優雅啟動與關機。
package main

import (
	"context"
	"errors"
	"net/http"
	"time"

	"vault-agent/internal/configs"
	"vault-agent/internal/infra/logger"
	"vault-agent/internal/infra/metrics"
	"vault-agent/internal/infra/telemetry"
	"vault-agent/internal/syncer/application"
	"vault-agent/internal/syncer/delivery"
	"vault-agent/internal/syncer/domain"
	"vault-agent/internal/syncer/infra"

	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/vincent119/commons/graceful"
	"github.com/vincent119/zlogger"
	"go.uber.org/zap"
)

func main() {
	// 1. 載入設定
	cfg, err := configs.LoadConfig()
	if err != nil {
		zlogger.Fatal("load config failed", zap.Error(err))
	}

	// 2. 初始化全域 Logger
	log, err := logger.InitLogger(cfg.App.Env, &cfg.Log)
	if err != nil {
		zlogger.Fatal("init logger failed", zap.Error(err))
	}
	defer func() { _ = log.Sync() }()

	log.Info("config loaded",
		zlogger.String("app.name", cfg.App.Name),
		zlogger.String("app.name", cfg.App.Name),
		zlogger.String("app.env", cfg.App.Env),
		zlogger.Int("app.port", cfg.App.Port),
		zlogger.Bool("telemetry.enabled", cfg.Telemetry.Enabled),
		zlogger.Bool("metrics.enabled", cfg.Metrics.Enabled),
		zlogger.Bool("metrics.basic_auth.enabled", cfg.Metrics.BasicAuth != ""),
		zlogger.Bool("k8s.kubeconfig.provided", cfg.K8s.Kubeconfig != ""),
		zlogger.String("k8s.namespace", cfg.K8s.Namespace),
		zlogger.Int("sync.interval_seconds", cfg.Sync.IntervalSeconds),
		zlogger.String("vault.address", cfg.Vault.Address),
		zlogger.String("vault.mount_path", cfg.Vault.MountPath),
		zlogger.Bool("vault.token.provided", cfg.Vault.Token != ""),
		zlogger.Bool("vault.k8s_auth.provided", cfg.Vault.AuthK8sPath != "" && cfg.Vault.AuthK8sRole != ""),
		zlogger.String("aws.region", cfg.AWS.Region),
	)

	// 3. OTLP Tracing（僅在 telemetry.enabled 時）
	var shutdownTracer func(context.Context) error
	if cfg.Telemetry.Enabled {
		if cfg.Telemetry.OTLPEndpoint == "" {
			log.Info("telemetry enabled (stdout exporter)")
		} else {
			log.Info("telemetry enabled (otlp exporter)",
				zlogger.String("telemetry.otlp_endpoint", cfg.Telemetry.OTLPEndpoint),
				zlogger.String("telemetry.otlp_transport", cfg.Telemetry.OTLPTransport),
				zlogger.String("telemetry.otlp_compression", cfg.Telemetry.OTLPCompression),
				zap.Bool("telemetry.otlp_headers.set", cfg.Telemetry.OTLPHeaders != ""),
				zlogger.Bool("telemetry.otlp_basic_auth.set", cfg.Telemetry.OTLPBasicAuth != ""))
		}
		st, err := telemetry.InitTracer(context.Background(), cfg.App.Name, &cfg.Telemetry)
		if err != nil {
			log.Warn("init tracer failed, continuing without tracing", zap.Error(err))
		} else {
			shutdownTracer = st
		}
	} else {
		log.Info("telemetry disabled")
	}

	// 4. K8s Repository（kubeconfig 為空時僅用 In-Cluster）
	k8sRepo, err := infra.NewK8sRepository(cfg.K8s.Kubeconfig)
	if err != nil {
		log.Warn("k8s repository disabled (no kubeconfig and not in-cluster), sync worker will not run", zap.Error(err))
		k8sRepo = nil
	} else {
		log.Info("k8s repository initialized")
	}

	// 5. 機密來源 Fetchers
	fetchers := make(map[string]domain.SecretFetcher)
	if cfg.Vault.Token != "" {
		log.Info("vault backend enabled (token auth)")
		vaultClient, err := infra.NewVaultClient(cfg.Vault.Address, cfg.Vault.Token, cfg.Vault.MountPath)
		if err != nil {
			log.Fatal("init vault client failed", zap.Error(err))
		}
		fetchers["vault"] = vaultClient
	} else if cfg.Vault.AuthK8sPath != "" && cfg.Vault.AuthK8sRole != "" {
		log.Info("vault backend enabled (kubernetes auth)", zap.String("vault.auth_k8s_path", cfg.Vault.AuthK8sPath), zap.String("vault.auth_k8s_role", cfg.Vault.AuthK8sRole))
		vaultClient, err := infra.NewVaultClientWithK8sAuth(context.Background(), cfg.Vault.Address, cfg.Vault.MountPath, cfg.Vault.AuthK8sPath, cfg.Vault.AuthK8sRole)
		if err != nil {
			log.Fatal("init vault client failed", zap.Error(err))
		}
		fetchers["vault"] = vaultClient
	} else {
		log.Warn("vault not configured (no token nor k8s auth), backend vault will be unavailable for mutate/sync")
	}

	log.Info("aws backend enabled", zap.String("aws.region", cfg.AWS.Region))
	awsClient, err := infra.NewAWSClient(context.Background(), cfg.AWS.Region)
	if err != nil {
		log.Fatal("init aws client failed", zap.Error(err))
	}
	fetchers["aws"] = awsClient

	// 6. Application 層
	mutateUC := application.NewMutateUseCase(fetchers)
	var syncWorker *application.SyncWorkerUseCase
	if k8sRepo != nil {
		log.Info("sync worker enabled")
		syncWorker = application.NewSyncWorkerUseCase(
			fetchers,
			k8sRepo,
			cfg.K8s.Namespace,
			time.Duration(cfg.Sync.IntervalSeconds)*time.Second,
		)
	} else {
		log.Info("sync worker disabled")
	}

	// 7. HTTP 路由與 Server
	metrics.InitMetrics()
	mux := http.NewServeMux()
	mux.Handle("/mutate", delivery.NewWebhookHandler(mutateUC))
	if cfg.Metrics.Enabled {
		if cfg.Metrics.BasicAuth != "" {
			log.Info("metrics enabled (basic auth)")
		} else {
			log.Info("metrics enabled")
		}
		mux.Handle("/metrics", metrics.WrapWithBasicAuth(promhttp.Handler(), cfg.Metrics.BasicAuth))
	} else {
		log.Info("metrics disabled")
	}
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, _ *http.Request) { w.WriteHeader(http.StatusOK) })

	srv := &http.Server{
		Addr:              cfg.App.Addr(),
		Handler:           mux,
		ReadHeaderTimeout: 5 * time.Second,
		ReadTimeout:       10 * time.Second,
		WriteTimeout:      10 * time.Second,
		IdleTimeout:      120 * time.Second,
	}

	// 8. 組裝 graceful 主任務：啟動 Sync Worker（若有）+ HTTP Server，等候訊號
	tlsCfg := cfg.TLS
	task := func(ctx context.Context) error {
		if syncWorker != nil {
			go syncWorker.Run(ctx)
		}
		errCh := make(chan error, 1)
		go func() {
			var serveErr error
			if tlsCfg.CertFile != "" && tlsCfg.KeyFile != "" {
				serveErr = srv.ListenAndServeTLS(tlsCfg.CertFile, tlsCfg.KeyFile)
			} else {
				serveErr = srv.ListenAndServe()
			}
			if serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
				errCh <- serveErr
			}
			close(errCh)
		}()
		log.Info("server listening", zap.String("addr", srv.Addr))
		select {
		case err := <-errCh:
			return err
		case <-ctx.Done():
			log.Info("shutdown signal received")
			return nil
		}
	}

	opts := []graceful.Option{
		graceful.WithTimeout(30 * time.Second),
	}
	if shutdownTracer != nil {
		opts = append(opts, graceful.WithCleanup(func(ctx context.Context) error {
			log.Info("shutting down tracer...")
			return shutdownTracer(ctx)
		}))
	}
	opts = append(opts, graceful.WithCleanup(func(ctx context.Context) error {
		log.Info("shutting down server...")
		if err := srv.Shutdown(ctx); err != nil {
			_ = srv.Close()
			return err
		}
		return nil
	}))

	if err := graceful.Run(task, opts...); err != nil {
		log.Error("application exited with error", zap.Error(err))
		// graceful 已執行完 cleanup，這裡只負責 exit code
	}
	log.Info("server exited")
}
