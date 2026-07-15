# Dnskeeper

Dnskeeper 是 [CoreDNS](https://coredns.io) `etcd` 插件的 DNS 管理工具。

## 快速开始

### Docker Compose

```bash
docker compose up --build
```

服务默认监听 `:8080`,内置管理员 `admin` / `admin123`(首次启动自动创建)。

### 二进制

```bash
go build -o dnskeeper ./cmd/dnskeeper
./dnskeeper -config ./config.toml
```

## 配置

最小配置见 `config.example.toml`:

```toml
[server]
listen = ":8080"

[log]
level = "info"

[etcd]
endpoints = ["127.0.0.1:2379"]
# username / password / cert / key / ca 可选

[coredns]
path = "/skydns"          # 须与 CoreDNS Corefile 的 etcd path 一致

[jwt]
secret = "change-me"      # 生产环境务必修改
access_ttl  = "30m"
refresh_ttl = "168h"
```

> 运行依赖:dnskeeper 与 CoreDNS 必须指向同一 etcd 实例,且 `coredns.path` 与 Corefile 的 `etcd path` 保持一致。

## License

[Apache License 2.0](LICENSE)
