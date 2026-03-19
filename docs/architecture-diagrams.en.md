# Architecture and Flow Diagrams

- **Architecture diagram**: How vault-agent relates to Kubernetes, Vault, and AWS.
- **Flow diagrams**: Vault authentication and fetching secret values.

> 繁體中文版請見：[`architecture-diagrams.zh-Hant.md`](architecture-diagrams.zh-Hant.md)

---

## 1. Overall Architecture

```mermaid
flowchart TB
    subgraph K8["Kubernetes Cluster"]
        Pod["User creates Pod<br/>inject annotations"]
        API["API Server<br/>Mutating Webhook"]
        Pod -->|1. Admission request| API
    end

    API -->|2. Call Webhook| Agent

    subgraph Agent["vault-agent Pod"]
        Mutate["POST /mutate<br/>MutateUseCase"]
        Sync["Sync Worker background"]
        Mutate -->|SecretFetcher| Fetch
        Sync -->|SecretFetcher| Fetch
        Sync -->|List / Get / Update| K8API
    end

    Fetch["FetchSecret path, keys"]

    Fetch --> Vault
    Fetch --> AWS

    Vault["HashiCorp Vault<br/>K8s Auth or Token · KV v2"]

    AWS["AWS Secrets Manager<br/>path = secret name"]

    K8API["Kubernetes API<br/>Secret CRUD"]
```

---

## 2. Vault Authentication and Fetching Values

Two authentication methods for the **Vault** backend and how secret values are retrieved.

### 2.1 Flow Overview (Mermaid)

```mermaid
flowchart TB
    subgraph init["On vault-agent startup"]
        A[Load config] --> B{VAULT_TOKEN set?}
        B -->|Yes| C[NewVaultClient<br/>Token Auth]
        B -->|No| D{auth_k8s_path &<br/>auth_k8s_role set?}
        D -->|Yes| E[NewVaultClientWithK8sAuth]
        D -->|No| F[Vault backend disabled]
        E --> G[Read Pod SA JWT<br/>/var/run/secrets/...]
        G --> H[Vault API<br/>auth/path/login]
        H --> I[Vault calls TokenReview<br/>to validate JWT]
        I --> J[Return ClientToken]
        J --> K[SetToken]
        C --> K
        K --> L[VaultClient ready]
    end

    subgraph fetch["Fetch Secret Value (Mutate / Sync)"]
        L --> M[FetchSecret path, keys]
        M --> N[client.KVv2 mountPath.Get path]
        N --> O[Vault API GET<br/>/v1/mount/data/path]
        O --> P{keys empty?}
        P -->|Yes| Q[Return all key-values]
        P -->|No| R[Return only specified keys]
        Q --> S[map string string]
        R --> S
    end
```

### 2.2 Token Auth Flow (Summary)

| Step | Description                                                               |
| ---- | ------------------------------------------------------------------------- |
| 1    | Config or env provides `vault.token`                                      |
| 2    | `NewVaultClient(address, token, mountPath)` and `SetToken(token)`         |
| 3    | Each `FetchSecret(ctx, path, keys)` calls Vault KV v2 API with that token |
| 4    | `client.KVv2(mountPath).Get(ctx, path)` → filter by `keys` and return     |

### 2.3 Kubernetes Auth Flow (Summary)

| Step | Description                                                                                               |
| ---- | --------------------------------------------------------------------------------------------------------- |
| 1    | Set `vault.auth_k8s_path` (Vault K8s auth mount) and `vault.auth_k8s_role` (Vault role name)              |
| 2    | `NewVaultClientWithK8sAuth(ctx, address, mountPath, authPath, role)`                                      |
| 3    | SDK reads ServiceAccount JWT from the Pod (default `/var/run/secrets/kubernetes.io/serviceaccount/token`) |
| 4    | Call Vault `POST auth/{authPath}/login` with `role` and `jwt`                                             |
| 5    | Vault uses that auth backend's `token_reviewer_jwt` to call Kubernetes TokenReview API                    |
| 6    | On success, Vault issues `ClientToken` according to the role's policies                                   |
| 7    | vault-agent calls `SetToken(ClientToken)`; from then on, same as token auth                               |
| 8    | `FetchSecret(ctx, path, keys)` uses this token to call KV v2 API and return values                        |

### 2.4 Sequence (K8s Auth)

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

## 3. Config Mapping

| Auth method | Config fields                                                                     | Notes                                                                                               |
| ----------- | --------------------------------------------------------------------------------- | --------------------------------------------------------------------------------------------------- |
| Token       | `vault.address`, `vault.token`, `vault.mount_path`                                | Static token; suitable for local or test                                                            |
| Kubernetes  | `vault.address`, `vault.auth_k8s_path`, `vault.auth_k8s_role`, `vault.mount_path` | Recommended for production; Vault must be set up per [Vault Server Setup](vault-server-setup.en.md) |

Vault must have Kubernetes auth enabled, policies written, and a role bound to the service account/namespace before vault-agent's Kubernetes auth can succeed.
