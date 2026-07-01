<p align="center">
  <picture>
    <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="300" />
  </picture>
</p>

<p align="center">
  <strong>AI Agent 控制平面</strong><br/>
  定义、部署、编排你的 AI Agent 舰队 —— 一个二进制文件即可运行。
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react&logoColor=white" alt="React" />
  <img src="https://img.shields.io/badge/Tailwind-3-06B6D4?style=flat&logo=tailwindcss&logoColor=white" alt="Tailwind" />
  <img src="https://img.shields.io/badge/Vite-5-646CFF?style=flat&logo=vite&logoColor=white" alt="Vite" />
  <img src="https://img.shields.io/badge/SQLite-3-003B57?style=flat&logo=sqlite&logoColor=white" alt="SQLite" />
  <img src="https://img.shields.io/badge/Docker-✓-2496ED?style=flat&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/i18n-ZH%2FEN-blue?style=flat" alt="i18n" />
</p>

<p align="center">
  <a href="https://github.com/mudern/agentbucket/blob/main/README.md">English Docs</a>
</p>

---

AgentBucket 是一个轻量级 AI Agent 控制平面。通过 TOML 清单定义 Agent，一键部署为 Docker 容器（自动注入 sidecar），并通过精致的 Web UI 或 REST API 统一管理。

## 功能特性

- **Agent 定义** — 通过 `agent.toml` 声明式定义 Agent 的模型、运行时、Skill 和 MCP 配置
- **一键部署** — 自动 Docker 构建 + 容器运行，包含 sidecar 注入、端口分配和健康监控
- **多模型支持** — DeepSeek、GLM、Kimi、MiniMax，Anthropic 兼容协议
- **多运行时** — Claude Code 和 Codex，同时支持本地和容器运行模式
- **实时 SSE 聊天** — 流式响应，Markdown 渲染 + 代码高亮 + 交互式选项按钮
- **Agent 总线** — Agent 之间相互发现、发送消息、协作通信（200 条内存缓冲 + SQLite 审计日志）
- **会话管理** — 每个 Agent 独立的聊天会话，支持历史记录、自动持久化和删除
- **Token 解析** — 通过 Sidecar 解析鉴权 Token，支持 Agent 级别的访问授权
- **前端界面** — 精致的仪表盘，带搜索的表格、能力选择器、部署进度监控
- **国际化** — 支持中文/英文 UI 切换，双语文档
- **Docker 原生** — DooD（Docker-out-of-Docker）部署，挂载 Docker socket 操作宿主机 Docker，非 DinD
- **API 优先** — 所有功能均可通过 curl 调用，适合 CI/CD 和 Agent 间通信

## 架构

```
┌──────────────────────────────────────────────────┐
│  AgentBucket 前端  (React + Vite + Tailwind)      │
├──────────────────────────────────────────────────┤
│  AgentBucket 后端  (Go 1.22 + SQLite)              │
│  ┌────────────────────────────────────────────┐   │
│  │  Agent 总线  (Agent 间对等通信)              │   │
│  └────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────┤
│  Docker Sidecar 集群  (自动编排)                   │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐         │
│  │ Agent 1  │ │ Agent 2  │ │ Agent N  │         │
│  │ :18043   │ │ :18239   │ │ :18020   │         │
│  │ClaudeCode│ │ClaudeCode│ │  Codex   │         │
│  └──────────┘ └──────────┘ └──────────┘         │
└──────────────────────────────────────────────────┘
```

## 快速开始

### 前置条件

- Go 1.22+
- Node.js 20+ / pnpm（前端开发需要）
- Docker（部署 Agent 容器时需要）
- AI provider token（自动从 `~/.config/ccs/providers/*.env` 导入）

### 本地开发

```bash
git clone git@github.com:mudern/agentbucket.git
cd agentbucket

# 安装前端依赖
pnpm install

# 启动后端
cd backend
go run ./cmd/server/
# => AgentBucket backend listening on http://127.0.0.1:8080

# 启动前端（另开终端）
pnpm dev
# => http://127.0.0.1:5177
```

