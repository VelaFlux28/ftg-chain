# FTG Energy Chain — Docker Build
# Multi-stage build for the ftgd blockchain daemon

# Stage 1: Build
FROM golang:1.22-alpine AS builder

RUN apk add --no-cache make gcc musl-dev linux-headers git

WORKDIR /app
COPY go.mod go.sum* ./
RUN go mod download 2>/dev/null || true

COPY . .
RUN go build -o /ftgd ./cmd/ftgd/

# Stage 2: Runtime
FROM alpine:3.19

RUN apk add --no-cache ca-certificates curl jq

COPY --from=builder /ftgd /usr/local/bin/ftgd

# Default ports
# 26656 - P2P
# 26657 - RPC
# 1317  - REST API
# 9090  - gRPC
# 9091  - gRPC Web
# 26660 - Prometheus metrics
EXPOSE 26656 26657 1317 9090 9091 26660

# Default data directory
VOLUME ["/root/.ftgd"]

ENTRYPOINT ["ftgd"]
CMD ["start"]
