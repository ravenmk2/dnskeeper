---
date: 2026-07-12
---

# Domain 管理模块

> 本模块写操作（create/update/delete）需 `admin` 权限；只读操作（list/get）仅需通过认证。

Domain 代表完整域名的第一部分，如 Zone `example.com` 下的 `www`、`api`，或 `www.beta`（多级，带 `.`）。完整域名为 `{domain}.{zone}`，如 `www.example.com`、`www.beta.example.com`；`domain` 为 `@` 时表示 Zone 根记录，完整域名等于 `zone`。

Domain 是其下 Record 的分组容器，本身不持有 DNS 数据（DNS 数据在 Record 中）。Domain 实体维护 `record_count` 统计与 `last_record_id` 序号计数器（后者为内部字段，不在 API 响应中暴露，详见 [数据存储设计](../design/data-storage.md)）。

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
            "zone": "example.com",
            "domain": "www",
            "name": "www.example.com",
            "record_count": 2,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        },
        {
            "zone": "example.com",
            "domain": "@",
            "name": "example.com",
            "record_count": 1,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    ],
    "error": null
}
```

**错误场景**

| 错误码           | HTTP | 说明                |
| ---------------- | ---- | ------------------- |
| `ZONE_NOT_FOUND` | 200  | Zone 不存在         |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`   | 401  | 未认证或 Token 无效 |

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
- `domain`: `@` 或一个及多个以 `.` 连接的 DNS 标签（每标签 1-63 字符，字母、数字、连字符，首尾为字母或数字），如 `www`、`www.beta`；`@` 表示 Zone 根，必填

**响应**

```jsonc
{
    "success": true,
    "data": {
        "zone": "example.com",
        "domain": "www",
        "name": "www.example.com",
        "record_count": 2,
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

创建 Domain。Domain 为空容器（`record_count` 初始为 0），创建后可在其下创建 Record。创建 Domain 不产生 CoreDNS 同步（CoreDNS 记录由 Record 驱动）。

**请求**

```http
POST /api/dns/domain/create
```

```jsonc
{
    "zone": "example.com",
    "domain": "www"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填（需已存在）
- `domain`: `@` 或多级 DNS 标签（见上文约束），必填

**响应**

```jsonc
{
    "success": true,
    "data": {
        "zone": "example.com",
        "domain": "www",
        "name": "www.example.com",
        "record_count": 0,
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
| `DOMAIN_EXISTS`    | 200  | Domain 已存在       |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 更新 Domain

更新 Domain。

> 当前 Domain 实体只包含名称与统计字段，此接口主要用于刷新 `updated_at` 时间戳。未来可能增加更多可更新字段。

**请求**

```http
POST /api/dns/domain/update
```

```jsonc
{
    "zone": "example.com",
    "domain": "www"
}
```

**字段约束**

- `zone`: 有效域名（FQDN），1-253 字符，必填
- `domain`: `@` 或多级 DNS 标签（见上文约束），必填

**响应**

```jsonc
{
    "success": true,
    "data": {
        "zone": "example.com",
        "domain": "www",
        "name": "www.example.com",
        "record_count": 2,
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
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 删除 Domain

删除 Domain。

> 删除 Domain 会**级联删除**该 Domain 下所有 Record，并清除对应的 CoreDNS 记录。

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
- `domain`: `@` 或多级 DNS 标签（见上文约束），必填

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
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |
