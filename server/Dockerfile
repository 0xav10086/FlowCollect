# ── Builder: 纯静态编译 Go 二进制 ──
FROM golang:1.25-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download

COPY *.go ./
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 \
    go build -ldflags="-s -w" -o /flow_server .

# ── Runner: 最小运行时镜像 ──
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata wget \
    && adduser -D -u 1000 appuser

WORKDIR /app
COPY --from=builder /flow_server ./flow_server

# 运行时目录由 ensureDirs() 自动创建，外部资源通过 Volume 挂载
# configs/  templates/  data/  logs/ 均需从宿主机注入

EXPOSE 8686

HEALTHCHECK --interval=30s --timeout=5s --retries=3 \
    CMD wget -qO- http://localhost:8686/health || exit 1

USER appuser
CMD ["./flow_server"]
