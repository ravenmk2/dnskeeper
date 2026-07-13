---
date: 2026-07-12
---

# DNS 数据同步设计

## 1. 概述

本文定义 dnskeeper 将 Record 期望状态同步至 CoreDNS etcd 服务记录的规则，作为 [数据存储设计](data-storage.md) 的姊妹篇，承接其"Record 增删改时同步至 CoreDNS 服务记录的机制另文描述"。

dnskeeper 与 CoreDNS 共用同一 etcd 实例：领域对象存于 `/dnskeeper/` 前缀，CoreDNS 服务记录存于 `/skydns/` 前缀（默认值，可配置，见第 5 节）。同一实例使两处写入可纳入单次 etcd 事务（`Txn`），从根本上消除双写漂移。

本文定义的规则**与触发时机解耦**：以 Domain 为同步单元（其下全部 Record 构成期望记录集），可应用于任意 Domain 集合——单个 Domain、某 Zone 下全部 Domain、或全部 Domain——规则本身不变，仅遍历范围与是否执行悬空清理不同。

## 2. CoreDNS etcd 记录模型

CoreDNS etcd 插件遵循 SkyDNS 约定：DNS 名按标签逆序映射为 etcd key 路径，记录值以 [SkyDNS message](https://github.com/skynetservices/skydns/blob/2fcff74cdc9f9a7dd64189a447ef27ac354b725f/msg/service.go#L26) 的 JSON 编码。详见 [CoreDNS etcd 插件参考](../references/coredns-etcd.md)。

### 2.1 Key 路径映射

DNS 名的各级标签逆序拼接，前置 `/skydns`，得到 CoreDNS 查找路径。Record 在 dnskeeper 中以 `(zone, domain, record-id)` 定位，映射规则：

- `zone` 各级标签逆序构成 Zone 段（Zone 可为任意级别 FQDN，通常二级/三级）；
- `domain` 为多级标签（可含 `.`，如 `www`、`www.beta`）时，其各级标签逆序追加在 Zone 段之后；
- `domain` 为 `@`（Zone 根）时，不追加任何段，记录直接落在 Zone 段下；
- `record-id` 作为叶子子键追加在 domain 段之后。

| Zone              | Domain     | record-id | 完整 DNS 名            | CoreDNS 记录 Key                    |
| ----------------- | ---------- | --------- | ---------------------- | ----------------------------------- |
| `example.com`     | `www`      | `0001`    | `www.example.com`      | `/skydns/com/example/www/0001`      |
| `example.com`     | `@`        | `0001`    | `example.com`          | `/skydns/com/example/0001`          |
| `example.com`     | `www.beta` | `0001`    | `www.beta.example.com` | `/skydns/com/example/beta/www/0001` |
| `sub.example.com` | `api`      | `0001`    | `api.sub.example.com`  | `/skydns/com/example/sub/api/0001`  |

> 完整 DNS 名的查询以前缀 `/skydns/{reversed-FQDN}/` 做范围查找，返回该前缀下全部子键（即 DNS RR），见 [coredns-etcd.md](../references/coredns-etcd.md) "Special Behavior"。

### 2.2 Service 消息格式

每条 CoreDNS 记录的值为一 JSON 对象（SkyDNS message），dnskeeper 按 `Record.type` 使用对应字段：

| type   | message                                                                    | 说明                                                                                     |
| ------ | -------------------------------------------------------------------------- | ---------------------------------------------------------------------------------------- |
| `A`    | `{"host":value,"ttl":ttl}`                                                 | `value` 为 IPv4 串，生成 A 记录。                                                        |
| `AAAA` | `{"host":value,"ttl":ttl}`                                                 | `value` 为 IPv6 串，生成 AAAA 记录；CoreDNS 按 IP 串格式判定 v4/v6，无需显式区分。       |
| `SRV`  | `{"host":value,"ttl":ttl,"priority":priority,"port":port,"weight":weight}` | `value` 为目标主机 FQDN（建议带尾点）；`priority`/`port`/`weight` 取自 Record 同名字段。 |
| `TXT`  | `{"text":value,"ttl":ttl}`                                                 | `value` 为文本，生成 TXT 记录。                                                          |

> dnskeeper 记录为持久 key（无 lease），DNS TTL 直接取自 `ttl` 字段；CoreDNS 的 `min/max-lease-ttl` 主要约束 lease 型记录，对持久 key 影响有限。

> CNAME/MX/PTR 等记录类型暂不支持（当前 Record 模型仅 A/AAAA/SRV/TXT），留作未来扩展。

### 2.3 子键命名（record-id → DNS RR）

一个 Domain 下每个 Record 在 CoreDNS 记录前缀下对应一个子键，**子键名直接取 record-id**：

| Zone          | Domain | Records                                          | ttl | CoreDNS 记录                                                                                                                           |
| ------------- | ------ | ------------------------------------------------ | --- | -------------------------------------------------------------------------------------------------------------------------------------- |
| `example.com` | `www`  | `A id=0001 192.168.1.1`、`A id=0002 192.168.1.2` | 300 | `/skydns/com/example/www/0001`=`{"host":"192.168.1.1","ttl":300}`<br>`/skydns/com/example/www/0002`=`{"host":"192.168.1.2","ttl":300}` |
| `example.com` | `@`    | `AAAA id=0001 2001:db8::1`                       | 600 | `/skydns/com/example/0001`=`{"host":"2001:db8::1","ttl":600}`                                                                          |

子键名 = record-id，与 dnskeeper Record 一一对应。前缀范围查询一次性返回子键名（即 record-id）与值（含类型字段），reconcile 的 diff 仅需比较子键名与一并返回的值，无需额外读取。

> **幻影子域查询**：record-id 为纯数字（不含 `.`），查询 `{record-id}.{domain}` 会经 CoreDNS 两段查找命中该记录（详见 [coredns-etcd.md](../references/coredns-etcd.md) "Special Behavior"）。此类名字无人主动查询，且记录本可经正式域名直接查到，无实质信息泄露；为换取 reconcile 的 1:1 简洁性，设计接受此副作用。旧设计以 IP 串作子键（含 `.`）规避此问题，但新模型 TXT/SRV 值不适合作子键，故改用 record-id。

> record-id 不复用（见 [data-storage.md](data-storage.md) §2），故子键名与 Record 内容的对应关系永久稳定，不会因 id 复用产生内容漂移。

## 3. 同步规则

同步以"期望记录集 → 实际收敛"建模，不绑定特定操作语义。

### 3.1 同步单元与期望记录集

同步单元为一个 Domain，由 `(zone, domain)` 定位。其**期望记录集**取自该 Domain 下全部 Record：

- 记录前缀：按 2.1 由 `(zone, domain)` 推导，即 `/skydns/{reversed-FQDN}/`；
- 期望子键集：该 Domain 下每个 Record 对应前缀下一个子键（子键名 = record-id）；
- 期望值：按 2.2 由 `Record.type` 与字段构成 SkyDNS message。

### 3.2 reconcile 算法

对该 Domain 的记录前缀执行收敛，步骤幂等、可重入：

1. 列出该前缀下全部子键（前缀范围查询，返回 key 与 value），得**实际记录集** `{record-id → message}`；
2. 与**期望记录集**求 diff：
    - 新增（期望 id 不在实际）：`Put` 子键 = 期望 message；
    - 移除（实际 id 不在期望）：`Delete` 对应子键；
    - 保留但 message 不一致（任一字段变化）：`Put` 子键（覆盖以更新）；
    - 保留且 message 一致：跳过。

该算法满足"系统自动比较新旧记录集，新增/删除差异记录"的可观察行为：仅对差异与字段变化做最小写入。

### 3.3 期望记录集为空

当 Domain 不存在（对象已删除），或其下无 Record 时，期望记录集为空：

- 删除该 Domain 的记录前缀（`/skydns/{reversed-FQDN}/` 前缀范围删，一次清空其下全部子键）。

此规则覆盖 Domain 删除与"Domain 下无 Record"两种状态收敛。

### 3.4 Domain 级移除

当某 Domain 及其全部 Record 不再存在，CoreDNS 侧对应 Domain 反转前缀下的记录应整体清除：

- 删除 `/skydns/{reversed-FQDN}/` 前缀（Domain 全部 CoreDNS 记录）；
- 删除 `/dnskeeper/dns/{zone}/{domain}/` 前缀（全部 Record 对象）；
- 删除 `/dnskeeper/dns/{zone}/{domain}`（Domain 对象）。

### 3.5 Zone 级移除

当某 Zone 及其全部 Domain/Record 不再存在，CoreDNS 侧对应整个 Zone 反转前缀下的记录应整体清除：

- 删除 `/skydns/{reversed-zone}/` 前缀（一次范围删清掉该 Zone 全部 CoreDNS 记录）；
- 删除 `/dnskeeper/dns/{zone}/` 前缀（全部 Domain 与 Record 对象）；
- 删除 `/dnskeeper/dns/{zone}`（Zone 对象）。

其中 `{reversed-zone}` 为 Zone 各级标签逆序拼接，如 `example.com` → `com/example`，`sub.example.com` → `com/example/sub`。

> 创建或更新 Zone 与 Domain 不产生 CoreDNS 同步：CoreDNS 的权威 Zone 由 Corefile 定义，etcd 仅承载记录而非 Zone/Domain 元数据；Zone 与 Domain 为 dnskeeper 自管的统计实体。

### 3.6 悬空记录清理

CoreDNS 记录须由 Record 对象支撑。无对应 Record 的记录为**悬空**，应删除。

集合级同步在逐个 Domain 执行 3.2 后，再做一次 Zone 级对账：

1. 列出 `/skydns/{reversed-zone}/` 下全部子键（含值，一次范围查询）；
2. 对每个子键，剥离 `/skydns/{reversed-zone}/` 前缀，按 `/` 切分得相对路径段序列：
    - 末段 = record-id；
    - 其余段为 domain 的逆序标签，将其**逆序**后以 `.` 连接得 domain（空序列 → `@`）；
3. 据此还原 `(zone, domain, record-id)`，若 `/dnskeeper/dns/{zone}/{domain}/{record-id}` 不存在，则该子键为悬空，删除。

> 多级 domain（含 `.`）在 CoreDNS 路径中被拆为多段逆序；还原时逆序再拼回，与正向映射互逆。同一 Domain 下多出的非法字段（Record 存在但 message 与期望不符）由 3.2 reconcile 覆盖，不属悬空。record-id 不复用，故已删除的 id 永不会被重新分配给不同内容，悬空识别无歧义。

### 3.7 应用粒度

上述规则以 Domain 为单元，可应用于任意 Domain 集合，规则本身不变：

- **单 Domain 级**：对单个 Domain 执行 3.2（或 3.3）。
- **集合级**：对集合内每个 Domain 逐个套用 3.2–3.3，之后执行 3.6 悬空清理，以保证 CoreDNS 侧与期望状态全局一致。

集合可为某 Zone 下全部 Domain，或全部 Domain；差异仅为遍历范围与是否执行悬空清理。

## 4. 原子性与一致性

dnskeeper 与 CoreDNS 共用同一 etcd，单次 `Txn` 可原子写 `/dnskeeper/` 与 `/skydns/` 双前缀，消除双写漂移窗口。`Txn` 支持混合 `Put` 与范围 `Delete`，DNS 对象的增删改与对应 CoreDNS 子键写入合为一个 `Txn`：

- 单 Record 创建：Record `Put` + Domain 实体 `Put`（`last_record_id`/`record_count` 递增）+ skydns 子键 `Put`，合为一个 `Txn`；
- 单 Record 更新：Record `Put` + skydns 子键 `Put`，合为一个 `Txn`；
- 单 Record 删除：Record `Delete` + Domain 实体 `Put`（`record_count` 递减，`last_record_id` 不回退）+ skydns 子键 `Delete`，合为一个 `Txn`；
- Domain 级移除：`/skydns/{reversed-FQDN}/` 范围删 + `/dnskeeper/dns/{zone}/{domain}/` 范围删 + Domain 对象删除 + Zone 实体 `Put`（`domain_count` 递减），合为一个 `Txn`；
- Zone 级移除：`/skydns/{reversed-zone}/` 范围删 + `/dnskeeper/dns/{zone}/` 范围删 + Zone 对象删除，合为一个 `Txn`。

> etcd `Txn` 默认受 `--max-txn-ops`（默认 128）限制。单 Record 操作的 ops 数远低于此；集合级同步应按 Domain 分批提交 `Txn`，每批不超过 ops 上限。

> 与 [data-storage.md](data-storage.md) 第 4 节的关系：彼处 User 对象低并发暂不启用 `Txn`；此处涉及 CoreDNS 服务记录的对外一致性，双写漂移会直接导致 DNS 解析错误，一致性收益高，故 DNS 对象统一启用 `Txn`。

## 5. 配置依赖

| 配置项                    | 默认值    | 说明                                                                                     |
| ------------------------- | --------- | ---------------------------------------------------------------------------------------- |
| CoreDNS 记录前缀 (`path`) | `/skydns` | 须与 CoreDNS Corefile 中 `etcd` 插件的 `path` 一致；dnskeeper 据此拼接所有 CoreDNS key。 |
| etcd endpoint             | —         | 复用 dnskeeper 既有的 etcd 连接，无需额外配置。                                          |

> **`/skydns` 仅为默认前缀，可通过 dnskeeper 配置文件修改**；修改后须与 CoreDNS Corefile 的 `path` 保持一致，否则 dnskeeper 写入的 key 不在 CoreDNS 查找范围内。

> 反向 PTR / `in-addr.arpa` 记录暂不在本文范围：Record 模型仅含前向 A/AAAA/SRV/TXT，无 PTR 概念；未来若引入反向记录再行扩展。

## 6. 设计取舍

- **子键命名取 record-id**：与 Record 一一对应，reconcile 按子键名比 diff；接受 dotless id 的幻影子域副作用（2.3）。
- **record-id 不复用**：Domain 内递增序号、删除不回退，id 稳定永不重复——悬空记录识别无歧义，同步语义清晰（见 [data-storage.md](data-storage.md) §2）。
- **同步以 reconcile 建模而非按操作建模**：规则描述"期望 → 收敛"的状态转移，与触发时机解耦，单 Domain 与集合级同一套规则，仅遍历范围与悬空清理与否不同。
- **多类型记录共用 reconcile 框架**：A/AAAA/SRV/TXT 仅 message 构造按类型不同，diff 与收敛逻辑统一。
- **集合级引入悬空清理**：单 Domain 的 reconcile 只能保证被同步 Domain 自身正确，无法清除无主记录；悬空清理以 Zone 反转前缀为单位对账，使 CoreDNS 侧全局收敛。
- **原子性启用 Txn**：共用 etcd 的天然优势，双写可原子；DNS 解析对一致性敏感，漂移代价高，故有别于领域对象存储中 User 的"暂不启用"取舍，DNS 对象统一启用。
- **记录类型限 A/AAAA/SRV/TXT**：贴合当前 Record 模型；CNAME/MX/PTR 待模型扩展后再纳入。
