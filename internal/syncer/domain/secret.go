// Package domain 提供 SecretFetcher 與 SecretRef 的定義，用於處理機密資料的獲取與解析。
package domain

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// Annotation 鍵值常數，集中定義避免散落在各層造成不一致。
const (
	AnnotationInject  = "inject-vault-agent"
	AnnotationBackend = "inject-vault-agent.backend"
	AnnotationPath    = "inject-vault-agent.path"
	AnnotationKeys    = "inject-vault-agent.keys"

	DefaultBackend = "vault"
)

// ErrSecretNotFound 代表查詢機密資料時，目標路徑不存在的語意錯誤。
var ErrSecretNotFound = errors.New("secret not found")

// ErrSecretFetchFailed 代表外部機密系統呼叫失敗（網路、憑證等原因）。
var ErrSecretFetchFailed = errors.New("secret fetch failed")

// SecretFetcher 定義從任何機密來源 (Vault / AWS Secrets Manager) 取得機密資料的抽象介面。
// 所有實作皆置於 infra 層。
type SecretFetcher interface {
	// FetchSecret 從指定路徑取出機密。
	//  - path: 機密路徑，由上層業務決定語意 (Vault path 或 AWS secret name)
	//  - keys: 若為空 slice，則回傳該機密路徑下的所有 key-value pairs
	//  回傳 map[key]value，若找不到則回傳 ErrSecretNotFound
	FetchSecret(ctx context.Context, path string, keys []string) (map[string]string, error)
}

// SecretRef 代表一個待注入機密的引用描述（來自 Pod / Secret annotation）。
type SecretRef struct {
	Backend string   // 機密來源後端，如 "vault" 或 "aws"
	Path    string   // 機密路徑
	Keys    []string // 指定要取出的 keys；空則取全部
}

// ParseSecretRef 從 annotation map 解析 SecretRef。
// 若 AnnotationPath 為空則回傳 nil（代表無需注入）。
// 注意：此函式不檢查 AnnotationInject，由呼叫端自行決定 opt-in 邏輯：
//   - MutateUseCase 在呼叫前先檢查 AnnotationInject == "true"
//   - SyncWorkerUseCase 已透過 label selector 完成過濾
func ParseSecretRef(annotations map[string]string) (*SecretRef, error) {
	backend := annotations[AnnotationBackend]
	if backend == "" {
		backend = DefaultBackend
	}

	path := annotations[AnnotationPath]
	if path == "" {
		return nil, nil
	}

	var keys []string
	if raw := annotations[AnnotationKeys]; raw != "" {
		if err := json.Unmarshal([]byte(raw), &keys); err != nil {
			return nil, fmt.Errorf("parse %s: %w", AnnotationKeys, err)
		}
	}

	return &SecretRef{Backend: backend, Path: path, Keys: keys}, nil
}
