# 部署（Kubernetes / Kustomize）— 繁體中文

baseline manifests 位於 `deployments/kustomize/base/`：

```bash
kubectl apply -k deployments/kustomize/base
```

## 重點

- Webhook 設定：`failurePolicy: Ignore`（避免 webhook 異常阻塞 Pod CREATE）
- Webhook 過濾：`objectSelector` 只處理 `inject-vault-agent=true` 的 Pod
- 安全性：Deployment 預設以 non-root（UID/GID 10000）執行
- TLS：Deployment 掛載 `vault-agent-tls`，並以 `TLS_CERT_FILE` / `TLS_KEY_FILE` 啟用 HTTPS

## 設定注入

Deployment 透過環境變數注入設定（例如 `APP_ENV`、`APP_PORT`、`K8S_NAMESPACE`），以及透過 `envFrom.secretRef` 讀取敏感設定（例如 Vault/AWS 憑證）。

> 建議：敏感資訊（Vault token、AWS credentials、metrics basic auth 等）一律放 Secret；非敏感設定放 ConfigMap 或直接在 Deployment env 設定。
