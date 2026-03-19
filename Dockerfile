# syntax=docker/dockerfile:1

FROM alpine:3.22 AS prep
RUN addgroup -S -g 10000 vault-agent && \
    adduser  -S -u 10000 -G vault-agent -s /sbin/nologin vault-agent

FROM --platform=$BUILDPLATFORM cgr.dev/chainguard/go:latest-dev@sha256:fe9cf0af05cab0bc2b640f9f636713c7aabd64542e970a21db785fb1df31af98 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
# 依 TARGETARCH 建置（BuildKit/buildx 會依 --platform 注入）。預設 amd64；若節點是 arm64 請用 buildx --platform linux/arm64 或下方多架構指令
ARG TARGETOS=linux
ARG TARGETARCH=amd64
RUN CGO_ENABLED=0 GOOS=${TARGETOS} GOARCH=${TARGETARCH} go build \
      -trimpath \
      -ldflags="-s -w" \
      -o /vault-agent ./cmd/vault-agent/

FROM cgr.dev/chainguard/static:latest@sha256:2fdfacc8d61164aa9e20909dceec7cc28b9feb66580e8e1a65b9f2443c53b61b

COPY --from=prep /etc/passwd /etc/passwd
COPY --from=prep /etc/group  /etc/group


ENV TZ=Asia/Taipei

COPY --from=builder /vault-agent /vault-agent

USER vault-agent:vault-agent

ENTRYPOINT ["/vault-agent"]
