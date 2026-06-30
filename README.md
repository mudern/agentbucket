<p align="center">
  <img src="public/logo.svg" alt="AgentBucket Logo" width="120" height="120" />
</p>

<h1 align="center">AgentBucket</h1>

<p align="center">
  AI Agent 控制平面 —— 一站管理、部署、编排你的 AI Agent 舰队
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat-square&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat-square&logo=react&logoColor=white" alt="React" />
  <img src="https://img.shields.io/badge/Tailwind-3-06B6D4?style=flat-square&logo=tailwindcss&logoColor=white" alt="Tailwind" />
  <img src="https://img.shields.io/badge/Docker-✓-2496ED?style=flat-square&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/SQLite-✓-003B57?style=flat-square&logo=sqlite&logoColor=white" alt="SQLite" />
  <img src="https://img.shields.io/badge/Vite-5-646CFF?style=flat-square&logo=vite&logoColor=white" alt="Vite" />
  <img src="https://img.shields.io/badge/license-MIT-blue?style=flat-square" alt="License" />
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Runtime-Claude%20Code-8B5CF6?style=flat-square" alt="Claude Code" />
  <img src="https://img.shields.io/badge/Runtime-Codex-10A37F?style=flat-square" alt="Codex" />
  <img src="https://img.shields.io/badge/Provider-DeepSeek-4F46E5?style=flat-square" alt="DeepSeek" />
  <img src="https://img.shields.io/badge/Provider-GLM-0284C7?style=flat-square" alt="GLM" />
  <img src="https://img.shields.io/badge/Provider-Kimi-7C3AED?style=flat-square" alt="Kimi" />
  <img src="https://img.shields.io/badge/Provider-MiniMax-F97316?style=flat-square" alt="MiniMax" />
</p>

---

AgentBucket 是一个轻量级的 AI Agent 控制平面。你可以用它来：

- **管理 Agent** — 通过 TOML 文件定义 Agent 的模型、Runtime、Skill、MCP
- **部署 Agent** — 一键 Docker 化部署，支持 extra install 自定义依赖
- **多模型切换** — DeepSeek / GLM / Kimi / MiniMax，Anthropic 兼容协议
- **多 Runtime** — Claude Code + Codex，本地和容器双模式
- **Agent 总线** — Agent 之间互相发现、发消息、协作
- **前端聊天** — Markdown 渲染 + 代码高亮 + 交互式选项

## 架构

```
┌──────────────────────────────────────────────────┐
│                AgentBucket UI                     │
│              React + Tailwind                     │
│              http://127.0.0.1:5177                │
├──────────────────────────────────────────────────┤
│                AgentBucket Backend                │
│              Go + SQLite                          │
│              http://127.0.0.1:8080                │
│  ┌────────────────────────────────────────────┐  │
│  │            Agent Bus (总 线)                │  │
│  │     Agent ↔ Agent 互通，最近 200 条消息     │  │
│  └────────────────────────────────────────────┘  │
├──────────────────────────────────────────────────┤
│              Docker Sidecar 集群                   │
│  ┌───────────┐ ┌───────────┐ ┌───────────┐      │
│  │  Agent 1  │ │  Agent 2  │ │  Agent N  │      │
│  │ :18043    │ │ :18239    │ │ :18020    │      │
│  │ ClaudeCode│ │ ClaudeCode│ │  Codex    │      │
│  └───────────┘ └───────────┘ └───────────┘      │
└──────────────────────────────────────────────────┘
```

## 快速开始

### 前置条件

- Go 1.22+
- Node.js 20+
- pnpm
- Docker（部署 Agent 容器时需要）
- AI Provider Token（通过 CCS 管理）

### 安装

```bash
git clone git@github.com:mudern/agentbucket.git
cd agentbucket

# 安装前端依赖
pnpm install

# 构建前端
pnpm build
```

### 启动后端

```bash
cd backend
go run ./cmd/server/

# 输出: AgentBucket backend listening on http://127.0.0.1:8080
```

默认绑定 `127.0.0.1:8080`（Docker sidecar 需要 `host.docker.internal:8080` 回调）。

环境变量：

| 变量 | 默认值 | 说明 |
|------|--------|------|
| `AGENTBUCKET_ADDR` | `127.0.0.1:8080` | 监听地址 |
| `AGENTBUCKET_BUILD_TIMEOUT` | `180s` | Docker build 超时 |
| `AGENTBUCKET_DATA_DIR` | `backend/.data` | 数据目录 |

