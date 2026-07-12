---
date: 2026-07-12
---

# Record 管理模块

> 本模块所有接口仅需通过认证（登录）。

Record 是 Domain 下的具体 DNS 记录，支持 `A`、`AAAA`、`SRV`、`TXT` 四种类型。Record 由 `(zone, domain, id)` 定位，`id` 为 Domain 内递增序号（4 位零补齐，如 `0001`），由服务端生成、Domain 内唯一且不复用（详见 [数据存储设计](../design/data-storage.md)）。

## 记录类型与字段

| type   | value 语义                | 必带字段                         | 说明                                                    |
| ------ | ------------------------- | -------------------------------- | ------------------------------------------------------- |
| `A`    | IPv4 地址                 | `value`、`ttl`                   | 生成 A 记录。                                           |
| `AAAA` | IPv6 地址                 | `value`、`ttl`                   | 生成 AAAA 记录。                                        |
| `SRV`  | 目标主机 FQDN（带尾点）   | `value`、`ttl`、`priority`、`port` | `weight` 可选，缺省 0；生成 SRV 记录。                 |
| `TXT`  | 文本（单条 ≤ 255 字节）   | `value`、`ttl`                   | 生成 TXT 记录。                                         |

> `priority`/`port`/`weight` 仅 SRV 携带；A/AAAA/TXT 请求中携带这些字段返回 `VALIDATION_ERROR`。`type` 不可通过 update 修改，需变更类型时先删除再创建。

> 同一 Domain 下禁止完全重复的记录：A/AAAA/TXT 以 `type`+`value` 判定，SRV 以 `type`+`value`+`priority`+`port`+`weight` 判定；重复时返回 `RECORD_EXISTS`。

---

## 列出 Record

列出 Domain 下所有 Record（按 `id` 升序）。

**请求**

```http
POST /api/dns/record/list
```

```jsonc
{
    "zone": "example.com",
    "domain": "www"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: `@` 或多级 DNS 标签（每标签 1-63 字符，字母、数字、连字符，首尾为字母或数字），如 `www`、`www.beta`；`@` 表示 Zone 根，必填

**响应**

```jsonc
{
    "success": true,
    "data": [
        {
            "id": "0001",
            "type": "A",
            "value": "192.168.1.1",
            "ttl": 300
        },
        {
            "id": "0002",
            "type": "AAAA",
            "value": "2001:db8::1",
            "ttl": 300
        },
        {
            "id": "0003",
            "type": "SRV",
            "value": "srv.example.com.",
            "ttl": 300,
            "priority": 10,
            "port": 8080,
            "weight": 0
        }
    ],
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在         |
| `DOMAIN_NOT_FOUND` | 200  | Domain 不存在       |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 获取 Record 详情

获取 Record 详情。

**请求**

```http
POST /api/dns/record/get
```

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "id": "0001"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: `@` 或多级 DNS 标签，必填
- `id`: 4 位 record-id，必填

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": "0001",
        "type": "A",
        "value": "192.168.1.1",
        "ttl": 300
    },
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在         |
| `DOMAIN_NOT_FOUND` | 200  | Domain 不存在       |
| `RECORD_NOT_FOUND` | 200  | Record 不存在       |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 创建 Record

创建 Record。`id` 由服务端按 Domain 内递增序号生成（`last_record_id + 1`，4 位零补齐），请求中无需也不可指定 `id`。

**请求**

```http
POST /api/dns/record/create
```

A 记录：

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "type": "A",
    "value": "192.168.1.1",
    "ttl": 300
}
```

SRV 记录：

```jsonc
{
    "zone": "example.com",
    "domain": "_sip._tcp",
    "type": "SRV",
    "value": "sip.example.com.",
    "ttl": 300,
    "priority": 10,
    "port": 5060,
    "weight": 0
}
```

TXT 记录：

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "type": "TXT",
    "value": "v=spf1 include:_spf.example.com ~all",
    "ttl": 300
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填（需已存在）
- `domain`: `@` 或多级 DNS 标签，必填（需已存在）
- `type`: `A`/`AAAA`/`SRV`/`TXT`，必填
- `value`: 必填，须与 `type` 匹配（A→IPv4，AAAA→IPv6，SRV→目标 FQDN，TXT→文本 ≤ 255 字节）
- `ttl`: 1-86400，必填
- `priority`/`port`: SRV 必填，0-65535；非 SRV 禁止携带
- `weight`: SRV 可选，0-65535，缺省 0；非 SRV 禁止携带

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": "0001",
        "type": "A",
        "value": "192.168.1.1",
        "ttl": 300
    },
    "error": null
}
```

> 创建 Record 会同步写入对应的 CoreDNS 记录，与 Domain 实体的 `last_record_id`/`record_count` 更新纳入同一 etcd `Txn`（详见 [DNS 数据同步设计](../design/dns-sync.md)）。

**错误场景**

| 错误码                | HTTP | 说明                          |
| --------------------- | ---- | ----------------------------- |
| `ZONE_NOT_FOUND`      | 200  | Zone 不存在                   |
| `DOMAIN_NOT_FOUND`    | 200  | Domain 不存在                 |
| `RECORD_EXISTS`       | 200  | 同 Domain 下已存在重复记录    |
| `RECORD_TYPE_INVALID` | 200  | `type` 非 A/AAAA/SRV/TXT      |
| `RECORD_ID_EXHAUSTED` | 200  | Domain 下序号达 9999 上限     |
| `VALIDATION_ERROR`    | 200  | 请求参数不符合约束            |
| `UNAUTHORIZED`        | 401  | 未认证或 Token 无效           |

---

## 更新 Record

更新 Record 的可变字段。`type` 不可变更；至少提供 `value`/`ttl`/`priority`/`port`/`weight` 之一。

**请求**

```http
POST /api/dns/record/update
```

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "id": "0001",
    "value": "192.168.1.10",
    "ttl": 600
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: `@` 或多级 DNS 标签，必填
- `id`: 4 位 record-id，必填
- `value`: 可选，须与原 `type` 匹配
- `ttl`: 可选，1-86400，不填则保持原值
- `priority`/`port`/`weight`: 仅 SRV 可选；非 SRV 禁止携带
- 至少提供一项可变字段，否则返回 `VALIDATION_ERROR`

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": "0001",
        "type": "A",
        "value": "192.168.1.10",
        "ttl": 600
    },
    "error": null
}
```

> 更新 Record 会同步覆盖对应的 CoreDNS 记录（字段变化时），与 Record 写入纳入同一 etcd `Txn`。

**错误场景**

| 错误码             | HTTP | 说明                          |
| ------------------ | ---- | ----------------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在                   |
| `DOMAIN_NOT_FOUND` | 200  | Domain 不存在                 |
| `RECORD_NOT_FOUND` | 200  | Record 不存在                 |
| `RECORD_EXISTS`    | 200  | 更新后与同 Domain 下记录重复  |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束            |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效           |

---

## 删除 Record

删除 Record。

> 删除 Record 会同步删除对应的 CoreDNS 记录，并递减所属 Domain 的 `record_count`（`last_record_id` 不回退，id 不复用）。删除与计数维护纳入同一 etcd `Txn`。

**请求**

```http
POST /api/dns/record/delete
```

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "id": "0001"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: `@` 或多级 DNS 标签，必填
- `id`: 4 位 record-id，必填

**响应**

```jsonc
{
    "success": true,
    "data": null,
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在         |
| `DOMAIN_NOT_FOUND` | 200  | Domain 不存在       |
| `RECORD_NOT_FOUND` | 200  | Record 不存在       |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |
