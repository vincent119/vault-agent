# 架構圖與流程圖

- **架構圖**：vault-agent 與 Kubernetes、Vault、AWS 的關係。
- **流程圖**：Vault 認證方式與從 Vault 取得 secret value 的流程。

> 英文版請見：[`architecture-diagrams.en.md`](architecture-diagrams.en.md)

---

## 1. 整體架構圖

```mermaid
flowchart TB
    subgraph K8["Kubernetes 叢集"]
        Pod["使用者建立 Pod<br/>inject 註解"]
        API["API Server<br/>Mutating Webhook"]
        Pod -->|1. Admission 請求| API
    end

    API -->|2. 呼叫 Webhook| Agent

    subgraph Agent["vault-agent Pod"]
        Mutate["POST /mutate<br/>MutateUseCase"]
        Sync["Sync Worker 背景"]
        Mutate -->|SecretFetcher| Fetch
        Sync -->|SecretFetcher| Fetch
        Sync -->|List / Get / Update| K8API
    end

    Fetch["FetchSecret path, keys"]

    Fetch --> Vault
    Fetch --> AWS

    Vault["HashiCorp Vault<br/>K8s Auth 或 Token · KV v2"]

    AWS["AWS Secrets Manager<br/>path = secret name"]

    K8API["Kubernetes API<br/>Secret CRUD"]
```

---

## 2. Vault 認證與取得 Value 流程圖

以下為 **Vault 後端**的兩種認證方式，以及取得 secret 資料的流程。

### 2.1 流程概觀（Mermaid）

```mermaid
flowchart TB
    subgraph init["vault-agent 啟動時"]
        A[載入設定] --> B{有 VAULT_TOKEN?}
        B -->|是| C[NewVaultClient<br/>Token Auth]
        B -->|否| D{有 auth_k8s_path<br/>且 auth_k8s_role?}
        D -->|是| E[NewVaultClientWithK8sAuth]
        D -->|否| F[不啟用 Vault 後端]
        E --> G[讀取 Pod SA JWT<br/>/var/run/secrets/...]
        G --> H[Vault API<br/>auth/path/login]
        H --> I[Vault 以 TokenReview<br/>驗證 JWT]
        I --> J[回傳 ClientToken]
        J --> K[SetToken]
        C --> K
        K --> L[VaultClient 就緒]
    end

    subgraph fetch["取得 Secret Value（Mutate / Sync 時）"]
        L --> M[FetchSecret path, keys]
        M --> N[client.KVv2 mountPath.Get path]
        N --> O[Vault API GET<br/>/v1/mount/data/path]
        O --> P{keys 為空?}
        P -->|是| Q[回傳全部 key-value]
        P -->|否| R[只回傳指定 keys]
        Q --> S[map string string]
        R --> S
    end
```

### 2.2 Token 認證流程（簡化）

| 步驟 | 說明                                                                               |
| ---- | ---------------------------------------------------------------------------------- |
| 1    | 設定檔或環境變數提供 `vault.token`                                                 |
| 2    | `NewVaultClient(address, token, mountPath)` 建立 client 並 `SetToken(token)`       |
| 3    | 之後每次 `FetchSecret(ctx, path, keys)` 直接呼叫 Vault KV v2 API                   |
| 4    | `client.KVv2(mountPath).Get(ctx, path)` → 取得 `secret.Data`，再依 `keys` 篩選回傳 |

### 2.3 Kubernetes 認證流程（簡化）

| 步驟 | 說明                                                                                                 |
| ---- | ---------------------------------------------------------------------------------------------------- |
| 1    | 設定 `vault.auth_k8s_path`（Vault 內 K8s auth mount 路徑）、`vault.auth_k8s_role`（Vault role 名稱） |
| 2    | `NewVaultClientWithK8sAuth(ctx, address, mountPath, authPath, role)`                                 |
| 3    | SDK 從 Pod 內讀取 ServiceAccount JWT（預設 `/var/run/secrets/kubernetes.io/serviceaccount/token`）   |
| 4    | 呼叫 Vault `POST auth/{authPath}/login`，body 含 `role` 與 `jwt`                                     |
| 5    | Vault 使用該 auth 設定的 `token_reviewer_jwt` 呼叫 K8s TokenReview API 驗證此 JWT                    |
| 6    | 驗證通過後 Vault 依 role 綁定的 policy 簽發 `ClientToken`                                            |
| 7    | vault-agent 對 client `SetToken(ClientToken)`，之後與 Token 認證相同                                 |
| 8    | 之後 `FetchSecret(ctx, path, keys)` 使用此 token 呼叫 KV v2 API 取得 value                           |

### 2.4 時序概念（K8s Auth）

```mermaid
sequenceDiagram
    participant Agent as vault-agent Pod
    participant Vault as Vault Server
    participant K8s as Kubernetes API

    Agent->>Vault: POST auth/uat-eks/login<br/>{ role, jwt }
    Vault->>K8s: TokenReview(jwt)
    K8s-->>Vault: { uid, ... }
    Vault-->>Agent: { auth: { client_token } }
    Note over Agent: SetToken(client_token)

    Agent->>Vault: GET /v1/org/data/uat/xxx
    Vault-->>Agent: { data: { data: { key: "val" } } }
```

---

## 3. 與設定檔對應

| 認證方式   | 設定欄位                                                                          | 說明                                                                        |
| ---------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------- |
| Token      | `vault.address`, `vault.token`, `vault.mount_path`                                | 靜態 token，適合本機或測試                                                  |
| Kubernetes | `vault.address`, `vault.auth_k8s_path`, `vault.auth_k8s_role`, `vault.mount_path` | 生產建議，Vault 需先完成 [Vault Server 設定](vault-server-setup.zh-Hant.md) |

Vault Server 端需先啟用 Kubernetes Auth、撰寫 policy、建立 role 綁定 SA/namespace，vault-agent 的 K8s 認證才會成功。
