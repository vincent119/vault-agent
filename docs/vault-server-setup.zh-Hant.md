# Vault Server 設定：Kubernetes Auth / Policy / Role（UAT EKS 範例）

本篇為 **Vault Server 端**的設定步驟：在 Vault 上為特定 Kubernetes 叢集（例如 UAT EKS）啟用 Kubernetes Auth、建立 policy、建立 auth role。

> 英文版請見：[`docs/vault-server-setup.en.md`](vault-server-setup.en.md)

## 前置條件

- 已可使用 `kubectl` 連上目標叢集（例如 UAT EKS）
- 已可使用 `vault` CLI 連到 Vault（且有足夠權限：啟用 auth、寫 config/policy/role）
- 目標 namespace 與 service account 已存在（Vault 會用其 JWT 做 TokenReview）

## 0. 變數準備

```bash
# Kubernetes 端（要被 Vault 綁定的 SA）
export NAME_SPACE="vault-agent"         # namespace
export SA_NAME="vault-agent"            # service account name
export SA_SECRET_NAME="vault-agent-sa"  # SA token secret name（依你的環境調整）

# Vault 端（K8s auth mount path 與 role 名稱）
export K8S_NAME="uat-eks"      # K8s auth mount path（UAT EKS 範例）
export K8S_AUTH_ROLE="vault-agent"      # Vault role name

# 取得 SA JWT（TokenReview 用）
export SA_JWT_TOKEN="$(kubectl -n "$NAME_SPACE" get secret/"$SA_SECRET_NAME" --output 'go-template={{ .data.token }}' | base64 --decode)"

# 取得 Kubernetes API server 與 CA
export CA_CERT="$(kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.certificate-authority-data}' | base64 --decode)"
export K8S_SERVER="$(kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.server}')"
```

> 注意：不同 Kubernetes 版本 / 設定下，ServiceAccount token secret 的取得方式可能不同；請以你叢集的實際 SA token secret 為準。

## 1. 啟用 Kubernetes Auth（針對該叢集）

```bash
vault auth enable --path="$K8S_NAME" kubernetes

vault write "auth/$K8S_NAME/config" \
  token_reviewer_jwt="$SA_JWT_TOKEN" \
  kubernetes_host="$K8S_SERVER" \
  kubernetes_ca_cert="$CA_CERT"
```

此步驟完成後，Vault 會用 `token_reviewer_jwt` 去呼叫 Kubernetes TokenReview API 驗證 SA JWT。

## 2. 建立 Policy（授權可讀取哪些 secrets）

以下為範例（請依你的 KV v2 mount / path 調整）：

```bash
export VAULT_POLICY="vault-agent-uat-policy"

vault policy write "$VAULT_POLICY" - <<'EOF'
path "org/data/uat/*" {
  capabilities = ["read", "list"]
}

path "sys/health" {
  capabilities = ["read"]
}
EOF
```

## 3. 建立 Kubernetes Auth Role（綁定 SA/Namespace 與 policy）

```bash
vault write "auth/$K8S_NAME/role/$K8S_AUTH_ROLE" \
  bound_service_account_names="$SA_NAME" \
  bound_service_account_namespaces="$NAME_SPACE" \
  policies="$VAULT_POLICY" \
  ttl="1h"
```

Role 建立完成後，Vault-Agent 在 K8s 內可透過：

- `vault.auth_k8s_path: "$K8S_NAME"`
- `vault.auth_k8s_role: "$K8S_AUTH_ROLE"`

使用該 ServiceAccount 的 JWT 向 Vault 取得 token。

## 4. 與 vault-agent 的設定對應

在 vault-agent 的 `configs/config.yaml`（或 Deployment env）需設定：

- `VAULT_ADDRESS`
- `VAULT_AUTH_K8S_PATH`（對應上面的 `K8S_NAME`）
- `VAULT_AUTH_K8S_ROLE`（對應上面的 `K8S_AUTH_ROLE`）

> K8s 部署建議不要使用 `VAULT_TOKEN`（靜態 token），改用 Kubernetes Auth。

流程圖與架構圖請見：[架構圖與流程圖](architecture-diagrams.zh-Hant.md#2-vault-認證與取得-value-流程圖)。
