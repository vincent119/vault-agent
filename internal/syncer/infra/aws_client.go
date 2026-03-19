// Package infra 提供 AWSClient 實作，用於從 AWS Secrets Manager 讀取機密資料。
package infra

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/secretsmanager"
	"github.com/vincent119/zlogger"

	"vault-agent/internal/syncer/domain"
)

// AWSClient 實作 domain.SecretFetcher 介面，從 AWS Secrets Manager 讀取機密資料。
type AWSClient struct {
	sm *secretsmanager.Client
}

// NewAWSClient 建立一個 AWS Secrets Manager Client，Region 由環境變數或 config 決定。
func NewAWSClient(ctx context.Context, region string) (*AWSClient, error) {
	cfg, err := config.LoadDefaultConfig(ctx,
		config.WithRegion(region),
	)
	if err != nil {
		zlogger.Error("aws client create failed", zlogger.String("aws.region", region), zlogger.Err(err))
		return nil, fmt.Errorf("aws load config: %w", err)
	}
	zlogger.Info("aws client created", zlogger.String("aws.region", region))
	return &AWSClient{sm: secretsmanager.NewFromConfig(cfg)}, nil
}

// FetchSecret 從 AWS Secrets Manager 讀取機密，secret 值通常為 JSON 格式。
// 若 keys 為空則回傳解析後的所有 key-value pairs；否則僅回傳指定 keys。
func (ac *AWSClient) FetchSecret(ctx context.Context, path string, keys []string) (map[string]string, error) {
	zlogger.DebugContext(ctx, "aws fetching secret", zlogger.String("secret.path", path), zlogger.Int("secret.keys_count", len(keys)))
	out, err := ac.sm.GetSecretValue(ctx, &secretsmanager.GetSecretValueInput{
		SecretId: aws.String(path),
	})
	if err != nil {
		zlogger.Warn("aws fetch secret failed", zlogger.String("secret.path", path), zlogger.Int("secret.keys_count", len(keys)), zlogger.Err(err))
		return nil, fmt.Errorf("%w: %w", domain.ErrSecretFetchFailed, err)
	}
	if out.SecretString == nil {
		zlogger.Warn("aws secret not found", zlogger.String("secret.path", path))
		return nil, fmt.Errorf("%w: path=%s", domain.ErrSecretNotFound, path)
	}

	// 解析 JSON 格式的 SecretString
	raw := make(map[string]string)
	if err := json.Unmarshal([]byte(*out.SecretString), &raw); err != nil {
		// 若非 JSON，則整個 value 以 "value" key 回傳
		raw["value"] = *out.SecretString
	}

	if len(keys) == 0 {
		zlogger.DebugContext(ctx, "aws return all keys from secret", zlogger.String("secret.path", path), zlogger.Int("returned_keys_count", len(raw)))
		return raw, nil
	}

	result := make(map[string]string, len(keys))
	for _, k := range keys {
		v, ok := raw[k]
		if !ok {
			return nil, fmt.Errorf("%w: key=%s at path=%s", domain.ErrSecretNotFound, k, path)
		}
		result[k] = v
	}
	zlogger.DebugContext(ctx, "aws return requested keys from secret", zlogger.String("secret.path", path), zlogger.Int("returned_keys_count", len(result)))
	return result, nil
}
