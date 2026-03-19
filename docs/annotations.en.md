# Labels / Annotations (English)

This document describes the labels and annotations used by Vault-Agent for Pod injection and Secret sync.

## 1. Webhook filtering

### 1.1 Namespace label (required)

The mutating webhook uses `MutatingWebhookConfiguration.namespaceSelector` and only processes Pods in namespaces labeled with:

- `vault-agent.io/admission-webhooks: "enabled"`

Without this label on the namespace, the API Server will never forward requests to vault-agent regardless of what is set on individual Pods.

```bash
kubectl label namespace <your-namespace> vault-agent.io/admission-webhooks=enabled
```

Or manage it via a `namespace.yaml` in GitOps:

```yaml
apiVersion: v1
kind: Namespace
metadata:
  name: <your-namespace>
  labels:
    vault-agent.io/admission-webhooks: "enabled"
```

### 1.2 Pod label (required)

The mutating webhook also uses `MutatingWebhookConfiguration.objectSelector` and only processes Pods with:

- `inject-vault-agent: "true"`

Definition: `deployments/kustomize/base/webhook.yaml`.

> **Important**: This label must be placed under **`spec.template.metadata.labels`** (the Pod template), not the top-level `metadata.labels` of the Deployment. Placing it in the wrong location will not cause an error, but the webhook's `objectSelector` matches labels on the Pod object itself — Deployment labels are not inherited by Pods.

```yaml
spec:
  template:
    metadata:
      labels:
        app: myapp
        inject-vault-agent: "true"  # ← correct location
```

## 2. Pod injection (Pod annotations)

### 2.1 Enable injection

- `inject-vault-agent: "true"` (required)

### 2.2 Backend / path / keys

- `inject-vault-agent.backend`: `vault` (default) or `aws`
- `inject-vault-agent.path`: secret path (required)
  - Vault: KV v2 path (without mount)
  - AWS: Secrets Manager secret name / ARN (typically name)
- `inject-vault-agent.keys` (optional): JSON array string
  - example: `["DB_USER","DB_PASS"]`
  - empty/omitted: fetch all key-value pairs

### 2.3 Examples (Deployment / CronJob)

#### Example A: Deployment env injection (Vault)

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
        inject-vault-agent: "true" # webhook objectSelector filter
      annotations:
        inject-vault-agent: "true" # explicit opt-in (required)
        inject-vault-agent.backend: "vault"
        inject-vault-agent.path: "uat/myapp" # KV v2 path (without mount)
        inject-vault-agent.keys: '["DB_USER","DB_PASS"]'
    spec:
      containers:
        - name: myapp
          image: myrepo/myapp:1.0.0
```

#### Example B: CronJob env injection (AWS)

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

## 3. Secret sync (Secret label / annotations)

The sync worker lists Secrets labeled with `inject-vault-agent=true` and uses the same annotations (backend/path/keys) to fetch and overwrite `Secret.Data`.
