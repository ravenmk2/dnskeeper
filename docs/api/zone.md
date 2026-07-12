---
date: 2026-07-12
---

# Zone 管理模块

> 本模块写操作（create/update/delete）需 `admin` 权限；只读操作（list/get）仅需通过认证。

Zone 代表域名区域（FQDN），如 `example.com`，通常为二级域名，亦支持三级（如 `sub.example.com`）。Zone 仅起分组作用，便于按 zone 批量管理其下 Domain 与 Record。

> 避免父子 Zone 同时存在（如 `example.com` 与 `a.example.com` 同时作为 Zone），以免查询时产生归属歧义。

## 列出所有 Zone

列出所有 Zone。

**请求**

```http
POST /api/dns/zone/list
```

无请求体。

**响应**

```jsonc
{
    "success": true,
    "data": [
        {
            "zone": "example.com",
            "domain_count": 5,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        }
    ],
    "error": null
}
```

**错误场景**

| 错误码         | HTTP | 说明                |
| -------------- | ---- | ------------------- |
| `UNAUTHORIZED` | 401  | 未认证或 Token 无效 |

---

## 获取 Zone 详情

获取 Zone 详情。

**请求**

```http
POST /api/dns/zone/get
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
    "data": {
        "zone": "example.com",
        "domain_count": 5,
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
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 创建 Zone

创建 Zone。

**请求**

```http
POST /api/dns/zone/create
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
    "data": {
        "zone": "example.com",
        "domain_count": 0,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    },
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `ZONE_EXISTS`      | 200  | Zone 已存在         |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 更新 Zone

更新 Zone。

> 当前 Zone 实体只包含 zone 名称与统计字段，此接口主要用于刷新 `updated_at` 时间戳。未来可能增加更多可更新字段。

**请求**

```http
POST /api/dns/zone/update
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
    "data": {
        "zone": "example.com",
        "domain_count": 5,
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
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 删除 Zone

删除 Zone。

> 删除 Zone 会**级联删除**该 Zone 下所有 Domain 及其 Record，并清除对应的 CoreDNS 记录。

**请求**

```http
POST /api/dns/zone/delete
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
    "data": null,
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `ZONE_NOT_FOUND`   | 200  | Zone 不存在         |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |
