# ---- 前端构建 ----
FROM node:22-alpine AS web-builder
WORKDIR /web
COPY web/package.json web/package-lock.json ./
RUN npm ci
COPY web/ .
RUN npm run build

# ---- 后端构建 ----
FROM golang:1.26 AS builder
WORKDIR /src
COPY go.mod go.sum ./
RUN go mod download
COPY . .
# 嵌入前端构建产物(go:embed all:dist)
COPY --from=web-builder /web/dist ./web/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /dnskeeper ./cmd/dnskeeper

FROM gcr.io/distroless/static:nonroot
COPY --from=builder /dnskeeper /dnskeeper
COPY config.example.toml /config.toml
EXPOSE 8080
ENTRYPOINT ["/dnskeeper", "-config", "/config.toml"]
