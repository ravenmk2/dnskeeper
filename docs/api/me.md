---
date: 2026-07-12
---

# 当前用户模块

## 获取当前用户

获取当前登录用户信息。

**请求**

```http
POST /api/me
```

无请求体。

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": "admin",
        "username": "admin",
        "user_type": "admin",
        "builtin": true,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    },
    "error": null
}
```

**错误场景**

| 错误码           | HTTP | 说明                |
| ---------------- | ---- | ------------------- |
| `USER_NOT_FOUND` | 200  | 用户不存在          |
| `UNAUTHORIZED`   | 401  | 未认证或 Token 无效 |

---

## 修改密码

修改当前用户密码。

**请求**

```http
POST /api/me/change-password
```

```jsonc
{
    "old_password": "oldpassword123",
    "new_password": "newpassword456"
}
```

**字段约束**

- `old_password`: 6-24 字符，必填
- `new_password`: 6-24 字符，至少含大写/小写/数字/特殊字符中的 2 类，必填

**响应**

```jsonc
{
    "success": true,
    "data": null,
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                     |
| ------------------ | ---- | ------------------------ |
| `WRONG_PASSWORD`   | 200  | 旧密码错误               |
| `SAME_PASSWORD`    | 200  | 新密码与旧密码相同       |
| `WEAK_PASSWORD`    | 200  | 新密码不符合强度规则     |
| `VALIDATION_ERROR` | 200  | 请求参数缺失或不符合约束 |
| `USER_NOT_FOUND`   | 200  | 用户不存在               |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效      |

> `WRONG_PASSWORD`、`SAME_PASSWORD` 返回 `200`：已认证场景下的密码规则校验属业务错误而非安全层错误。
> `new_password` 须满足强度规则（6-24 字符、至少 2/4 字符类），不符合返回 `WEAK_PASSWORD`。
> 修改密码不会使已签发的 token 失效；access token 与 refresh token 继续有效直至自然过期。
