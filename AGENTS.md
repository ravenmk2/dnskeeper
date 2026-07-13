# AGENTS.md

Dnskeeper 是一个 CoreDNS + Etcd 插件的管理工具

## 文档索引

```txt
docs/
├── api/
│   ├── overview.md            # API 概览
│   ├── auth.md                # 认证模块
│   ├── domain.md              # Domain 管理
│   ├── health.md              # 健康检查
│   ├── me.md                  # 当前用户
│   ├── record.md              # Record 管理
│   ├── user.md                # 用户管理
│   └── zone.md                # Zone 管理
├── conventions/
│   └── api-design.md          # RPC 风格 API 设计规范
├── design/
│   ├── data-storage.md        # 数据存储设计
│   └── dns-sync.md            # DNS 数据同步设计
└── references/
    └── coredns-etcd.md        # CoreDNS etcd 插件参考
```

## 文档写作风格

- 语言保持简洁、精确，逻辑清晰、一致。
- 使用公共规范的标准术语，禁止自编自造。
- 使用结构化的方式表达，避免冗长的段落和流水账。
