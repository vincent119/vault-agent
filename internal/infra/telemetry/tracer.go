// Package telemetry 提供 OpenTelemetry 初始化，用於追踪 Vault Agent 的運行狀態。
package telemetry

import (
	"context"
	"encoding/base64"
	"strings"

	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracegrpc"
	"go.opentelemetry.io/otel/exporters/otlp/otlptrace/otlptracehttp"
	"go.opentelemetry.io/otel/exporters/stdout/stdouttrace"
	"go.opentelemetry.io/otel/sdk/resource"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	semconv "go.opentelemetry.io/otel/semconv/v1.17.0"

	"vault-agent/internal/configs"
)

// parseOTLPHeaders 將 "key1=value1,key2=value2" 解析為 map；空字串回傳 nil。
func parseOTLPHeaders(s string) map[string]string {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	out := make(map[string]string)
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		idx := strings.Index(part, "=")
		if idx <= 0 {
			continue
		}
		k := strings.TrimSpace(part[:idx])
		v := strings.TrimSpace(part[idx+1:])
		if k != "" {
			out[k] = v
		}
	}
	if len(out) == 0 {
		return nil
	}
	return out
}

// buildOTLPHeaders 合併 otlp_headers 與 otlp_basic_auth；若 basicAuth 非空則設定 Authorization: Basic base64(user:password)（覆寫 headers 內同 key）。
func buildOTLPHeaders(headersStr, basicAuth string) map[string]string {
	headers := parseOTLPHeaders(headersStr)
	if basicAuth != "" {
		if headers == nil {
			headers = make(map[string]string)
		}
		headers["Authorization"] = "Basic " + base64.StdEncoding.EncodeToString([]byte(strings.TrimSpace(basicAuth)))
	}
	return headers
}

// InitTracer 初始化全域 TracerProvider。回傳一個 Shutdown 函式供 main 結束前呼叫。
// 若 cfg.OTLPEndpoint 為空則使用 stdout exporter；否則依 cfg.OTLPTransport 使用 gRPC 或 HTTP，
// 並依 cfg.OTLPCompression、cfg.OTLPHeaders 設定壓縮與認證 header（對應 [configgrpc](https://github.com/open-telemetry/opentelemetry-collector/tree/main/config/configgrpc)）。
func InitTracer(ctx context.Context, serviceName string, cfg *configs.TelemetryConfig) (func(context.Context) error, error) {
	var exporter sdktrace.SpanExporter
	var err error

	if cfg == nil || cfg.OTLPEndpoint == "" {
		exporter, err = stdouttrace.New(stdouttrace.WithPrettyPrint())
		if err != nil {
			return nil, err
		}
	} else {
		headers := buildOTLPHeaders(cfg.OTLPHeaders, cfg.OTLPBasicAuth)
		transport := strings.ToLower(strings.TrimSpace(cfg.OTLPTransport))
		if transport == "" {
			transport = "grpc"
		}
		compression := strings.ToLower(strings.TrimSpace(cfg.OTLPCompression))

		if transport == "http" {
			opts := []otlptracehttp.Option{
				otlptracehttp.WithEndpoint(cfg.OTLPEndpoint),
				otlptracehttp.WithInsecure(),
			}
			if compression == "gzip" {
				opts = append(opts, otlptracehttp.WithCompression(otlptracehttp.GzipCompression))
			}
			if len(headers) > 0 {
				opts = append(opts, otlptracehttp.WithHeaders(headers))
			}
			exporter, err = otlptracehttp.New(ctx, opts...)
		} else {
			// gRPC（預設）
			opts := []otlptracegrpc.Option{
				otlptracegrpc.WithEndpoint(cfg.OTLPEndpoint),
				otlptracegrpc.WithInsecure(),
			}
			if compression == "gzip" {
				opts = append(opts, otlptracegrpc.WithCompressor("gzip"))
			}
			if len(headers) > 0 {
				opts = append(opts, otlptracegrpc.WithHeaders(headers))
			}
			exporter, err = otlptracegrpc.New(ctx, opts...)
		}
		if err != nil {
			return nil, err
		}
	}

	res, err := resource.New(ctx,
		resource.WithAttributes(
			semconv.ServiceNameKey.String(serviceName),
		),
	)
	if err != nil {
		return nil, err
	}

	tp := sdktrace.NewTracerProvider(
		sdktrace.WithBatcher(exporter),
		sdktrace.WithResource(res),
	)
	otel.SetTracerProvider(tp)
	return tp.Shutdown, nil
}
