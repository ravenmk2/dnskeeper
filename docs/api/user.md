---
date: 2026-07-12
---

# 用户管理模块

> 本模块所有接口需 `admin` 权限。

`username` 须由字母、数字、下划线、连字符组成（`[a-zA-Z0-9_-]`），3-32 字符，字母开头、字母/数字结尾，下划线与连字符不可连续；`username` 大小写不敏感，`user-id` 为其小写派生（`lowercase(username)`），作 API 定位符与 etcd key。

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
            "id": "admin",
            "username": "admin",
            "user_type": "admin",
            "builtin": true,
            "created_at": "2024-01-01T00:00:00Z",
            "updated_at": "2024-01-01T00:00:00Z"
        },
        {
            "id": "user1",
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

- `username`: 3-32 字符；字符集 `[a-zA-Z0-9_-]`；必须以字母开头、以字母或数字结尾；下划线与连字符不可连续出现；大小写不敏感（与已存在用户名冲突返回 `USER_EXISTS`）；必填
- `password`: 6-24 字符；至少包含大写字母、小写字母、数字、特殊字符中的 2 类；必填
- `user_type`: `admin` 或 `normal`，必填

> 特殊字符指 ASCII 可见非字母数字字符（如 `!@#$%^&*()-_=+` 等）。`username` 含大写时，`id` 为其小写形式（如 `username="Alice"` → `id="alice"`）。

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": "newuser",
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

| 错误码             | HTTP | 说明                         |
| ------------------ | ---- | ---------------------------- |
| `USER_EXISTS`      | 200  | 用户名已存在（大小写不敏感） |
| `WEAK_PASSWORD`    | 200  | 密码不符合强度规则           |
| `VALIDATION_ERROR` | 200  | 请求参数不符合约束           |
| `FORBIDDEN`        | 403  | 非 Admin 用户                |
| `UNAUTHORIZED`     | 401  | 未认证或 Token 无效          |

---

## 更新用户

更新用户。

**请求**

```http
POST /api/user/update
```

```jsonc
{
    "id": "newuser",
    "password": "newpassword123",
    "user_type": "normal"
}
```

**字段约束**

- `id`: string，必填（`lowercase(username)`）
- `password`: 6-24 字符，至少含大写/小写/数字/特殊字符中的 2 类，可选
- `user_type`: `admin` 或 `normal`，可选

> `password`、`user_type` 至少提供一项，否则返回 `VALIDATION_ERROR`。
> `username` 不可更新（`id` 派生自 `username`，变更会导致 `id` 漂移）；如需变更用户名，须删除后重建。

**响应**

```jsonc
{
    "success": true,
    "data": {
        "id": "newuser",
        "username": "newuser",
        "user_type": "normal",
        "builtin": false,
        "created_at": "2024-01-01T00:00:00Z",
        "updated_at": "2024-01-02T00:00:00Z"
    },
    "error": null
}
```

**错误场景**

| 错误码                  | HTTP | 说明                |
| ----------------------- | ---- | ------------------- |
| `USER_NOT_FOUND`        | 200  | 用户不存在          |
| `CANNOT_DEMOTE_BUILTIN` | 200  | 不能降级内置用户    |
| `WEAK_PASSWORD`         | 200  | 密码不符合强度规则  |
| `VALIDATION_ERROR`      | 200  | 请求参数不符合约束  |
| `FORBIDDEN`             | 403  | 非 Admin 用户       |
| `UNAUTHORIZED`          | 401  | 未认证或 Token 无效 |

---

## 删除用户

删除用户。

**请求**

```http
POST /api/user/delete
```

```jsonc
{
    "id": "newuser"
}
```

**字段约束**

- `id`: string，必填（`lowercase(username)`）

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
