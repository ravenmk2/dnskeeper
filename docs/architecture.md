---
date: 2026-07-14
---

# 架构设计

## 1. 概述

dnskeeper 是 CoreDNS etcd 插件的管理后端，与 CoreDNS 共用同一 etcd 实例：领域对象存于 `/dnskeeper/` 前缀，CoreDNS 服务记录存于 `/skydns/` 前缀。领域模型与 Key 规划见 [数据存储设计](design/data-storage.md)，同步规则见 [DNS 数据同步设计](design/dns-sync.md)，API 约定见 [API 概览](api/overview.md)。本文聚焦工程层面：技术栈、分层、错误模型、请求生命周期、配置、启动、测试与部署。开发语言为 Go。

## 2. 技术栈

| 类别        | 选型                                                    | 说明                       |
| ----------- | ------------------------------------------------------- | -------------------------- |
| 语言        | Go 1.25+                                                | 基线版本                   |
| Web 框架    | Echo                                                    | JSON 绑定+校验、中间件     |
| etcd 客户端 | `go.etcd.io/etcd/client/v3`                             | 官方 v3                    |
| JWT         | `github.com/golang-jwt/jwt/v5`                          | HS256，access+refresh 轮换 |
| 密码哈希    | `golang.org/x/crypto/bcrypt`                            | bcrypt                     |
| 校验        | `github.com/go-playground/validator`                    | 标签驱动，Echo `Bind` 触发 |
| 日志        | `github.com/sirupsen/logrus`                            | 结构化                     |
| 配置        | `github.com/BurntSushi/toml`                            | TOML 文件                  |
| 测试        | `github.com/stretchr/testify` + `go.etcd.io/etcd/embed` | 嵌入式 etcd 集成测试       |
| DI          | 手写构造器                                              | 不引入 wire/dig            |

## 3. 项目结构

```txt
cmd/dnskeeper/main.go        # 最小入口
internal/
├── app/                     # 依赖装配 + Echo 初始化
├── apperr/                  # AppError + 错误码表
├── config/                  # TOML 解析
├── envelope/                # 响应信封封装
├── handler/                 # HTTP 层
├── jwt/                     # HS256 签发/校验
├── log/                     # logrus 装配
├── middleware/              # request-id/logger/recover/auth
├── service/                 # 业务层
└── store/                   # etcd 封装
Dockerfile                   # 多阶段构建
docker-compose.yml           # dev: etcd + dnskeeper
```

包内按领域对象分文件（user/zone/domain/record/auth/health）。

## 4. 分层与职责

三层 + 按层分包，依赖单向流动 `handler → service → store`。

| 层      | 包               | 职责                                                                                        | 不做什么          |
| ------- | ---------------- | ------------------------------------------------------------------------------------------- | ----------------- |
| handler | internal/handler | JSON 绑定（触发 validator）、调 service、写信封                                             | 不含业务规则      |
| service | internal/service | Txn 组装、record-id 分配、嵌套冲突预检、级联删除、随路同步写入、登录/改密、密码强度         | 不直接碰 etcd API |
| store   | internal/store   | 封装 `clientv3`：Get/Put/Delete/WithPrefix/Txn，按实体分文件                                | 不含业务规则      |
| app     | internal/app     | 唯一装配点：构造 etcd client→store→service→handler→Echo(路由+中间件)；暴露 New/Run/Shutdown | —                 |

## 5. 错误处理

跨层错误统一用 `internal/apperr` 的 AppError 类型，错误码集中定义、与 [API 概览](api/overview.md) §5 一一对应。service/store 返回 AppError 表达可预期业务错误，未预期错误包装冒泡；handler 用 `errors.As` 提取转信封，未识别的 error 映射 `INTERNAL_ERROR`/500。`VALIDATION_ERROR` 可携 `details`，对标 [API 设计规范](conventions/api-design.md)。

## 6. 请求生命周期

中间件链（由 `internal/middleware` 提供、`internal/app` 装配）：

```txt
request-id → logger → recover → auth(JWT + 权限分线) → handler → 统一信封
```

公开端点（login/refresh/health）跳过 auth。权限分线对标 [API 概览](api/overview.md) §3：`normal` 管 Record 与只读 Zone/Domain；`admin` 管用户与 Zone/Domain 写操作。响应统一由 `internal/envelope` 封装为 `{success, data, error}`。

## 7. etcd 访问与 CoreDNS 同步

store 层共享 `*clientv3.Client`（构造器注入），封装 `Get`/`Put`/`Delete`/`WithPrefix`/`Txn` 与 JSON 序列化，key 前缀常量（`/dnskeeper/`、`/skydns/`）集中定义。Txn 组合规则与同步模式详见 [dns-sync.md](design/dns-sync.md) §3–§4；service 层随 Record CRUD 在同一 Txn 内直接写入对应 CoreDNS 子键（随路写入），Domain/Zone 级移除按范围删同 Txn。显式 reconcile（含 list+diff 收敛与悬空清理）为独立同步接口的预留扩展。

## 8. 配置

单一 TOML 文件，无环境变量覆盖；不配内置 admin。Schema 草案：

```toml
[server]
listen = ":8080"

[log]
level = "info"            # panic|fatal|error|warn|info|debug|trace

[etcd]
endpoints = ["127.0.0.1:2379"]
username = ""             # 可选
password = ""             # 可选
# cert/key/ca             # TLS 可选

[coredns]
path = "/skydns"          # 须与 CoreDNS Corefile etcd path 一致

[jwt]
secret = "change-me"
access_ttl  = "30m"
refresh_ttl = "168h"      # 7 天
```

## 9. 启动与引导

`main.go` 仅最小入口：载入配置 → log 装配 → 监听信号 → `app.New(cfg)` 一次性装配 → 启动时幂等种子（`/dnskeeper/users/admin` 不存在则建 `admin`/`admin123`、`builtin=true`）→ `app.Run()` → 信号触发 `app.Shutdown`（停接受新请求 → drain → 关 etcd client）。

## 10. 测试策略

- **集成测试**：`etcd/embed` 在 `TestMain` 启进程内 etcd，覆盖 store/Txn/级联，CI 无 Docker 依赖。
- **单元测试**：service 用 store 接口 fake/mock（testify）驱动，覆盖业务规则。
- 断言库：testify。

## 11. 构建与部署

- 多阶段 Dockerfile：`CGO_ENABLED=0` 静态二进制 + distroless/static runtime；同时保留 `go build` 直出。
- dev：`docker-compose.yml` 起 etcd + dnskeeper。
- 运行依赖：dnskeeper 与 CoreDNS 须指向同一 etcd 实例，`[coredns].path` 与 Corefile `etcd path` 一致。
