# Vault Server Setup: Kubernetes Auth / Policy / Role (UAT EKS example)

This document describes the **Vault server-side** steps to enable Kubernetes Auth for a specific Kubernetes cluster (e.g., UAT EKS), create a policy, and create a Kubernetes auth role.

> Traditional Chinese version: [`docs/vault-server-setup.zh-Hant.md`](vault-server-setup.zh-Hant.md)

## Prerequisites

- `kubectl` can access the target cluster (e.g., UAT EKS)
- `vault` CLI is authenticated with sufficient privileges (enable auth, write config/policy/role)
- target namespace and service account already exist

## 0. Prepare variables

```bash
# Kubernetes side (the SA that Vault will bind to)
export NAME_SPACE="vault-agent"
export SA_NAME="vault-agent"
export SA_SECRET_NAME="vault-agent-sa"  # adjust to your environment

# Vault side (K8s auth mount path and role name)
export K8S_NAME="uat-eks"      # mount path for this cluster (UAT example)
export K8S_AUTH_ROLE="vault-agent"      # Vault role name

# SA JWT used for TokenReview
export SA_JWT_TOKEN="$(kubectl -n "$NAME_SPACE" get secret/"$SA_SECRET_NAME" --output 'go-template={{ .data.token }}' | base64 --decode)"

# Kubernetes API server and CA
export CA_CERT="$(kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.certificate-authority-data}' | base64 --decode)"
export K8S_SERVER="$(kubectl config view --raw --minify --flatten --output='jsonpath={.clusters[].cluster.server}')"
```

Note: how ServiceAccount token secrets are exposed depends on your Kubernetes version/settings. Use the token secret that exists in your cluster.

## 1. Enable Kubernetes auth for the cluster

```bash
vault auth enable --path="$K8S_NAME" kubernetes

vault write "auth/$K8S_NAME/config" \
  token_reviewer_jwt="$SA_JWT_TOKEN" \
  kubernetes_host="$K8S_SERVER" \
  kubernetes_ca_cert="$CA_CERT"
```

## 2. Create a policy

Example (adjust KV v2 mount/path to your environment):

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

## 3. Create a Kubernetes auth role

```bash
vault write "auth/$K8S_NAME/role/$K8S_AUTH_ROLE" \
  bound_service_account_names="$SA_NAME" \
  bound_service_account_namespaces="$NAME_SPACE" \
  policies="$VAULT_POLICY" \
  ttl="1h"
```

After this, vault-agent can authenticate in-cluster by setting:

- `vault.auth_k8s_path: "$K8S_NAME"`
- `vault.auth_k8s_role: "$K8S_AUTH_ROLE"`

## 4. Map to vault-agent config

In `configs/config.yaml` (or Deployment env), configure:

- `VAULT_ADDRESS`
- `VAULT_AUTH_K8S_PATH` (same as `K8S_NAME`)
- `VAULT_AUTH_K8S_ROLE` (same as `K8S_AUTH_ROLE`)

Recommendation: for Kubernetes deployments, prefer Kubernetes auth over static `VAULT_TOKEN`.

For flow and sequence diagrams, see [Architecture & flow diagrams](architecture-diagrams.en.md#2-vault-authentication-and-fetching-values).
