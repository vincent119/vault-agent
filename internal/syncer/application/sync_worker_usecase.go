package application

import (
	"context"
	"fmt"
	"time"

	"github.com/vincent119/zlogger"
	corev1 "k8s.io/api/core/v1"

	"github.com/prometheus/client_golang/prometheus"

	"vault-agent/internal/infra/metrics"
	"vault-agent/internal/syncer/domain"
)

// K8sSecretRepo 定義 Sync Worker 需要的 K8s 操作介面（Interface Segregation）。
type K8sSecretRepo interface {
	ListSecretsByLabel(ctx context.Context, namespace, labelSelector string) ([]corev1.Secret, error)
	UpdateSecret(ctx context.Context, secret *corev1.Secret) error
}

// SyncWorkerUseCase 定時掃描帶有同步標籤的 K8s Secret，並從機密來源重整更新。
type SyncWorkerUseCase struct {
	fetchers      map[string]domain.SecretFetcher
	k8s           K8sSecretRepo
	namespace     string
	syncInterval  time.Duration
	labelSelector string
}

// NewSyncWorkerUseCase 建立 SyncWorkerUseCase。
func NewSyncWorkerUseCase(
	fetchers map[string]domain.SecretFetcher,
	k8s K8sSecretRepo,
	namespace string,
	syncInterval time.Duration,
) *SyncWorkerUseCase {
	return &SyncWorkerUseCase{
		fetchers:      fetchers,
		k8s:           k8s,
		namespace:     namespace,
		syncInterval:  syncInterval,
		labelSelector: domain.AnnotationInject + "=true",
	}
}

// Run 啟動定時輪詢。監聽 ctx.Done() 以支援優雅關機。
// 每輪 syncOnce 使用獨立 context（帶 syncInterval timeout），
// 確保關機信號不會中斷正在進行的同步操作。
func (uc *SyncWorkerUseCase) Run(ctx context.Context) {
	ctx = zlogger.WithComponent(ctx, "sync_worker")
	ticker := time.NewTicker(uc.syncInterval)
	defer ticker.Stop()

	zlogger.InfoContext(ctx, "sync worker started", zlogger.Duration("interval", uc.syncInterval))

	for {
		select {
		case <-ctx.Done():
			zlogger.InfoContext(ctx, "sync worker stopped")
			return
		case <-ticker.C:
			// 使用獨立 context，讓當輪同步能完成而不被關機信號打斷
			func() {
				syncCtx, cancel := context.WithTimeout(context.Background(), uc.syncInterval)
				defer cancel()
				syncCtx = zlogger.WithComponent(syncCtx, "sync_worker")
				syncCtx = zlogger.WithOperation(syncCtx, "sync_once")
				uc.syncOnce(syncCtx)
			}()
		}
	}
}

// syncOnce 執行一次完整的掃描與同步循環。
func (uc *SyncWorkerUseCase) syncOnce(ctx context.Context) {
	zlogger.DebugContext(ctx, "starting secret scan", zlogger.String("namespace", uc.namespace))
	secrets, err := uc.k8s.ListSecretsByLabel(ctx, uc.namespace, uc.labelSelector)
	if err != nil {
		zlogger.ErrorContext(ctx, "list secrets failed", zlogger.Err(err))
		return
	}

	zlogger.DebugContext(ctx, "scan completed", zlogger.Int("secrets_count", len(secrets)))
	for i := range secrets {
		s := &secrets[i]
		if err := uc.syncSecret(ctx, s); err != nil {
			zlogger.ErrorContext(ctx, "sync secret failed",
				zlogger.String("namespace", s.Namespace),
				zlogger.String("name", s.Name),
				zlogger.Err(err),
			)
		}
	}
}

// syncSecret 針對單一 Secret，從對應的機密來源重新拉取資料並比對後更新。
func (uc *SyncWorkerUseCase) syncSecret(ctx context.Context, secret *corev1.Secret) error {
	zlogger.DebugContext(ctx, "evaluating secret", zlogger.String("namespace", secret.Namespace), zlogger.String("name", secret.Name))
	ref, err := domain.ParseSecretRef(secret.Annotations)
	if err != nil {
		return fmt.Errorf("parse secret ref: %w", err)
	}
	if ref == nil {
		return nil
	}

	fetcher, ok := uc.fetchers[ref.Backend]
	if !ok {
		zlogger.WarnContext(ctx, "unknown backend, skipping secret",
			zlogger.String("backend", ref.Backend),
			zlogger.String("namespace", secret.Namespace),
			zlogger.String("name", secret.Name),
		)
		return nil
	}

	timer := prometheus.NewTimer(metrics.SecretFetchDuration.WithLabelValues(ref.Backend))
	data, fetchErr := fetcher.FetchSecret(ctx, ref.Path, ref.Keys)
	timer.ObserveDuration()
	if fetchErr != nil {
		metrics.SyncErrorsTotal.WithLabelValues(ref.Backend).Inc()
		return fetchErr
	}

	// 比對並更新 Secret.Data（僅在有差異時才 Update，避免不必要的 API 呼叫）
	updated := false
	if secret.Data == nil {
		secret.Data = make(map[string][]byte)
	}
	for k, v := range data {
		if string(secret.Data[k]) != v {
			secret.Data[k] = []byte(v)
			updated = true
		}
	}

	if !updated {
		zlogger.DebugContext(ctx, "secret data unchanged, skipping update", zlogger.String("namespace", secret.Namespace), zlogger.String("name", secret.Name))
		return nil
	}

	zlogger.InfoContext(ctx, "updating secret",
		zlogger.String("namespace", secret.Namespace),
		zlogger.String("name", secret.Name),
	)
	return uc.k8s.UpdateSecret(ctx, secret)
}
