---
date: 2026-07-12
---

# 认证模块

## 登录

用户登录，验证用户名密码后签发 JWT。

**请求**

```http
POST /api/auth/login
```

```jsonc
{
    "username": "admin",
    "password": "admin123"
}
```

**响应**

```jsonc
{
    "success": true,
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    },
    "error": null
}
```

**错误场景**

| 错误码                | HTTP | 说明                   |
| --------------------- | ---- | ---------------------- |
| `INVALID_CREDENTIALS` | 401  | 用户名或密码错误       |
| `VALIDATION_ERROR`    | 200  | 请求参数缺失或格式错误 |

> `INVALID_CREDENTIALS` 返回 `401`：登录属安全层，凭证错误按未认证处理。

---

## 刷新 Token

刷新 Token，公开端点。通过请求体携带 `refresh_token`，校验通过后签发新的 access Token 与 refresh_token（轮换）。

**请求**

```http
POST /api/auth/refresh
```

```jsonc
{
    "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
}
```

**响应**

```jsonc
{
    "success": true,
    "data": {
        "token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
        "refresh_token": "eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9..."
    },
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                                         |
| ------------------ | ---- | -------------------------------------------- |
| `VALIDATION_ERROR` | 200  | 请求参数缺失或格式错误                       |
| `INVALID_TOKEN`    | 401  | refresh_token 无效（格式、签名错误或已过期） |
