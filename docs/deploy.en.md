# Deployment (Kubernetes / Kustomize) — English

Baseline manifests live in `deployments/kustomize/base/`:

```bash
kubectl apply -k deployments/kustomize/base
```

## Notes

- Webhook: `failurePolicy: Ignore` to avoid blocking Pod CREATE on failures
- Filtering: `objectSelector` processes only Pods labeled `inject-vault-agent=true`
- Security: runs as non-root by default (UID/GID 10000)
- TLS: Deployment mounts `vault-agent-tls` and enables HTTPS via `TLS_CERT_FILE` / `TLS_KEY_FILE`

## Configuration injection

The Deployment injects config via env vars (e.g. `APP_ENV`, `APP_PORT`, `K8S_NAMESPACE`) and sensitive values via `envFrom.secretRef`.

Recommendation: keep secrets in Kubernetes Secret (Vault token, AWS creds, metrics basic auth, etc.) and non-sensitive settings in ConfigMap or Deployment env.
