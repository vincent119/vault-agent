// Package application 提供 MutateUseCase 實作，用於處理 K8s Mutating Webhook 的核心流程。
package application

import (
	"context"
	"encoding/json"
	"fmt"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/vincent119/zlogger"

	"vault-agent/internal/infra/metrics"
	"vault-agent/internal/syncer/domain"
)

// jsonPatch 代表單個 JSON Patch 操作。
type jsonPatch struct {
	Op    string      `json:"op"`
	Path  string      `json:"path"`
	Value interface{} `json:"value"`
}

// MutateUseCase 處理 K8s Mutating Webhook 的核心流程。
type MutateUseCase struct {
	fetchers map[string]domain.SecretFetcher // key: backend 名稱 (vault/aws)
}

// NewMutateUseCase 建立 MutateUseCase，注入所有可用的 SecretFetcher 實作。
func NewMutateUseCase(fetchers map[string]domain.SecretFetcher) *MutateUseCase {
	return &MutateUseCase{fetchers: fetchers}
}

// Execute 解析 AdmissionReview 並回傳注入機密後的 JSON Patch bytes。
// 若 Pod 上無 inject annotation（inject-vault-agent != "true"），則直接允許通過（空 patch）。
func (uc *MutateUseCase) Execute(ctx context.Context, req *admissionv1.AdmissionRequest) ([]byte, error) {
	var pod corev1.Pod
	if err := json.Unmarshal(req.Object.Raw, &pod); err != nil {
		return nil, fmt.Errorf("unmarshal pod: %w", err)
	}

	// 明確檢查 inject opt-in annotation，未啟用直接放行
	annotations := pod.Annotations
	if annotations == nil || annotations[domain.AnnotationInject] != "true" {
		zlogger.DebugContext(ctx, "mutate skipped, inject not enabled")
		return []byte("[]"), nil
	}

	ref, err := domain.ParseSecretRef(annotations)
	if err != nil {
		return nil, fmt.Errorf("parse secret ref: %w", err)
	}
	if ref == nil {
		zlogger.DebugContext(ctx, "mutate skipped, no path")
		return []byte("[]"), nil
	}

	fetcher, ok := uc.fetchers[ref.Backend]
	if !ok {
		return nil, fmt.Errorf("unknown backend: %s", ref.Backend)
	}

	timer := prometheus.NewTimer(metrics.SecretFetchDuration.WithLabelValues(ref.Backend))
	data, fetchErr := fetcher.FetchSecret(ctx, ref.Path, ref.Keys)
	timer.ObserveDuration()
	if fetchErr != nil {
		return nil, fmt.Errorf("fetch secret (backend=%s path=%s): %w", ref.Backend, ref.Path, fetchErr)
	}

	patches, err := buildEnvPatches(pod.Spec.Containers, data)
	if err != nil {
		return nil, fmt.Errorf("build patches: %w", err)
	}

	patchBytes, err := json.Marshal(patches)
	if err != nil {
		return nil, fmt.Errorf("marshal patches: %w", err)
	}
	zlogger.InfoContext(ctx, "mutate applied",
		zlogger.String("backend", ref.Backend),
		zlogger.String("path", ref.Path),
		zlogger.Int("containers", len(pod.Spec.Containers)),
	)
	return patchBytes, nil
}

// buildEnvPatches 針對 Pod 所有 container 建構注入 Env 的 JSON Patch 列表。
// 若 container.Env 為 nil，會先 add 一個空 array，再 append 各個環境變數，
// 避免直接對 nil env 使用 "-" 路徑導致 patch 失敗。
func buildEnvPatches(containers []corev1.Container, data map[string]string) ([]jsonPatch, error) {
	if len(data) == 0 {
		return nil, nil
	}

	var patches []jsonPatch
	for i, c := range containers {
		if c.Env == nil {
			patches = append(patches, jsonPatch{
				Op:    "add",
				Path:  fmt.Sprintf("/spec/containers/%d/env", i),
				Value: []corev1.EnvVar{},
			})
		}
		for k, v := range data {
			patches = append(patches, jsonPatch{
				Op:   "add",
				Path: fmt.Sprintf("/spec/containers/%d/env/-", i),
				Value: corev1.EnvVar{
					Name:  k,
					Value: v,
				},
			})
		}
	}
	return patches, nil
}
