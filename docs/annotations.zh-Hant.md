# Annotation / Label 規格（繁體中文）

本文件描述 Vault-Agent 的注入與同步規格（Pod / Secret 上的 label 與 annotation）。

## 1. Webhook 過濾（Label）

### 1.1 Namespace Label（必填）

Mutating Webhook 透過 `MutatingWebhookConfiguration.namespaceSelector` 過濾請求，只處理帶有以下 label 的 Namespace 內的 Pod：

- `vault-agent.io/admission-webhooks: "enabled"`

若 Namespace 上沒有此 label，API Server 不會將任何請求轉發給 vault-agent，無論 Pod 本身如何設定都無效。

```bash
kubectl label namespace <your-namespace> vault-agent.io/admission-webhooks=enabled
```

或以 GitOps 方式管理，新增 `namespace.yaml`：

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: <your-namespace>
  labels:
    vault-agent.io/admission-webhooks: "enabled"
```

### 1.2 Pod Label（必填）

Mutating Webhook 同時透過 `MutatingWebhookConfiguration.objectSelector` 過濾請求，只處理符合以下 label 的 Pod：

- `inject-vault-agent: "true"`

> 定義位置：`deployments/kustomize/base/webhook.yaml`
> **注意**：此 label 必須加在 **`spec.template.metadata.labels`**（Pod template），而非 Deployment 頂層的 `metadata.labels`。加在錯誤位置不會報錯，但 webhook 的 `objectSelector` 比對的是 Pod 物件本身的 label，Deployment 的 label 不會繼承到 Pod 上。

```yaml
spec:
  template:
    metadata:
      labels:
        app: myapp
        inject-vault-agent: "true"  # ← 正確位置
```

## 2. Pod 注入（Annotations）

### 2.1 啟用注入

- `inject-vault-agent: "true"`（必填）

### 2.2 指定 backend / path / keys

- `inject-vault-agent.backend`：`vault`（預設）或 `aws`
- `inject-vault-agent.path`：機密路徑（必填）
  - Vault：KV v2 的 path（不含 mount）
  - AWS：Secrets Manager 的 secret name / ARN（實務上通常用 name）
- `inject-vault-agent.keys`（可選）：JSON array 字串
  - 例：`["DB_USER","DB_PASS"]`
  - 空/未填：取全部 key-value

### 2.3 範例（Deployment / CronJob）

#### 範例 A：Deployment 注入 env（Vault）

```yaml
apiVersion: apps/v1
kind: Deployment
metadata:
  name: myapp
  namespace: myns
spec:
  selector:
    matchLabels:
      app: myapp
  template:
    metadata:
      labels:
        app: myapp
        inject-vault-agent: "true" # webhook objectSelector 過濾
      annotations:
        inject-vault-agent: "true" # 明確 opt-in（必填）
        inject-vault-agent.backend: "vault"
        inject-vault-agent.path: "uat/myapp" # KV v2 path（不含 mount）
        inject-vault-agent.keys: '["DB_USER","DB_PASS"]'
    spec:
      containers:
        - name: myapp
          image: myrepo/myapp:1.0.0
```

#### 範例 B：CronJob 注入 env（AWS）

```yaml
apiVersion: batch/v1
kind: CronJob
metadata:
  name: nightly-job
  namespace: myns
spec:
  schedule: "0 2 * * *"
  jobTemplate:
    spec:
      template:
        metadata:
          labels:
            inject-vault-agent: "true"
          annotations:
            inject-vault-agent: "true"
            inject-vault-agent.backend: "aws"
            inject-vault-agent.path: "uat/myapp/cronjob-secret" # secret name / arn
        spec:
          serviceAccountName: my-sa
          restartPolicy: Never
          containers:
            - name: job
              image: myrepo/job:1.0.0
```

## 3. Secret 同步（Secret 的 label / annotations）

背景同步 Worker 會掃描帶有 label `inject-vault-agent=true` 的 Secret，並使用 **同一組 annotations**（backend/path/keys）重新拉取並覆寫 `Secret.Data`。

> Sync Worker 仍以 label 進行掃描過濾（見程式碼內 selector：`inject-vault-agent=true`）。
