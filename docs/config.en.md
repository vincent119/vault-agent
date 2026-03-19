# Configuration — English

## Load precedence

Vault-Agent loads configuration in this order:

1. **OS environment variables**
2. `configs/config.yaml`
3. defaults

Examples:

- `configs/config.sample.yaml`
- `configs/.env.example`

## Config keys

### `app`

- `app.name`: service name (used for telemetry service name)
- `app.port`: HTTP server port (default 8080)
- `app.env`: `dev` / `prod` (affects log defaults)

### `log`

Maps to `github.com/vincent119/zlogger` `Config`:

- `log.level`: `debug/info/warn/error/fatal`
- `log.format`: `console/json`
- `log.outputs`: `[console]`, `["file"]`, or both
- `log.log_path`: directory for file output
- `log.file_name`: filename for file output (empty => date-based)
- `log.add_caller`
- `log.add_stacktrace`
- `log.development`
- `log.color_enabled`

When `app.env=prod`, Vault-Agent overrides some options to reduce risk of misconfiguration:

- forces `format=json`
- forces `development=false`
- forces `color_enabled=false`

### `vault`

Vault KV v2 config and authentication (choose one):

- `vault.address`
- `vault.mount_path` (default `secret`)
- **Static token** (local/dev): `vault.token`
- **Kubernetes auth** (in-cluster):
  - `vault.auth_k8s_path`
  - `vault.auth_k8s_role`

### `aws`

- `aws.region`

### `k8s`

- `k8s.kubeconfig`
  - empty: **in-cluster only** (no implicit `~/.kube/config`)
  - non-empty: use the provided kubeconfig file path
- `k8s.namespace`: restrict sync scan (empty => all namespaces)

### `sync`

- `sync.interval_seconds` (default 60)

### `telemetry`

- `telemetry.enabled`
- `telemetry.otlp_endpoint`: empty => stdout exporter; non-empty => OTLP exporter (gRPC or HTTP per `otlp_transport`)
- `telemetry.otlp_transport`: `grpc` (default, port 4317) or `http` (port 4318)
- `telemetry.otlp_compression`: `none` or `gzip`
- `telemetry.otlp_headers`: headers in `key1=value1,key2=value2` format (e.g. for auth)
- `telemetry.otlp_basic_auth`: Basic Auth for OTLP collector, format `user:password` (gRPC and HTTP)

### `metrics`

- `metrics.enabled`
- `metrics.basic_auth`: non-empty `user:password` enables HTTP Basic Auth on `/metrics`

### `tls`

To serve HTTPS:

- `tls.cert_file`
- `tls.key_file`
