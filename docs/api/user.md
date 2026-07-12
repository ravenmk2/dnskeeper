---
date: 2026-07-12
---

# 用户管理模块

> 本模块所有接口需 `admin` 权限。

## 列出所有用户

列出所有用户。

**请求**

```http
POST /api/user/list
```

无请求体。

**响应**

```jsonc
{
    "success": true,
    "data": [
        {
            "id": 1701320967,
            "username": "admin",
            "user_type": "admin",
            "builtin": true,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        },
        {
            "id": 1701320968,
            "username": "user1",
            "user_type": "normal",
            "builtin": false,
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
| `FORBIDDEN`    | 403  | 非 Admin 用户       |
| `UNAUTHORIZED` | 401  | 未认证或 Token 无效 |

---

## 创建用户

创建用户。

**请求**

```http
POST /api/user/create
```

```jsonc
{
    "username": "newuser",
    "password": "password123",
    "user_type": "normal"
}
```

**字段约束**

- `username`: 3-32 字符，必填
- `password`: 6-72 字符，必填
- `user_type`: `admin` 或 `normal`，必填

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": 1701320967,
        "username": "newuser",
        "user_type": "normal",
        "builtin": false,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-01T00:00:00Z"
    },
    "error": null
}
```

**错误场景**

| 错误码             | HTTP | 说明                |
| ------------------ | ---- | ------------------- |
| `USER_EXISTS`      | 200  | 用户名已存在        |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束  |
| `FORBIDDEN`        | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效 |

---

## 更新用户

更新用户。

**请求**

```http
POST /api/user/update
```

```jsonc
{
    "id": 1701320967,
    "username": "updateduser",
    "password": "newpassword123",
    "user_type": "normal"
}
```

**字段约束**

- `id`: 必填
- `username`: 3-32 字符，可选
- `password`: 6-72 字符，可选
- `user_type`: `admin` 或 `normal`，可选

> `username`、`password`、`user_type` 至少提供一项，否则返回 `VALIDATION_ERROR`。

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": 1701320967,
        "username": "updateduser",
        "user_type": "normal",
        "builtin": false,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-02T00:00:00Z"
    },
    "error": null
}
```

**错误场景**

| 错误码                  | HTTP | 说明                   |
| ----------------------- | ---- | ---------------------- |
| `USER_NOT_FOUND`        | 200  | 用户不存在             |
| `USER_EXISTS`           | 200  | 更新后的用户名已被占用 |
| `CANNOT_DEMOTE_BUILTIN` | 200  | 不能降级内置用户       |
| `VALIDATION_ERROR`      | 200  | 请求参数不符合约束     |
| `FORBIDDEN`             | 403  | 非 Admin 用户          |
| `UNAUTHORIZED`          | 401  | 未认证或 Token 无效    |

---

## 删除用户

删除用户。

**请求**

```http
POST /api/user/delete
```

```jsonc
{
    "id": 1701320967
}
```

**字段约束**

- `id`: 必填

**响应**

```jsonc
{
    "success": true,
    "data": null,
    "error": null
}
```

**错误场景**

| 错误码                  | HTTP | 说明                |
| ----------------------- | ---- | ------------------- |
| `USER_NOT_FOUND`        | 200  | 用户不存在          |
| `CANNOT_DELETE_BUILTIN` | 200  | 不能删除内置用户    |
| `FORBIDDEN`             | 403  | 非 Admin 用户       |
| `VALIDATION_ERROR`      | 200  | 请求参数不符合约束  |
| `UNAUTHORIZED`          | 401  | 未认证或 Token 无效 |
