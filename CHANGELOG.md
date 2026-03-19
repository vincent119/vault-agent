# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.1.0/),
and this project aims to follow [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## Unreleased

## 0.1.0 - 2026-03-12

### Added

- DDD-style structure with webhook delivery, application use-cases, domain abstractions, and infra implementations.
- Mutating webhook `POST /mutate` with opt-in annotations and label-based filtering.
- Background sync worker to reconcile Kubernetes Secrets from Vault/AWS (enabled only when Kubernetes config is available).
- Vault KV v2 secret fetcher with optional Kubernetes Auth login.
- AWS Secrets Manager secret fetcher (JSON parsing with fallback to raw string).
- Observability:
  - Prometheus metrics with enable/disable switch and optional Basic Auth for `/metrics`
  - OpenTelemetry tracing with enable/disable switch and OTLP/stdout exporters
  - Structured logging via `zlogger` with request_id correlation
- Docker multi-stage build with pinned image digests; runs as non-root (UID/GID 10000) with timezone set.
- Formal documentation split into Traditional Chinese and English under `docs/`.

