# 可觀測性（Observability）— 繁體中文

## Metrics（Prometheus）

- `/metrics` 預設啟用，可用 `metrics.enabled` 關閉
- 可選 Basic Auth：
  - `metrics.basic_auth` 非空（格式 `user:password`）時，`/metrics` 需要 HTTP Basic Auth

### 重要指標

指標定義位於 `internal/infra/metrics/metrics.go`（名稱可能包含）：

- `vault_agent_mutate_requests_total{status="success|error"}`
- `vault_agent_sync_errors_total{source="vault|aws"}`
- `vault_agent_secret_fetch_duration_seconds{backend="vault|aws"}`

### Prometheus 整合 (ServiceMonitor)

如果您在 Kubernetes 中使用了 Prometheus Operator，您可以透過部署 `ServiceMonitor` 來自動抓取 Vault Agent 所暴露的 metrics 數據：

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

### Grafana 儀表板

您可以在專案中找到預先建置好的 Grafana Dashboard JSON 檔案，用來即時監控 Vault Agent 運行狀況。

**檔案位置：**
`docs/Grafana/vault-agent-dashbord.json`

**使用方式：**

1. 開啟您的 Grafana 網頁介面。
2. 前往左側選單的 **Dashboards** > **Import**。
3. 點選 Upload JSON file 或貼上檔案內容。
4. 選擇您環境中的 Prometheus 作為資料來源（Data Source）後儲存即可。

## Tracing（OpenTelemetry）

- `telemetry.enabled`：開關
- `telemetry.otlp_endpoint`：
  - 空字串：stdout exporter（開發用）
  - 非空：依 `otlp_transport` 使用 **gRPC**（預設，port 4317）或 **HTTP**（port 4318）
- `telemetry.otlp_transport`：`grpc`（預設）或 `http`
- `telemetry.otlp_compression`：`none`（預設）或 `gzip`，對應 [OpenTelemetry Collector configgrpc](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc) 的 compression
- `telemetry.otlp_headers`：認證或自訂 header，格式 `key1=value1,key2=value2`（與 `OTEL_EXPORTER_OTLP_HEADERS` 一致），例如 `Authorization=Bearer <token>`
- `telemetry.otlp_basic_auth`：對 OTLP 收集器使用 **HTTP Basic 認證**（gRPC 與 HTTP 皆支援），格式 `user:password`；對應 [configauth](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configauth) 的 Client Basic Auth

## Logging（zlogger）

- `log.*` 由 `configs/config.yaml` 或環境變數 `LOG_*` 載入
- Webhook 入口會產生 `X-Request-ID`，並注入到 context；delivery/application/infra 層使用 `zlogger.*Context` 會自動帶入 request_id
