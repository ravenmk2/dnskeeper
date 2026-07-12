---
date: 2026-06-26
---

# RPC Style HTTP API

RPC Style HTTP API 以"操作"为建模单位, 每个端点对应一个独立的动作, 参数统一通过请求体传递。区别于以资源为中心的 REST, 它不依赖 HTTP 动词区分 CRUD 语义, 更适合动作语义复杂、流程驱动的业务场景。

## HTTP Method

HTTP Method 总是使用 `POST`。

操作语义已在 URL 路径中显式表达, 无需依赖 HTTP 动词区分 CRUD; 请求参数统一通过请求体传递, 便于网关、拦截器统一处理与扩展。

## Status Code

推荐统一返回 `200`, 通过响应信封的 `success`/`error` 字段区分业务结果的成功与失败; `4xx`/`5xx` 仅用于传输层与系统层错误, 例如鉴权失败、限流、服务不可用等。

这样设计的好处:

- 客户端只需解析响应体即可判断业务结果, 处理逻辑统一。
- 网关与代理无需感知业务语义, 一致性更好。
- 业务错误与传输错误解耦, 职责清晰。

## URL Path

- 路径与操作一一对应, 方便访问控制、统计和日志定位。
- 使用 `Kebab Case` 命名。
- 由模块、分组和操作组成; 操作名使用动词, 如 `create`/`get`/`update`/`delete`/`list` 等。
- 资源名使用单数形式。
- 参数统一通过请求体传递, 路径中不使用路径参数。
- 总是以 `/api` 为前缀 (可选)。

格式:

- `/api/{module}/{action}`
- `/api/{module}/{group}/{action}`

示例:

- `/api/user/create`
- `/api/user/delete`
- `/api/user/batch-delete`

## Content-Type

请求与响应的 `Content-Type` 总是使用 `application/json`。

## 响应消息

- 响应消息信封在统一返回 `200` 的情况下区分成功与失败。
- 响应信封的属性使用 `Camel Case` 命名。

字段说明:

| 字段                      | 类型           | 说明                                    |
| ------------------------- | -------------- | --------------------------------------- |
| `success`                 | boolean        | 业务是否成功。                          |
| `data`                    | object \| null | 成功时返回的业务数据, 失败时为 `null`。 |
| `error`                   | object \| null | 失败时的错误信息, 成功时为 `null`。     |
| `error.code`              | string         | 错误码, 大写下划线命名。                |
| `error.message`           | string         | 面向用户的错误描述。                    |
| `error.details`           | array          | 可选, 校验类错误的详细字段列表。        |
| `error.details[].code`    | string         | 字段级错误码。                          |
| `error.details[].message` | string         | 字段级错误描述。                        |
| `error.details[].target`  | string         | 出错的字段名。                          |

成功:

```jsonc
{
    "success": true,
    "error": null,
    "data": {}
}
```

数据分页:

分页响应的 `data` 结构如下:

| 字段        | 类型   | 说明                  |
| ----------- | ------ | --------------------- |
| `items`     | array  | 当前页数据列表。      |
| `page`      | number | 当前页码, 从 1 开始。 |
| `size`      | number | 每页大小。            |
| `total`     | number | 数据总条数。          |
| `pageCount` | number | 总页数。              |

```jsonc
{
    "success": true,
    "error": null,
    "data": {
        "items": [{}],
        "page": 1,
        "size": 10,
        "total": 100,
        "pageCount": 10
    }
}
```

错误:

```jsonc
{
    "success": false,
    "data": null,
    "error": {
        "code": "XXX_ERROR",
        "message": "User not exists"
    }
}
```

详细错误信息:

```jsonc
{
    "success": false,
    "data": null,
    "error": {
        "code": "VALIDATION_ERROR",
        "message": "Validation failed.",
        "details": [
            { "code": "USERNAME_EXISTS", "message": "Username already exists", "target": "username" },
            { "code": "WEAK_PASSWORD", "message": "Password is too weak", "target": "password" }
        ]
    }
}
```

## 错误码设计

- 命名规范: 使用大写下划线 (`UPPER_SNAKE_CASE`), 建议带模块前缀以避免冲突, 例如 `USER_NOT_FOUND`、`ORDER_OUT_OF_STOCK`。
- 分类建议:
  - **校验类**: 参数或业务规则校验失败, 如 `VALIDATION_ERROR`, 通常配合 `details` 给出字段级错误。
  - **业务类**: 业务流程无法继续, 如 `USERNAME_EXISTS`、`USER_NOT_FOUND`。
  - **系统类**: 系统级错误, 如 `INTERNAL_ERROR`、`SERVICE_UNAVAILABLE`, 通常不向终端用户暴露细节。
- `code` 稳定不变, 客户端应基于 `code` 而非 `message` 进行逻辑判断。
- `message` 面向用户, 应清晰可读; 敏感信息不应出现在 `message` 中。
