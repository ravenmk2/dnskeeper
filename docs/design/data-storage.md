---
date: 2026-07-12
---

# 数据存储设计

## 1. 概述

Dnskeeper 使用 etcd v3 作为领域对象的存储。本设计聚焦 User、Zone、Domain 三类领域对象的 Key 规划、数据结构与设计取舍。

作为 CoreDNS etcd 插件的管理工具，dnskeeper 与 CoreDNS 共用同一 etcd 实例：领域对象存放于 `/dnskeeper/` 前缀下；Domain 增删改时同步至 CoreDNS 服务记录的机制另文描述，不在本设计范围内。

## 2. etcd Key 规划

所有领域对象统一存放于 `/dnskeeper/` 前缀下，按领域对象类型分目录组织：

| 数据类型 | Key 格式                             | 示例                                 |
| -------- | ------------------------------------ | ------------------------------------ |
| User     | `/dnskeeper/users/{user-id}`         | `/dnskeeper/users/1701234567890`     |
| Zone     | `/dnskeeper/zones/{zone}`            | `/dnskeeper/zones/example.com`       |
| Domain   | `/dnskeeper/domains/{zone}/{domain}` | `/dnskeeper/domains/example.com/www` |

设计要点：

- **按领域对象类型分目录**：`/dnskeeper/users/`、`/dnskeeper/zones/`、`/dnskeeper/domains/` 各为一个前缀命名空间，互不冲突。
- **多级路径表达从属关系**：Domain 的 Key 嵌入所属 zone（`/dnskeeper/domains/{zone}/{domain}`），天然表达"某 zone 下的某 domain"。
- **前缀范围查询支撑集合列举**：对 `/dnskeeper/users/`、`/dnskeeper/domains/example.com/` 等前缀发起 `WithPrefix` 查询即可获取该领域对象集合或子集合，无需额外索引。
- **标识符约定**：`{user-id}` 使用单调生成值（如时间戳）以避免并发冲突；`{zone}` 为二级域名（如 `example.com`）；`{domain}` 为子域名部分（如 `www`）。

## 3. 数据结构

领域对象以 JSON 序列化存入 etcd。时间戳均为 Unix 秒（`int64`）。

### 3.1 User

```jsonc
{
    "id": "1701234567890",
    "username": "admin",
    "password": "$2a$10$N9qo8uLOickgx2ZMRZoMy.Mrq4v3mZ.mfv6UoZ...",
    "user_type": "admin",
    "created_at": 1701234567,
    "updated_at": 1701234567
}
```

| 字段         | 类型   | 说明                                                  |
| ------------ | ------ | ----------------------------------------------------- |
| `id`         | string | 用户唯一标识，单调生成（如时间戳）。                  |
| `username`   | string | 登录用户名。                                          |
| `password`   | string | 密码的 bcrypt 哈希值，非明文；不应在 API 响应中暴露。 |
| `user_type`  | string | 账号类型：`admin`（管理员）或 `normal`（普通用户）。  |
| `created_at` | int64  | 创建时间戳（Unix 秒）。                               |
| `updated_at` | int64  | 最近更新时间戳（Unix 秒）。                           |

### 3.2 Zone

```jsonc
{
    "zone": "example.com",
    "domain_count": 3,
    "created_at": 1701234567,
    "updated_at": 1701234567
}
```

| 字段           | 类型   | 说明                                               |
| -------------- | ------ | -------------------------------------------------- |
| `zone`         | string | 二级域名，如 `example.com`，作为 Zone 的主键。     |
| `domain_count` | int    | 该 Zone 下属 Domain 的数量，由 Domain 增删时维护。 |
| `created_at`   | int64  | 创建时间戳（Unix 秒）。                            |
| `updated_at`   | int64  | 最近更新时间戳（Unix 秒）。                        |

### 3.3 Domain

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "name": "www.example.com",
    "ips": ["192.168.1.1", "192.168.1.2"],
    "ttl": 300,
    "created_at": 1701234567,
    "updated_at": 1701234567
}
```

| 字段         | 类型     | 说明                                                                    |
| ------------ | -------- | ----------------------------------------------------------------------- |
| `zone`       | string   | 所属 Zone，如 `example.com`。                                           |
| `domain`     | string   | 子域名部分，如 `www`；与 `zone` 共同构成完整域名。                      |
| `name`       | string   | 完整域名，`{domain}.{zone}`，如 `www.example.com`；冗余字段，便于展示。 |
| `ips`        | string[] | IP 地址列表，多 IP 对应同一域名的多条记录。                             |
| `ttl`        | int      | TTL，单位秒。                                                           |
| `created_at` | int64    | 创建时间戳（Unix 秒）。                                                 |
| `updated_at` | int64    | 最近更新时间戳（Unix 秒）。                                             |

## 4. 设计取舍

- **领域对象亦存 etcd**：与 CoreDNS 共用同一 etcd 实例，统一存储、简化部署，避免引入第二种数据源（如 SQL）及其同步负担。
- **跨 key 非事务**：etcd v3 单 key 原子，但跨 key 的"检查-写入"、"级联删除"不保证原子，存在最终一致性窗口；对 DNS 元数据这类低并发场景可接受。
