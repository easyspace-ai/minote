# 后端开发规范

> **目标读者：AI 代码生成模型（如 ChatGPT / Claude Code）**  
> **目标：在本项目中稳定、可预测地生成高质量 Go 后端代码**

本规范适用于 **YouMind / Minote** 项目后端开发，采用 **Go 1.25 + Eino + 标准库 net/http + GORM + 分层架构**。

---

## 技术栈

| 技术 | 版本 | 用途 |
|------|------|------|
| Go | 1.25+ | 编程语言 |
| Eino | 0.8+ | LLM 应用框架 |
| 标准库 net/http | 1.22+ | HTTP 服务（Go 1.22+ 路由） |
| GORM | 2.x | ORM 框架 |
| PostgreSQL / SQLite | - | 数据库 |
| 标准库 log | - | 日志框架 |

---

## 目录

| 序号 | 规范文件 | 说明 |
|------|---------|------|
| 01 | [总体原则](./01-总体原则.md) | 必须遵守的核心原则 |
| 02 | [项目分层](./02-项目分层.md) | 目录结构和分层职责 |
| 03 | [Handler 规范](./03-Handler规范.md) | HTTP 接口层规范（标准库 net/http） |
| 04 | [Service 规范](./04-Service规范.md) | 业务编排层规范 |
| 05 | [错误处理规范](./05-错误处理规范.md) | 错误定义和处理方式 |
| 06 | [Domain 规范](./06-Domain规范.md) | 领域模型规范 |
| 07 | [Repository 规范](./07-Repository规范.md) | 数据访问层规范 |
| 08 | [Model 规范](./08-Model规范.md) | 数据库模型规范 |
| 09 | [GORM 使用规范](./09-GORM使用规范.md) | GORM 强制规则 |
| 10 | [事务规范](./10-事务规范.md) | 事务控制规范 |
| 11 | [DTO 规范](./11-DTO规范.md) | 数据传输对象规范 |
| 12 | [Response 规范](./12-Response规范.md) | 统一响应结构 |
| 13 | [依赖注入规范](./13-依赖注入规范.md) | 依赖管理方式 |
| 14 | [测试规范](./14-测试规范.md) | 各层测试策略 |
| 15 | [禁止清单](./15-禁止清单.md) | 红线规则 |

---

## 快速导航

### 必读
1. [总体原则](./01-总体原则.md) - 核心约束
2. [项目分层](./02-项目分层.md) - 架构基础
3. [禁止清单](./15-禁止清单.md) - 红线规则

### 核心层规范
- [Handler 规范](./03-Handler规范.md) - HTTP 入口（标准库 net/http）
- [Service 规范](./04-Service规范.md) - 业务逻辑
- [Repository 规范](./07-Repository规范.md) - 数据访问

### 数据相关
- [Domain 规范](./06-Domain规范.md) - 领域模型
- [Model 规范](./08-Model规范.md) - 数据库模型
- [DTO 规范](./11-DTO规范.md) - 数据传输对象

### GORM 相关
- [GORM 使用规范](./09-GORM使用规范.md) - 使用规则
- [事务规范](./10-事务规范.md) - 事务控制

### 工程实践
- [错误处理规范](./05-错误处理规范.md) - 错误处理
- [Response 规范](./12-Response规范.md) - 统一响应
- [依赖注入规范](./13-依赖注入规范.md) - 依赖管理
- [测试规范](./14-测试规范.md) - 测试策略

---

## 核心约束

```
违反分层 = 错误实现
混用数据结构 = 错误实现
跨层调用 = 错误实现
```

---

## YouMind 项目结构

```
github.com/easyspace-ai/minote/
├── cmd/                      # 服务入口
│   ├── gateway/             # 统一网关 (:8080)
│   ├── notex/               # 核心业务 (:8787)
│   ├── agent/               # Agent 执行
│   ├── langgraph/           # LangGraph 兼容
│   └── ...
├── internal/                # 内部实现
│   └── notex/              # Notex 内部模块
│       ├── server_core.go   # HTTP 服务与路由
│       ├── types.go         # 领域模型
│       └── *_domain.go      # 领域逻辑
├── pkg/                     # 共享包
│   ├── agent/              # Agent 实现 (Eino)
│   ├── llm/                # LLM 客户端与工具
│   ├── tools/              # 工具注册与实现
│   ├── langgraphcompat/    # LangGraph API 兼容
│   └── ...
└── skills/                  # Skills YAML 配置
```

---

> **注意**: 本项目使用 Go 1.22+ 标准库 `net/http` 和新的路由模式，**不使用 Gin 框架**。
