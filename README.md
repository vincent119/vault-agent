# Vault-Agent

> Kubernetes 密鑰同步控制器 · A Kubernetes Secret Sync Controller

[![GitHub](https://img.shields.io/badge/github-vincent119%2Fvault--agent-blue?logo=github)](https://github.com/vincent119/vault-agent)
![Go](https://img.shields.io/badge/Go-1.25-00ADD8?logo=go&logoColor=white)
![License](https://img.shields.io/badge/License-MIT-blue)
![Architecture](https://img.shields.io/badge/Architecture-DDD%20%2B%20Clean-lightgrey)
![GitHub Release](https://img.shields.io/github/v/tag/vincent119/vault-agent?label=release)

---

## 概覽 | Overview

Vault-Agent 是一個以 **Go** 重新打造的 Kubernetes 密鑰同步控制器，遵循 **DDD（領域驅動設計）+ Clean Architecture** 原則，不依賴 DI 框架，完全使用標準函式庫 `net/http`。

它解決了「應用程式部署時如何安全取得最新密鑰」的問題：不僅在 Pod 建立當下注入密鑰，也能在背景主動確保 Kubernetes Secret 與遠端密鑰儲存的資料一致。

Vault-Agent is a **Go**-based Kubernetes secret sync controller following **DDD + Clean Architecture**, with no DI framework and a standard `net/http` server. It solves the problem of securely delivering up-to-date secrets to applications — both at Pod creation time and continuously in the background.

---

## 主要功能 | Key Features

| 功能 | 說明 |
| ------ | ------ |
| **Mutating Webhook** | `POST /mutate` — 解析 AdmissionReview，回傳 JSON Patch 將密鑰注入 Pod 環境變數 |
| **背景同步 Worker** | 定期掃描帶有標籤的 Kubernetes Secret，主動從遠端後端覆寫更新，確保資料一致 |
| **多密鑰後端** | 同時支援 **HashiCorp Vault**（KV v2，Token / K8s Auth）與 **AWS Secrets Manager** |
| **可觀測性** | Prometheus `/metrics`、可選 OTLP gRPC Tracing、結構化日誌（zlogger） |
| **優雅關機** | 監聽 `SIGINT`/`SIGTERM`，30 秒 shutdown timeout，安全排空請求後退出 |
| **安全執行** | 以 non-root（UID/GID 10000）執行，Webhook `failurePolicy: Ignore` 確保不影響叢集穩定性 |

---

## 架構 | Architecture

採 DDD 標準分層，依賴在 `cmd/vault-agent/main.go` 手動組裝：

```bash
vault-agent/
├── cmd/vault-agent/main.go        # 進入點：手動組裝各層、啟動 Server
├── internal/
│   ├── configs/                   # 設定載入（viper，支援 OS Env > config.yaml > 預設值）
│   ├── syncer/
│   │   ├── domain/                # 領域層：SecretFetcher 介面、SecretRef、Domain Errors
│   │   ├── application/           # 應用層：MutateUseCase、SyncWorkerUseCase
│   │   ├── infra/                 # 實作層：Vault Client、AWS Client、K8s Repository
│   │   └── delivery/              # 介面層：Webhook HTTP Handler
│   └── infra/
│       ├── logger/                # zlogger 封裝
│       ├── telemetry/             # OTLP Tracing
│       └── metrics/               # Prometheus 指標
├── deployments/kustomize/         # Kubernetes manifests（base + overlays）
├── configs/config.sample.yaml     # 設定檔範例
└── Dockerfile                     # Multi-stage build（Chainguard distroless）
```

**HTTP 端點：**

| 端點 | 說明 |
| ------ | ------ |
| `POST /mutate` | Kubernetes Mutating Webhook |
| `GET /healthz` | Liveness / Readiness Probe |
| `GET /metrics` | Prometheus metrics（可加 Basic Auth） |

---

## 快速開始 | Quick Start

**本機執行：**

```bash
cp configs/config.sample.yaml configs/config.yaml
# 編輯 config.yaml 填入 Vault / AWS 設定
make run
```

**執行測試：**

```bash
make test
```

**部署至 Kubernetes：**

```bash
kubectl apply -k deployments/kustomize/base
```

> 若要啟用背景同步 Worker，需提供 `k8s.kubeconfig`（或以 In-Cluster 方式執行）；否則僅啟動 HTTP 端點。

---

## 文件 | Documentation

| 語言    | 入口   |
|--------|-------|
| 繁體中文 | [`docs/README.zh-Hant.md`](docs/README.zh-Hant.md) |
| English | [`docs/README.en.md`](docs/README.en.md) |

詳細文件：

- [設定說明](docs/config.zh-Hant.md) / [Configuration](docs/config.en.md)
- [Annotation / Label 規範](docs/annotations.zh-Hant.md) / [Annotations](docs/annotations.en.md)
- [可觀測性](docs/observability.zh-Hant.md) / [Observability](docs/observability.en.md)
- [部署指南](docs/deploy.zh-Hant.md) / [Deployment](docs/deploy.en.md)
- [Vault Server 設定](docs/vault-server-setup.zh-Hant.md) / [Vault Server Setup](docs/vault-server-setup.en.md)
- [AWS Secrets Manager](docs/aws-secrets-manager.zh-hant.md) / [AWS Secrets Manager](docs/aws-secrets-manager.en.md)
- [架構圖](docs/architecture-diagrams.zh-Hant.md) / [Architecture Diagrams](docs/architecture-diagrams.en.md)
