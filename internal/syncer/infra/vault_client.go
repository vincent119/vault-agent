package infra

import (
	"context"
	"errors"
	"fmt"
	"net/http"

	vault "github.com/hashicorp/vault/api"
	"github.com/hashicorp/vault/api/auth/kubernetes"
	"github.com/vincent119/zlogger"

	"vault-agent/internal/syncer/domain"
)

// VaultClient 實作 domain.SecretFetcher 介面，從 HashiCorp Vault 讀取機密資料。
type VaultClient struct {
	client    *vault.Client
	mountPath string // KV v2 mount 名稱，如 "secret"
	k8sAuth   *kubernetes.KubernetesAuth // 非 nil 時支援 token 過期後自動重新認證
}

// NewVaultClient 建立一個連線至指定 Vault 地址的 Client（使用靜態 token）。
// mountPath 為 KV v2 的 mount 名稱（空字串則預設 "secret"）。
func NewVaultClient(address, token, mountPath string) (*VaultClient, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = address

	c, err := vault.NewClient(cfg)
	if err != nil {
		zlogger.Error("vault client create failed", zlogger.String("vault.address", address), zlogger.Err(err))
		return nil, fmt.Errorf("new vault client: %w", err)
	}
	c.SetToken(token)

	if mountPath == "" {
		mountPath = "secret"
	}
	zlogger.Info("vault client created (token auth)", zlogger.String("vault.address", address), zlogger.String("vault.mount_path", mountPath))
	return &VaultClient{client: c, mountPath: mountPath}, nil
}

// NewVaultClientWithK8sAuth 使用 Kubernetes 認證登入 Vault，取得 token 後建立 Client。
// authPath 為 Vault 內 K8s auth 的 mount 路徑（如 im-devops-eks、kubernetes）；
// role 為 Vault 內對應的 role 名稱（如 vault-agent）。Pod 內預設讀取 ServiceAccount JWT。
func NewVaultClientWithK8sAuth(ctx context.Context, address, mountPath, authPath, role string) (*VaultClient, error) {
	cfg := vault.DefaultConfig()
	cfg.Address = address

	c, err := vault.NewClient(cfg)
	if err != nil {
		zlogger.Error("vault client create failed", zlogger.String("vault.address", address), zlogger.Err(err))
		return nil, fmt.Errorf("new vault client: %w", err)
	}

	opts := []kubernetes.LoginOption{kubernetes.WithMountPath(authPath)}
	k8sAuth, err := kubernetes.NewKubernetesAuth(role, opts...)
	if err != nil {
		zlogger.Error("vault kubernetes auth init failed", zlogger.String("vault.auth_k8s_path", authPath), zlogger.String("vault.auth_k8s_role", role), zlogger.Err(err))
		return nil, fmt.Errorf("new kubernetes auth: %w", err)
	}

	secret, err := k8sAuth.Login(ctx, c)
	if err != nil {
		zlogger.Error("vault kubernetes login failed", zlogger.String("vault.address", address), zlogger.String("vault.auth_k8s_path", authPath), zlogger.String("vault.auth_k8s_role", role), zlogger.Err(err))
		return nil, fmt.Errorf("vault kubernetes login: %w", err)
	}
	if secret == nil || secret.Auth == nil {
		zlogger.Error("vault kubernetes login returned empty auth", zlogger.String("vault.address", address), zlogger.String("vault.auth_k8s_path", authPath), zlogger.String("vault.auth_k8s_role", role))
		return nil, fmt.Errorf("vault kubernetes login: empty auth response")
	}
	c.SetToken(secret.Auth.ClientToken)

	if mountPath == "" {
		mountPath = "secret"
	}
	zlogger.Info("vault client created (kubernetes auth)", zlogger.String("vault.address", address), zlogger.String("vault.auth_k8s_path", authPath), zlogger.String("vault.auth_k8s_role", role), zlogger.String("vault.mount_path", mountPath))
	return &VaultClient{client: c, mountPath: mountPath, k8sAuth: k8sAuth}, nil
}

// reAuthenticate 使用 Kubernetes auth 重新登入 Vault，並更新 client token。
func (vc *VaultClient) reAuthenticate(ctx context.Context) error {
	secret, err := vc.k8sAuth.Login(ctx, vc.client)
	if err != nil {
		return fmt.Errorf("vault re-auth login: %w", err)
	}
	if secret == nil || secret.Auth == nil {
		return fmt.Errorf("vault re-auth: empty auth response")
	}
	vc.client.SetToken(secret.Auth.ClientToken)
	zlogger.InfoContext(ctx, "vault re-authentication succeeded")
	return nil
}

// isVaultPermissionDenied 判斷 err 是否為 Vault 403 Permission Denied。
func isVaultPermissionDenied(err error) bool {
	var respErr *vault.ResponseError
	return errors.As(err, &respErr) && respErr.StatusCode == http.StatusForbidden
}

// FetchSecret 從 Vault KV v2 讀取機密資料，並僅回傳指定 keys。
// 若 keys 為空則回傳路徑下的全部 key-value pairs。
// 當回應為 403 且使用 Kubernetes auth 時，會自動重新認證後重試一次。
func (vc *VaultClient) FetchSecret(ctx context.Context, path string, keys []string) (map[string]string, error) {
	zlogger.Debug("fetching secret from vault", zlogger.String("vault.mount_path", vc.mountPath), zlogger.String("secret.path", path), zlogger.Int("secret.keys_count", len(keys)))
	secret, err := vc.client.KVv2(vc.mountPath).Get(ctx, path)
	if err != nil && vc.k8sAuth != nil && isVaultPermissionDenied(err) {
		zlogger.WarnContext(ctx, "vault token expired or permission denied, re-authenticating",
			zlogger.String("secret.path", path),
			zlogger.Err(err),
		)
		if reAuthErr := vc.reAuthenticate(ctx); reAuthErr != nil {
			zlogger.ErrorContext(ctx, "vault re-authentication failed", zlogger.Err(reAuthErr))
		} else {
			secret, err = vc.client.KVv2(vc.mountPath).Get(ctx, path)
		}
	}
	if err != nil {
		zlogger.Warn("vault fetch secret failed", zlogger.String("vault.mount_path", vc.mountPath), zlogger.String("secret.path", path), zlogger.Int("secret.keys_count", len(keys)), zlogger.Err(err))
		return nil, fmt.Errorf("%w: %w", domain.ErrSecretFetchFailed, err)
	}
	if secret == nil || secret.Data == nil {
		zlogger.Warn("vault secret not found", zlogger.String("vault.mount_path", vc.mountPath), zlogger.String("secret.path", path))
		return nil, fmt.Errorf("%w: path=%s", domain.ErrSecretNotFound, path)
	}

	result := make(map[string]string)

	if len(keys) == 0 {
		for k, v := range secret.Data {
			if s, ok := v.(string); ok {
				result[k] = s
			}
		}
		return result, nil
	}

	for _, k := range keys {
		v, ok := secret.Data[k]
		if !ok {
			return nil, fmt.Errorf("%w: key=%s at path=%s", domain.ErrSecretNotFound, k, path)
		}
		s, ok := v.(string)
		if !ok {
			return nil, fmt.Errorf("unexpected type for key %s", k)
		}
		result[k] = s
	}

	zlogger.Debug("fetch secret succeeded", zlogger.String("secret.path", path), zlogger.Int("result_keys", len(result)))
	return result, nil
}