### Docker 部署

```bash
docker-compose up -d
# => http://localhost:8080
```

后端挂载 `/var/run/docker.sock` 来管理宿主机 Docker daemon 上的 sidecar 容器——**不是 Docker-in-Docker**。

### 环境变量

| 变量 | 默认值 | 说明 |
|---|---|---|
| `AGENTBUCKET_ADDR` | `127.0.0.1:8080` | 监听地址 |
| `AGENTBUCKET_DATA_DIR` | `backend/.data` | SQLite 和构建文件存储 |
| `AGENTBUCKET_BUILD_TIMEOUT` | `300s` | Docker 构建超时 |
| `AGENTBUCKET_SIDECAR_HOST` | `127.0.0.1` | Sidecar 可达地址（Docker 中为 `host.docker.internal`） |
| `AGENTBUCKET_PROVIDERS_DIR` | `~/.config/ccs/providers` | CCS provider 环境变量文件目录 |
| `AGENTBUCKET_ADMIN_TOKEN` | 自动生成 | 管理员 API token |

## Agent 定义

```toml
# agents/my-agent/agent.toml
id              = "my-agent"
name            = "我的 Agent"
description     = "Agent 功能描述"
model           = "deepseek-v4-pro[1m]"
runtime         = "claudecode"
runtime_version = "latest"
api_token       = "deepseek"
skills          = ["knowledge-base", "web-browser"]
mcps            = ["github-mcp", "filesystem-mcp"]
extra_install   = ["apk add --no-cache github-cli"]
```

| 字段 | 说明 |
|---|---|
| `id` | Agent 唯一标识 |
| `name` | 显示名称 |
| `model` | AI 模型名称 |
| `runtime` | `claudecode` 或 `codex` |
| `api_token` | 关联的 AI Token 名称 |
| `skills` | 启用的技能目录 |
| `mcps` | MCP 服务器配置 |
| `extra_install` | Dockerfile 额外的 RUN 命令 |

## API 概览

完整 API 文档见 `.skills/agentbucket-admin/SKILL.md`。

### Agent 管理
```bash
GET    /api/agents
POST   /api/agent-definitions/scan
```

### 部署
```bash
GET    /api/deploy-options
POST   /api/deployments
GET    /api/deployments/{id}
POST   /api/deployments/{id}/start
POST   /api/deployments/{id}/stop
```

### 聊天与会话
```bash
GET    /api/agents/{id}/sessions
POST   /api/agents/{id}/sessions
DELETE /api/agents/{id}/sessions/{sessionId}
GET    /api/agents/{id}/messages?sessionId=xxx
POST   /api/agents/{id}/messages       # stream: true 启用 SSE 流式
```

### Agent 总线
```bash
GET    /api/bus/agents
POST   /api/bus/agents/{id}/register
POST   /api/bus/agents/{id}/message
GET    /api/bus/messages?toAgent=xxx
```

### Token 与仓库管理
```bash
GET    /api/ai-tokens          POST   /api/ai-tokens
GET    /api/auth-tokens        POST   /api/auth-tokens
GET    /api/repositories       POST   /api/repositories
PATCH  /api/repositories/{id}  DELETE /api/repositories/{id}
```

## 项目结构

```
AgentBucket/
├── backend/
│   ├── cmd/server/          # Go 后端（HTTP + SQLite + Docker 编排）
│   ├── cmd/sidecar/         # Sidecar（编译到每个部署镜像中）
│   ├── examples/agent-repo/ # 示例 Agent 定义
│   ├── tokens/              # Token 解析脚本
│   └── Dockerfile           # 生产环境 Docker 镜像
├── src/                     # React 前端
│   ├── pages/               # 页面组件
│   ├── components/          # 共享 UI 组件
│   ├── api/                 # API 调用层
│   └── i18n/                # 国际化（中文/英文）
├── .skills/                 # Claude Code 开发/管理技能
├── docker-compose.yml       # Docker 编排配置
└── README.md
```

## 许可

MIT License
