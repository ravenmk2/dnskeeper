---
date: 2026-07-12
---

# Domain 管理模块

Domain 代表完整域名（子域名），如 Zone `example.com` 下的 `www` 或 `@`（根）。本模块接口需通过 JWT 认证。

## 列出 Domain

列出 Zone 下所有 Domain。

**请求**

```http
POST /api/dns/domain/list
```

```jsonc
{
    "zone": "example.com"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填

**响应**

```jsonc
{
    "success": true,
    "data": [
        {
            "domain": "www",
            "name": "www.example.com",
            "ips": ["192.168.1.1", "192.168.1.2"],
            "ttl": 300,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        },
        {
            "domain": "@",
            "name": "example.com",
            "ips": ["192.168.1.10"],
            "ttl": 600,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    ],
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在         |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 获取 Domain 详情

获取 Domain 详情。

**请求**

```http
POST /api/dns/domain/get
```

```jsonc
{
    "zone": "example.com",
    "domain": "www"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: 不带 `.` 的子域名标签（1-63 字符，字母、数字、连字符，首尾为字母或数字），如 `www`；或 `@`（特例，表示 Zone 根记录），必填

**响应**

```jsonc
{
    "success": true,
    "data": {
        "domain": "www",
        "name": "www.example.com",
        "ips": ["192.168.1.1", "192.168.1.2"],
        "ttl": 300,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    },
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

## 创建 Domain

创建 Domain。

**请求**

```http
POST /api/dns/domain/create
```

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "ips": ["192.168.1.1", "192.168.1.2"],
    "ttl": 300
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填（需已存在）
- `domain`: 不带 `.` 的子域名标签（1-63 字符，字母、数字、连字符，首尾为字母或数字），如 `www`；或 `@`（特例，表示 Zone 根记录），必填
- `ips`: IP 地址数组（IPv4 与 IPv6），必填，允许为空数组，每个 IP 必须是有效格式；重复 IP 自动去重
- `ttl`: TTL（秒），必填，最小值 1，最大值 86400

**响应**

```jsonc
{
    "success": true,
    "data": {
        "domain": "www",
        "name": "www.example.com",
        "ips": ["192.168.1.1", "192.168.1.2"],
        "ttl": 300,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    },
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                       |
| ------------------ | ---- | -------------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在，需先创建 Zone |
| `DOMAIN_EXISTS`    | 200  | Domain 已存在              |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束         |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效        |

---

## 更新 Domain

更新 Domain。

**请求**

```http
POST /api/dns/domain/update
```

```jsonc
{
    "zone": "example.com",
    "domain": "www",
    "ips": ["192.168.1.3", "192.168.1.4"],
    "ttl": 600
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: 不带 `.` 的子域名标签（1-63 字符，字母、数字、连字符，首尾为字母或数字），如 `www`；或 `@`（特例，表示 Zone 根记录），必填
- `ips`: IP 地址数组（IPv4 与 IPv6），必填，允许为空数组，**替换**现有所有 IP；为空数组时删除该 Domain 的所有 CoreDNS 记录；重复 IP 自动去重
- `ttl`: 可选，不填则保持原值，最小值 1，最大值 86400

> 系统自动比较新旧 IP 列表，新增/删除差异 IP，保持 CoreDNS 记录与请求一致。

**响应**

```jsonc
{
    "success": true,
    "data": {
        "domain": "www",
        "name": "www.example.com",
        "ips": ["192.168.1.3", "192.168.1.4"],
        "ttl": 600,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-02T00:00:00Z"
    },
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

## 删除 Domain

删除 Domain。

> 删除 Domain 会**级联删除**该 Domain 的所有 CoreDNS 记录。

**请求**

```http
POST /api/dns/domain/delete
```

```jsonc
{
    "zone": "example.com",
    "domain": "www"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: 不带 `.` 的子域名标签（1-63 字符，字母、数字、连字符，首尾为字母或数字），如 `www`；或 `@`（特例，表示 Zone 根记录），必填

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
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |
