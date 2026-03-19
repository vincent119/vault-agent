# Observability — English

## Metrics (Prometheus)

- `/metrics` is enabled by default, can be disabled via `metrics.enabled`
- Optional Basic Auth:
  - when `metrics.basic_auth` is non-empty (`user:password`), `/metrics` requires HTTP Basic Auth

### Key metrics

Metrics are defined in `internal/infra/metrics/metrics.go`, including (names may include):

- `vault_agent_mutate_requests_total{status="success|error"}`
- `vault_agent_sync_errors_total{source="vault|aws"}`
- `vault_agent_secret_fetch_duration_seconds{backend="vault|aws"}`

### Prometheus Integration (ServiceMonitor)

If you are using the Prometheus Operator in your Kubernetes cluster, you can deploy a `ServiceMonitor` to automatically scrape the metrics exposed by the Vault Agent.

```yaml
  - job_name: "vault-agent"
    scheme: https
    tls_config:
      insecure_skip_verify: true
    metrics_path: "/metrics"
    scrape_interval: 30s
    scrape_timeout: 10s
    static_configs:
      - targets: ["vault-agent.vault-agent.svc.cluster.local:8080"]
        labels:
          instance: uat-vault-agent
          job: vault-agent
          env: uat
    # If basic_auth is enabled in config, provide the credentials secret here:
    # basicAuth:
    #   username:
    #     name: vault-agent-metrics-auth
    #     key: username
    #   password:
    #     name: vault-agent-metrics-auth
    #     key: password
```

### Grafana Dashboard

A pre-configured Grafana Dashboard JSON is included in this repository to help you visualize the Vault Agent metrics instantly.

You can find the dashboard template at:
`docs/Grafana/vault-agent-dashbord.json`

To use it:

1. Open your Grafana UI.
2. Navigate to **Dashboards** > **Import**.
3. Upload the JSON file or paste its content.
4. Select your Prometheus data source when prompted.

## Tracing (OpenTelemetry)

- `telemetry.enabled`: toggle
- `telemetry.otlp_endpoint`:
  - empty: stdout exporter (dev)
  - non-empty: OTLP exporter via **gRPC** (default, port 4317) or **HTTP** (port 4318), depending on `otlp_transport`
- `telemetry.otlp_transport`: `grpc` (default) or `http`
- `telemetry.otlp_compression`: `none` (default) or `gzip`; aligns with [OpenTelemetry Collector configgrpc](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc)
- `telemetry.otlp_headers`: auth or custom headers, format `key1=value1,key2=value2` (same as `OTEL_EXPORTER_OTLP_HEADERS`), e.g. `Authorization=Bearer <token>`
- `telemetry.otlp_basic_auth`: **HTTP Basic Auth** to the OTLP collector (both gRPC and HTTP); format `user:password`; aligns with [configauth](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configauth) Client Basic Auth

## Logging (zlogger)

- `log.*` is loaded from `configs/config.yaml` or `LOG_*` env vars
- the webhook sets `X-Request-ID` and injects request_id into context; logs written via `zlogger.*Context` will include the correlated fields
