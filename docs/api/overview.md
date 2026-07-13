---
date: 2026-07-12
---

# API 概览

## 1. 概述

Dnskeeper 提供基于 etcd 的 DNS 管理 API。所有接口以操作为建模单位，参数统一通过请求体传递。

> 本文档描述 Dnskeeper 的具体 API 约定；与 [API 设计规范](../conventions/api-design.md) 不一致时，以本文档为准。

- **基础 URL**: `http://{host}:{port}/api`
- **默认端口**: `8080`
- **Content-Type**: `application/json`

除健康检查同时支持 `GET` 与 `POST` 外，所有接口均使用 `POST`。路径与操作一一对应，使用 Kebab Case 命名，由模块、分组和操作组成。

## 2. 鉴权

除登录、刷新 Token、健康检查为公开端点外，所有接口需在请求头携带 JWT：

```txt
Authorization: Bearer <token>
```

Token 过期后需调用刷新接口续期。

## 3. 权限级别

| 类型     | 说明                                     |
| -------- | ---------------------------------------- |
| `normal` | 普通用户，可管理 Record                  |
| `admin`  | 管理员，可管理用户、Zone、Domain、Record |

用户管理接口需 `admin` 权限；Zone 与 Domain 管理的只读操作（list/get）仅需通过认证，其写操作（create/update/delete）需 `admin` 权限；Record 管理与当前用户接口仅需通过认证。

## 4. 响应格式

响应信封统一通过 `success`/`data`/`error` 区分业务结果，详见 [API 设计规范](../conventions/api-design.md)。

成功：

```jsonc
{
    "success": true,
    "data": {},
    "error": null
}
```

失败：

```jsonc
{
    "success": false,
    "data": null,
    "error": {
        "code": "USER_NOT_FOUND",
        "message": "..."
    }
}
```

### 列表端点

当前所有列表端点均不分页，`data` 直接返回数组。

### HTTP 状态码约定

- `200`：业务成功与业务错误（如实体不存在、已存在、参数校验失败等），通过信封的 `success`/`error` 区分。
- `4xx`：传输层与安全层错误，如未认证（`401`）、权限不足（`403`）。
- `5xx`：系统层错误，如服务不可用（`503`）、内部错误（`500`）。

## 5. 错误码总表

| 错误码                  | HTTP | 说明                                          |
| ----------------------- | ---- | --------------------------------------------- |
| `VALIDATION_ERROR`      | 200  | 请求参数校验失败                              |
| `INVALID_CREDENTIALS`   | 401  | 用户名或密码错误                              |
| `WRONG_PASSWORD`        | 200  | 旧密码错误                                    |
| `SAME_PASSWORD`         | 200  | 新密码与旧密码相同                            |
| `WEAK_PASSWORD`         | 200  | 密码不符合强度规则（长度或字符类不足）        |
| `UNAUTHORIZED`          | 401  | 受保护端点：未携带 access token 或其无效/过期 |
| `INVALID_TOKEN`         | 401  | 刷新端点：refresh token 无效/过期             |
| `FORBIDDEN`             | 403  | 权限不足                                      |
| `CANNOT_DELETE_BUILTIN` | 200  | 不能删除内置用户                              |
| `CANNOT_DEMOTE_BUILTIN` | 200  | 不能降级内置用户                              |
| `USER_NOT_FOUND`        | 200  | 用户不存在                                    |
| `USER_EXISTS`           | 200  | 用户已存在                                    |
| `ZONE_NOT_FOUND`        | 200  | Zone 不存在                                   |
| `ZONE_EXISTS`           | 200  | Zone 已存在                                   |
| `DOMAIN_NOT_FOUND`      | 200  | Domain 不存在                                 |
| `DOMAIN_EXISTS`         | 200  | Domain 已存在                                 |
| `RECORD_NOT_FOUND`      | 200  | Record 不存在                                 |
| `RECORD_EXISTS`         | 200  | Record 已存在（同 Domain 下重复记录）         |
| `RECORD_TYPE_INVALID`   | 200  | Record 类型字段非法（非 A/AAAA/SRV/TXT）      |
| `RECORD_ID_EXHAUSTED`   | 200  | Domain 下 record-id 序号达 9999 上限          |
| `SERVICE_UNAVAILABLE`   | 503  | etcd 服务不可用                               |
| `INTERNAL_ERROR`        | 500  | 服务器内部错误                                |

> `VALIDATION_ERROR` 可选携带 `error.details` 字段级错误列表（见 [API 设计规范](../conventions/api-design.md)）；简单校验场景可省略 `details`，仅返回 `code` 与 `message`。记录值与类型不匹配（如 `type=A` 但 `value` 为 IPv6）按 `VALIDATION_ERROR` 处理。

## 6. 接口清单

| 模块        | Method   | Path                      | 鉴权 | 说明                      |
| ----------- | -------- | ------------------------- | ---- | ------------------------- |
| 健康检查    | GET/POST | `/api/health`             | 公开 | 反映 etcd 连接状态        |
| 认证        | POST     | `/api/auth/login`         | 公开 | 用户登录                  |
| 认证        | POST     | `/api/auth/refresh`       | 公开 | 刷新 Token                |
| 当前用户    | POST     | `/api/me`                 | 登录 | 获取当前用户信息          |
| 当前用户    | POST     | `/api/me/change-password` | 登录 | 修改当前用户密码          |
| 用户管理    | POST     | `/api/user/list`          | 管理 | 列出所有用户              |
| 用户管理    | POST     | `/api/user/create`        | 管理 | 创建用户                  |
| 用户管理    | POST     | `/api/user/update`        | 管理 | 更新用户                  |
| 用户管理    | POST     | `/api/user/delete`        | 管理 | 删除用户                  |
| Zone 管理   | POST     | `/api/dns/zone/list`      | 登录 | 列出所有 Zone             |
| Zone 管理   | POST     | `/api/dns/zone/get`       | 登录 | 获取 Zone 详情            |
| Zone 管理   | POST     | `/api/dns/zone/create`    | 管理 | 创建 Zone                 |
| Zone 管理   | POST     | `/api/dns/zone/update`    | 管理 | 更新 Zone                 |
| Zone 管理   | POST     | `/api/dns/zone/delete`    | 管理 | 删除 Zone（级联）         |
| Domain 管理 | POST     | `/api/dns/domain/list`    | 登录 | 列出 Zone 下所有 Domain   |
| Domain 管理 | POST     | `/api/dns/domain/get`     | 登录 | 获取 Domain 详情          |
| Domain 管理 | POST     | `/api/dns/domain/create`  | 管理 | 创建 Domain               |
| Domain 管理 | POST     | `/api/dns/domain/update`  | 管理 | 更新 Domain               |
| Domain 管理 | POST     | `/api/dns/domain/delete`  | 管理 | 删除 Domain（级联）       |
| Record 管理 | POST     | `/api/dns/record/list`    | 登录 | 列出 Domain 下所有 Record |
| Record 管理 | POST     | `/api/dns/record/get`     | 登录 | 获取 Record 详情          |
| Record 管理 | POST     | `/api/dns/record/create`  | 登录 | 创建 Record               |
| Record 管理 | POST     | `/api/dns/record/update`  | 登录 | 更新 Record               |
| Record 管理 | POST     | `/api/dns/record/delete`  | 登录 | 删除 Record               |
