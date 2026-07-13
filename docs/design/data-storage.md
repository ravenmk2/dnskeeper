---
date: 2026-07-12
---

# 数据存储设计

## 1. 概述

Dnskeeper 使用 etcd v3 作为领域对象的存储。本设计聚焦 User、Zone、Domain、Record 四类领域对象的 Key 规划、数据结构与设计取舍。

作为 CoreDNS etcd 插件的管理工具，dnskeeper 与 CoreDNS 共用同一 etcd 实例：领域对象存放于 `/dnskeeper/` 前缀下；Record 增删改时同步至 CoreDNS 服务记录的机制另文描述（见 [dns-sync.md](dns-sync.md)），不在本设计范围内。

## 2. etcd Key 规划

所有领域对象统一存放于 `/dnskeeper/` 前缀下，按领域对象类型分目录组织：

| 数据类型 | Key 格式                                     | 示例                                  |
| -------- | -------------------------------------------- | ------------------------------------- |
| User     | `/dnskeeper/users/{user-id}`                 | `/dnskeeper/users/admin`              |
| Zone     | `/dnskeeper/dns/{zone}`                      | `/dnskeeper/dns/example.com`          |
| Domain   | `/dnskeeper/dns/{zone}/{domain}`             | `/dnskeeper/dns/example.com/www`      |
| Record   | `/dnskeeper/dns/{zone}/{domain}/{record-id}` | `/dnskeeper/dns/example.com/www/0001` |

设计要点：

- **按领域对象类型分目录**：`/dnskeeper/users/` 与 `/dnskeeper/dns/` 各为独立前缀命名空间，互不冲突；DNS 相关对象（Zone/Domain/Record）统一归于 `/dnskeeper/dns/` 下，User 归于 `/dnskeeper/users/`。
- **多级路径表达从属关系**：Domain 的 Key 嵌入所属 zone（`/dnskeeper/dns/{zone}/{domain}`），Record 的 Key 嵌入所属 zone 与 domain（`/dnskeeper/dns/{zone}/{domain}/{record-id}`），天然表达"某 zone 下某 domain 下某 record"的从属。
- **前缀范围查询支撑集合列举**：
    - `/dnskeeper/dns/{zone}/` 前缀 → 该 Zone 下全部 Domain 与 Record；
    - `/dnskeeper/dns/{zone}/{domain}/` 前缀（带尾 `/`）→ 该 Domain 下全部 Record（排除 Domain 实体自身的 key）；
    - `/dnskeeper/users/` 前缀 → 全部 User。
- **Domain 路径段可含 `.`**：`{domain}` 支持多级（如 `www.beta`），在 etcd Key 中为单段字面字符，`.` 不被视为路径分隔符，亦不被拆分；前缀查询仍按完整 `{domain}` 段匹配。
- **标识符约定**：
    - `{user-id}`：由 `username` 小写化派生，全小写、仅 `[a-z0-9_-]`，等于 `lowercase(username)`；`username` 大小写不敏感（"Admin" 与 "admin" 视为同一用户名），`username` 原样存储（保留大小写），`user-id` 为其规范小写形式。如 `username="Alice"` → `user-id="alice"` → key `/dnskeeper/users/alice`。
    - `{zone}`：域名（FQDN），至少 2 级标签（如 `example.com`、`sub.example.com`），1-253 字符；单标签（如 `com`）不允许。
    - `{domain}`：子域名，`@`（Zone 根，表示 Zone 本身）或一个及多个以 `.` 连接的 DNS 标签（每标签 1-63 字符、字母/数字/连字符、首尾为字母或数字）；与 `{zone}` 共同构成完整域名 `{domain}.{zone}`（`@` 时等于 `{zone}`），整体 ≤ 253 字符。
    - `{record-id}`：Domain 内递增序号，4 位十进制零补齐（`0001`–`9999`）。最近分配的序号保存于 Domain 实体的 `last_record_id` 字段；创建 Record 时 `last_record_id + 1` 作为新 id，**仅递增、不复用**（删除 Record 不回退序号），保证 id 永不重复使用；`last_record_id` 更新与 Record 写入纳入同一 etcd `Txn`，确保序号分配与记录落盘原子一致。序号达到 `9999` 后该 Domain 不可再创建 Record（返回 `RECORD_ID_EXHAUSTED`）。

