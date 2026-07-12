---
date: 2026-07-12
---

# DNS 数据同步设计

## 1. 概述

本文定义 dnskeeper 将 Domain 期望状态同步至 CoreDNS etcd 服务记录的规则，作为 [数据存储设计](data-storage.md) 的姊妹篇，承接其"Domain 增删改时同步至 CoreDNS 服务记录的机制另文描述"。

dnskeeper 与 CoreDNS 共用同一 etcd 实例：领域对象存于 `/dnskeeper/` 前缀，CoreDNS 服务记录存于 `/skydns/` 前缀（默认值，可配置，见第 5 节）。同一实例使两处写入可纳入单次 etcd 事务（`Txn`），从根本上消除双写漂移。

本文定义的规则**与触发时机解耦**：以 Domain 为同步单元，可应用于任意 Domain 集合——单个 Domain、某 Zone 下全部 Domain、或全部 Domain——规则本身不变，仅遍历范围与是否执行孤儿清理不同。

## 2. CoreDNS etcd 记录模型

CoreDNS etcd 插件遵循 SkyDNS 约定：DNS 名按标签逆序映射为 etcd key 路径，记录值以 [SkyDNS message](https://github.com/skynetservices/skydns/blob/2fcff74cdc9f9a7dd64189a447ef27ac354b725f/msg/service.go#L26) 的 JSON 编码。详见 [CoreDNS etcd 插件参考](../references/coredns-etcd.md)。

### 2.1 Key 路径映射

DNS 名的各级标签逆序拼接，前置 `/skydns`，得到 CoreDNS 查找路径。Domain 在 dnskeeper 中以 `(zone, domain)` 定位，映射规则：

- `zone` 各级标签逆序构成 Zone 段（Zone 可为任意级别 FQDN）；
- `domain` 为子域名标签（如 `www`，单标签、不含 `.`）时，追加在 Zone 段之后；
- `domain` 为 `@`（Zone 根）时，不追加任何段，记录直接落在 Zone 段下。

| Zone              | Domain | 完整 DNS 名           | CoreDNS 记录前缀               |
| ----------------- | ------ | --------------------- | ------------------------------ |
| `example.com`     | `www`  | `www.example.com`     | `/skydns/com/example/www/`     |
| `example.com`     | `@`    | `example.com`         | `/skydns/com/example/`         |
| `sub.example.com` | `api`  | `api.sub.example.com` | `/skydns/com/example/sub/api/` |

> 记录前缀以 `/` 结尾：CoreDNS 对 `www.example.com` 的查询以前缀 `/skydns/com/example/www/` 做范围查找，返回该前缀下全部子键（即 DNS RR），见 [coredns-etcd.md](../references/coredns-etcd.md) "Special Behavior"。

### 2.2 Service 消息格式

每条 CoreDNS 记录的值为一 JSON 对象，dnskeeper 仅使用 `host` 与 `ttl` 两个字段：

```jsonc
{"host": "192.168.1.1", "ttl": 300}
```

| 字段   | 类型   | 说明                                                                                                                                                                  |
| ------ | ------ | --------------------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `host` | string | 主机地址。IPv4 串生成 A 记录，IPv6 串生成 AAAA 记录，由 CoreDNS 按 IP 串格式自动判定，无需显式区分类型。                                                              |
| `ttl`  | int    | DNS 缓存 TTL，单位秒，取自 Domain 的 `ttl`。dnskeeper 记录为持久 key（无 lease），DNS TTL 直接取自 `ttl` 字段；CoreDNS 的 `min/max-lease-ttl` 主要约束 lease 型记录。 |

Domain 的 `ttl` 约束为 1–86400 秒（见 [domain.md](../api/domain.md)），与 DNS TTL 语义一致，直接填入 `ttl` 字段。

> SRV、TXT、CNAME 等记录类型依赖 `priority`/`port`/`text` 等字段，当前 Domain 模型仅含 `ips`，暂不支持，留作未来扩展。

### 2.3 子键命名约定（多 IP → DNS RR）

一个 Domain 可有多个 IP（IPv4 与 IPv6 混排，自动去重），每个 IP 写为记录前缀下的一个子键，构成 DNS RR。**子键名直接取 IP 串本身**：

| Zone          | Domain | ips                              | ttl | CoreDNS 记录                                                                                                                                             |
| ------------- | ------ | -------------------------------- | --- | -------------------------------------------------------------------------------------------------------------------------------------------------------- |
| `example.com` | `www`  | `["192.168.1.1","192.168.1.2"]`  | 300 | `/skydns/com/example/www/192.168.1.1` = `{"host":"192.168.1.1","ttl":300}`<br>`/skydns/com/example/www/192.168.1.2` = `{"host":"192.168.1.2","ttl":300}` |
| `example.com` | `@`    | `["192.168.1.10","2001:db8::1"]` | 600 | `/skydns/com/example/192.168.1.10` = `{"host":"192.168.1.10","ttl":600}`<br>`/skydns/com/example/2001:db8::1` = `{"host":"2001:db8::1","ttl":600}`       |

子键名自描述、与记录内容一一对应。前缀范围查询一次性返回子键名（即 IP）与值（含 `ttl`），故 reconcile 的 diff 无需额外读取。

> IPv6 地址应规范化（小写、压缩形式）后再作为子键，避免同一地址的不同文本表示产生重复子键。

**安全性**：IP 串中的 `.`（IPv4）与 `:`（IPv6）作为单段内的字面字符，不被 etcd 视为路径分隔符，亦不被 CoreDNS 拆为子域名标签。对 `192.168.1.1.www.example.com` 的查询会映射到 `/skydns/com/example/www/192/168/1/1/`，与存储的单段键 `/skydns/com/example/www/192.168.1.1` 不匹配，故不产生幻影子域查询（见 [coredns-etcd.md](../references/coredns-etcd.md) "Special Behavior"）。

## 3. 同步规则

同步以"期望记录集 → 实际收敛"建模，不绑定特定操作语义。

### 3.1 同步单元与期望记录集

同步单元为一个 Domain，由 `(zone, domain)` 定位。其**期望记录集**取自该 Domain 对象的 `ips` 与 `ttl`：

- 记录前缀：按 2.1 规则由 `(zone, domain)` 推导；
- 期望子键集：`ips` 中每个 IP 对应前缀下一个子键（子键名 = IP 串）；
- 期望值：`{"host":ip,"ttl":ttl}`。

### 3.2 reconcile 算法

对单个 Domain 的记录前缀执行收敛，步骤幂等、可重入：

1. 列出该前缀下全部子键（前缀范围查询，返回 key 与 value），得到**实际记录集** `{IP → ttl}`；
2. 与**期望记录集**求 diff：
   - 新增（期望 IP 不在实际中）：`Put` 子键 = `{"host":ip,"ttl":ttl}`；
   - 移除（实际 IP 不在期望中）：`Delete` 对应子键；
   - 保留且 `ttl` 不一致：`Put` 子键（覆盖以更新 `ttl`）；
   - 保留且 `ttl` 一致：跳过。

该算法满足 [domain.md](../api/domain.md) 中"系统自动比较新旧 IP 列表，新增/删除差异 IP"的可观察行为：仅对差异与 `ttl` 变化做最小写入。

### 3.3 期望记录集为空

当 Domain 不存在（对象已删除），或其 `ips` 为空数组时，期望记录集为空：

- 删除该 Domain 的记录前缀（前缀范围删，一次完成），清空其下全部子键。

此规则覆盖 Domain 删除与"更新为空 `ips`"两种状态收敛（见 [domain.md](../api/domain.md) 删除级联与空数组语义）。

### 3.4 Zone 级移除

当某 Zone 及其全部 Domain 不再存在时，CoreDNS 侧对应整个 Zone 反转前缀下的记录应整体清除：

- 删除 `/skydns/{reversed-zone}/` 前缀（一次范围删清掉该 Zone 全部 CoreDNS 记录）；
- 删除 `/dnskeeper/domains/{zone}/` 前缀（全部 Domain 对象）；
- 删除 `/dnskeeper/zones/{zone}`（Zone 对象）。

其中 `{reversed-zone}` 为 Zone 各级标签逆序拼接，如 `example.com` → `com/example`，`sub.example.com` → `com/example/sub`。

> 创建或更新 Zone 不产生 CoreDNS 同步：CoreDNS 的权威 Zone 由 Corefile 定义，etcd 仅承载记录而非 Zone 元数据；Zone 对象是 dnskeeper 自管的统计实体。

### 3.5 孤儿记录清理

CoreDNS 记录须由 Domain 对象支撑。无对应 Domain 的记录为**孤儿**，应删除。

集合级同步在逐个 Domain 执行 3.2 后，再做一次 Zone 级对账：

1. 列出 `/skydns/{reversed-zone}/` 下全部子键（含值，一次范围查询）；
2. 对每个子键，剥离 `/skydns/{reversed-zone}/` 前缀，取末段为实例键（IP），其前段还原为 Domain 标签：
   - 剩余为 `{label}/{instance}` → `domain = {label}`；
   - 剩余为 `{instance}` → `domain = @`（Zone 根）；
   - 剩余多于两段 → 当前模型不产生，视为孤儿；
3. 据此还原 `(zone, domain)`，若 `/dnskeeper/domains/{zone}/{domain}` 不存在，则该子键为孤儿，删除。

> 孤儿清理覆盖 Domain 对象被绕过 dnskeeper 直接删除、或上次同步中断所致的漂移。同一 Domain 下多出的非法 IP（Domain 存在但 IP 不在其 `ips` 中）由 3.2 reconcile 清除，不属于孤儿。

### 3.6 应用粒度

上述规则以 Domain 为单元，可应用于任意 Domain 集合，规则本身不变：

- **单 Domain 级**：对单个 Domain 执行 3.2（或 3.3）。
- **集合级**：对集合内每个 Domain 逐个套用 3.2–3.3，之后执行 3.5 孤儿清理，以保证 CoreDNS 侧与期望状态全局一致。

集合可为某 Zone 下全部 Domain，或全部 Domain；差异仅为遍历范围与是否执行孤儿清理。

## 4. 原子性与一致性

dnskeeper 与 CoreDNS 共用同一 etcd，单次 `Txn` 可原子写 `/dnskeeper/` 与 `/skydns/` 双前缀，消除双写漂移窗口。`Txn` 支持混合 `Put` 与范围 `Delete`，故单 Domain reconcile、Zone 级移除均可纳入一个 `Txn`：

- 单 Domain：其全部新增/移除/重写子键 + Domain 对象写入，合为一个 `Txn`；
- Zone 级移除：`/skydns/{reversed-zone}/` 范围删 + `/dnskeeper/domains/{zone}/` 范围删 + `/dnskeeper/zones/{zone}` 删除，合为一个 `Txn`。

> etcd `Txn` 默认受 `--max-txn-ops`（默认 128）限制。单 Domain 的 IP 数通常远低于此；集合级同步应按 Domain 分批提交 `Txn`，每批不超过 ops 上限。

与 [数据存储设计](data-storage.md) 第 4 节"暂未启用事务"的关系：彼处针对领域对象自身的跨 key 操作，为简化暂缓启用；此处涉及 CoreDNS 服务记录的对外一致性，双写漂移会直接导致 DNS 解析错误，一致性收益高，故启用 `Txn`。

## 5. 配置依赖

| 配置项                    | 默认值    | 说明                                                                                     |
| ------------------------- | --------- | ---------------------------------------------------------------------------------------- |
| CoreDNS 记录前缀 (`path`) | `/skydns` | 须与 CoreDNS Corefile 中 `etcd` 插件的 `path` 一致；dnskeeper 据此拼接所有 CoreDNS key。 |
| etcd endpoint             | —         | 复用 dnskeeper 既有的 etcd 连接，无需额外配置。                                          |

> **`/skydns` 仅为默认前缀，可通过 dnskeeper 配置文件修改**；修改后须与 CoreDNS Corefile 的 `path` 保持一致，否则 dnskeeper 写入的 key 不在 CoreDNS 查找范围内。

> 反向 PTR / `in-addr.arpa` 记录暂不在本文范围：Domain 模型仅含 `ips`（前向 A/AAAA），无 PTR 概念；未来若引入反向记录再行扩展。

## 6. 设计取舍

- **子键命名取 IP 串**：自描述，子键集合即 IP 集合，reconcile 的 diff 仅需比较子键名与（范围查询一并返回的）值，无需额外读取；且不产生幻影子域查询（见 2.3）。
- **同步以 reconcile 建模而非按操作建模**：规则描述"期望 → 收敛"的状态转移，与触发时机解耦，单 Domain 与集合级同一套规则，仅遍历范围与孤儿清理与否不同；附带满足 [domain.md](../api/domain.md) 的差异增删承诺。
- **集合级引入孤儿清理**：单 Domain 的 reconcile 只能保证被同步 Domain 自身正确，无法清除无主记录；孤儿清理以 Zone 反转前缀为单位对账，使 CoreDNS 侧全局收敛。
- **原子性启用 Txn**：共用 etcd 的天然优势，双写可原子；DNS 解析对一致性敏感，漂移代价高，故有别于领域对象存储的"暂未启用"取舍，此处明确启用。
- **记录类型限前向 A/AAAA**：贴合当前 Domain 模型（`ips`）；SRV/TXT/PTR 待模型扩展后再纳入。
