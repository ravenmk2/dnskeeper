---
date: 2026-07-12
---

# 健康检查

## 健康检查

反映 etcd 连接状态。公开端点，无需认证。同时支持 `GET` 与 `POST`。

**请求**

```http
GET /api/health
POST /api/health
```

无请求体。

**响应（etcd up）**

HTTP 200：

```jsonc
{
    "success": true,
    "data": null,
    "error": null
}
```

**响应（etcd down）**

HTTP 503：

```jsonc
{
    "success": false,
    "data": null,
    "error": {
        "code": "SERVICE_UNAVAILABLE",
        "message": "etcd service unavailable"
    }
}
```

**说明**

健康检查遵循 [API 响应规范](overview.md)。etcd 正常时返回 200；etcd 不可用时返回 503 与 `SERVICE_UNAVAILABLE` 错误（见 [错误码总表](overview.md)）。