## 3. 数据结构

领域对象以 JSON 序列化存入 etcd。时间字段使用 RFC 3339 格式（如 `2024-01-01T00:00:00Z`），UTC 时区。

### 3.1 User

```jsonc
{
    "id": "admin",
    "username": "admin",
    "password": "$2a$10$N9qo8uLOickgx2ZMRZoMy.Mrq4v3mZ.mfv6UoZ...",
    "user_type": "admin",
    "builtin": true,
    "created_at": "2023-11-30T04:29:27Z",
    "updated_at": "2023-11-30T04:29:27Z"
}
```

| 字段         | 类型    | 说明                                                                                                   |
| ------------ | ------- | ------------------------------------------------------------------------------------------------------ |
| `id`         | string  | 用户唯一标识，`username` 的小写派生（`lowercase(username)`），与 username 一一对应；大小写不敏感。     |
| `username`   | string  | 登录用户名。                                                                                           |
| `password`   | string  | 密码的 bcrypt 哈希值，非明文；不应在 API 响应中暴露。                                                  |
| `user_type`  | string  | 账号类型：`admin`（管理员）或 `normal`（普通用户）。                                                   |
| `builtin`    | boolean | 是否为内置用户（如初始化的默认管理员），不可删除且不可降级；仅系统初始化时置 true，不可通过 API 修改。 |
| `created_at` | string  | 创建时间，RFC 3339 格式。                                                                              |
| `updated_at` | string  | 最近更新时间，RFC 3339 格式。                                                                          |

### 3.2 Zone

```jsonc
{
    "zone": "example.com",
    "domain_count": 3,
    "created_at": "2023-11-30T04:29:27Z",
    "updated_at": "2023-11-30T04:29:27Z"
}
```

| 字段           | 类型   | 说明                                                                     |
| -------------- | ------ | ------------------------------------------------------------------------ |
| `zone`         | string | 域名（FQDN），如 `example.com`，作为 Zone 的主键，通常二级、亦支持三级。 |
| `domain_count` | int    | 该 Zone 下属 Domain 的数量，由 Domain 增删时维护。                       |
| `created_at`   | string | 创建时间，RFC 3339 格式。                                                |
| `updated_at`   | string | 最近更新时间，RFC 3339 格式。                                            |

### 3.3 Domain

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "name": "www.example.com",
    "record_count": 2,
    "last_record_id": 2,
    "created_at": "2023-11-30T04:29:27Z",
    "updated_at": "2023-11-30T04:29:27Z"
}
```

| 字段             | 类型   | 说明                                                                                             |
| ---------------- | ------ | ------------------------------------------------------------------------------------------------ |
| `zone`           | string | 所属 Zone，如 `example.com`。                                                                    |
| `domain`         | string | 子域名，`@`（Zone 根）或多级标签（可含 `.`，如 `www`、`www.beta`）；与 `zone` 共同构成完整域名。 |
| `name`           | string | 冗余的完整域名，`@` 时等于 `zone`，否则为 `{domain}.{zone}`；便于展示。                          |
| `record_count`   | int    | 该 Domain 下属 Record 的数量，由 Record 增删时维护。                                             |
| `last_record_id` | int    | 最近分配的 record-id 序号，由 Record 创建时递增维护，不回退；**内部字段，不在 API 响应中暴露**。 |
| `created_at`     | string | 创建时间，RFC 3339 格式。                                                                        |
| `updated_at`     | string | 最近更新时间，RFC 3339 格式。                                                                    |

> Domain 实体 Key 为 `/dnskeeper/dns/{zone}/{domain}`（无尾 `/`），其下 Record 为 `/dnskeeper/dns/{zone}/{domain}/{record-id}`；两者在 etcd v3 扁平 Key 下共存。前缀查询 `/dnskeeper/dns/{zone}/{domain}/`（带尾 `/`）返回该 Domain 下全部 Record，直查 `/dnskeeper/dns/{zone}/{domain}` 返回 Domain 实体。

### 3.4 Record

```jsonc
{
    "id": "0001",
    "type": "A",
    "value": "192.168.1.1",
    "ttl": 300
}
```

SRV 示例：

```jsonc
{
    "id": "0002",
    "type": "SRV",
    "value": "srv.example.com.",
    "ttl": 300,
    "priority": 10,
    "port": 8080,
    "weight": 0
}
```

| 字段       | 类型   | 说明                                                                    |
| ---------- | ------ | ----------------------------------------------------------------------- |
| `id`       | string | record-id，Domain 内递增序号，4 位十进制零补齐。                        |
| `type`     | string | 记录类型：`A`/`AAAA`/`SRV`/`TXT`。                                      |
| `value`    | string | 记录值，语义随 `type`：A→IPv4，AAAA→IPv6，SRV→目标主机 FQDN，TXT→文本。 |
| `ttl`      | int    | TTL，单位秒，1–86400。                                                  |
| `priority` | int    | SRV 优先级，0–65535，仅 SRV 携带。                                      |
| `port`     | int    | SRV 端口，0–65535，仅 SRV 携带。                                        |
| `weight`   | int    | SRV 权重，0–65535，仅 SRV 携带。                                        |

> Record 无时间字段；`id` 为 Domain 内递增序号（非时间戳），仅表示创建顺序。`priority`/`port`/`weight` 仅 SRV 类型携带并存储，A/AAAA/TXT 不存储这些字段。

## 4. 设计取舍

- **领域对象亦存 etcd**：与 CoreDNS 共用同一 etcd 实例，统一存储、简化部署，避免引入第二种数据源（如 SQL）及其同步负担。
- **DNS 对象写入启用 Txn**：Zone/Domain/Record 的增删改因需与 CoreDNS 服务记录保持一致，统一纳入 etcd `Txn`（跨 `/dnskeeper/` 与 `/skydns/` 双前缀，详见 [dns-sync.md](dns-sync.md) §4）；统计字段（`domain_count`/`record_count`/`last_record_id`）的维护与对象写入同 `Txn`，保证计数与实体原子一致。
- **User 对象暂不启用 Txn**：User 写入低并发，跨 Key 操作（如"检查-写入"）暂沿用非事务方式，最终一致性可接受；如需强一致可引入 `Txn`。
- **user-id 由 username 小写派生**：`id = lowercase(username)`，与 username 一一对应，作 etcd key 与 API 定位符；username 保留大小写存储但大小写不敏感（id 为规范小写形式）。因 id 派生自 username，username 不可变更（否则 id 漂移），故 update user 不支持改 username；如需变更用户名须删除后重建。
- **record-id 采用 Domain 内递增序号**：序号保存在 Domain 实体的 `last_record_id`，创建时递增、不复用，id 稳定且永不重复——同步 reconcile 可直接按 id 一一对应，删除后的 id 不会被重新分配给不同内容，悬空记录识别无歧义（见 [dns-sync.md](dns-sync.md)）；代价是序号不可复用，4 位上限为 9999，达上限后该 Domain 不可再创建 Record。
- **Zone 至少 2 级标签**：`{zone}` 强制 ≥ 2 级标签（如 `example.com`），单标签（如 `com`）不允许，作为 FQDN 格式校验一部分，违例返回 `VALIDATION_ERROR`。
- **Domain 创建预防与 Zone 嵌套冲突**：Domain 创建时将其全名 `{domain}.{zone}` 及逐级删除 `domain` 最左标签（保留 ≥1 级）所得祖先名，逐一与既有 Zone 列表比对，任一命中返回 `DOMAIN_ZONE_CONFLICT`；`@`（Zone 根）跳过。该校验为前置预检查（同 `ZONE_EXISTS`/`DOMAIN_EXISTS` 语义，非强一致），旨在消除父子 Zone/Domain 归属歧义（见 [zone.md](../api/zone.md) 模块说明）。