### 启动前端开发服务器

```bash
pnpm dev
# 输出: http://127.0.0.1:5177
```

## AI Provider 配置

AI Token 从 `~/.config/ccs/providers/*.env` 自动导入。每个 `.env` 文件包含：

```bash
ANTHROPIC_AUTH_TOKEN="sk-xxx"
ANTHROPIC_BASE_URL="https://api.deepseek.com/anthropic"
ANTHROPIC_MODEL="deepseek-v4-pro[1m]"
```

支持通过 Anthropic 兼容协议接入的 Provider：

| Provider | Base URL |
|----------|----------|
| DeepSeek | `api.deepseek.com/anthropic` |
| GLM | `open.bigmodel.cn/api/anthropic` |
| Kimi | `api.moonshot.cn/anthropic` |
| MiniMax | `api.minimaxi.com/anthropic` |

## Agent 定义

```toml
# agents/my-agent/agent.toml
id          = "my-agent"
name        = "我的 Agent"
description = "Agent 功能描述"
model       = "deepseek-v4-pro[1m]"
runtime     = "claudecode"
runtime_version = "latest"
api_token   = "deepseek"
skills      = ["knowledge-base", "web-browser"]
mcps        = ["github-mcp", "filesystem-mcp"]
extra_install = ["apk add --no-cache github-cli"]
```

字段说明：

| 字段 | 说明 |
|------|------|
| `id` | Agent 唯一标识 |
| `name` | 显示名称 |
| `model` | AI 模型名 |
| `runtime` | 运行时：`claudecode` / `codex` |
| `api_token` | 关联的 AI Token 名称 |
| `skills` | 启用的 Skill 列表 |
| `mcps` | MCP 配置列表 |
| `extra_install` | 部署时额外安装命令（Dockerfile RUN） |

## Skills

标准 Skill 目录结构：

```
skills/
  knowledge-base/
    SKILL.md
  web-browser/
    SKILL.md
```

内置 Skills：

| Skill | 说明 |
|-------|------|
| `knowledge-base` | 知识库查询 |
| `document-parser` | 文档解析 |
| `git-reader` | Git 仓库读取 |
| `web-browser` | 网页浏览 |
| `agentbucket-comms` | Agent 间总线通信与 Token 解析 |

## API

完整 API 文档见 `agentbucket-api-skill/SKILL.md`。

### Agent 管理

```bash
# 列出 Agent
GET  /api/agents

# 扫描 Agent 定义
POST /api/agent-definitions/scan
```

### 部署

```bash
# 查看部署选项
GET  /api/deploy-options

# 部署 Agent
POST /api/deployments
Body: { repositoryId, agentId, apiTokenId, runtime, skills, mcps, ... }

# 查看部署
GET  /api/deployments/{id}
GET  /api/deployments/{id}/status

# 启停容器
POST /api/deployments/{id}/start
POST /api/deployments/{id}/stop
```

### 对话

```bash
# 会话管理
GET  /api/agents/{id}/sessions
POST /api/agents/{id}/sessions

# 消息
GET  /api/agents/{id}/messages?sessionId=xxx
POST /api/agents/{id}/messages
```

### Agent 总线

```bash
# 查看总线 Agent
GET  /api/bus/agents

# 注册到总线
POST /api/bus/agents/{id}/register

# 发送消息
POST /api/bus/agents/{id}/message

# 查看消息
GET  /api/bus/messages?toAgent=xxx
```

## 项目结构

```
AgentBucket/
├── backend/
│   ├── cmd/server/main.go       # Go 后端（单文件）
│   ├── examples/agent-repo/     # 示例 Agent 仓库
│   │   ├── agents/              # Agent 定义 (TOML)
│   │   ├── skills/              # Skill 定义
│   │   └── mcp/                 # MCP 配置
│   ├── AGENT_STANDARD.md        # Agent 标准文档
│   └── go.mod
├── src/                         # React 前端
│   ├── pages/                   # 页面组件
│   ├── components/              # 通用组件
│   └── api/                     # API 调用层
├── agentbucket-api-skill/       # API Skill 文档
├── public/                      # 静态资源
└── README.md
```

## 许可

MIT License
