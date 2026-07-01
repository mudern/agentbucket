<p align="center">
  <picture>
    <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="300" />
  </picture>
</p>

<p align="center">
  <strong>AI Agent Control Plane</strong><br/>
  Define, deploy, and orchestrate your AI agent fleet — with a single binary.
</p>

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go&logoColor=white" alt="Go" />
  <img src="https://img.shields.io/badge/React-18-61DAFB?style=flat&logo=react&logoColor=white" alt="React" />
  <img src="https://img.shields.io/badge/Tailwind-3-06B6D4?style=flat&logo=tailwindcss&logoColor=white" alt="Tailwind" />
  <img src="https://img.shields.io/badge/Vite-5-646CFF?style=flat&logo=vite&logoColor=white" alt="Vite" />
  <img src="https://img.shields.io/badge/SQLite-3-003B57?style=flat&logo=sqlite&logoColor=white" alt="SQLite" />
  <img src="https://img.shields.io/badge/Docker-✓-2496ED?style=flat&logo=docker&logoColor=white" alt="Docker" />
  <img src="https://img.shields.io/badge/i18n-EN%2FZH-blue?style=flat" alt="i18n" />
</p>

<p align="center">
  <a href="https://github.com/mudern/agentbucket/blob/main/README.zh.md">中文文档</a>
</p>

---

AgentBucket is a lightweight AI Agent control plane. Define agents via TOML manifests, deploy them as Docker containers with automatic sidecar orchestration, and manage everything through a polished web UI or REST API.

## Features

- **Agent Definition** — Declare agents in `agent.toml` with model, runtime, skills, and MCP configs
- **One-Click Deploy** — Automatic Docker build + container run with sidecar injection, port allocation, and health monitoring
- **Multi-Provider** — DeepSeek, GLM, Kimi, MiniMax via Anthropic-compatible API
- **Multi-Runtime** — Claude Code and Codex, both local and container modes
- **Real-Time SSE Chat** — Streaming responses with Markdown rendering, syntax highlighting, and interactive option buttons
- **Agent Bus** — Peer-to-peer agent discovery, messaging, and collaboration (200-message ring buffer + SQLite audit log)
- **Session Management** — Per-agent chat sessions with history, auto-persistence, and delete support
- **Token Resolution** — Auth tokens resolved through sidecars with agent-level authorization
- **Frontend UI** — Polished dashboard with searchable tables, capability pickers, deployment progress monitoring
- **i18n Ready** — English and Chinese UI, bilingual documentation
- **Docker-Native** — DooD (Docker-out-of-Docker) deployment with `docker-compose`, no DinD required
- **API-First** — Every feature accessible via curl; suitable for CI/CD and agent-to-agent communication

## Architecture

```
┌──────────────────────────────────────────────────┐
│  AgentBucket UI  (React + Vite + Tailwind)        │
├──────────────────────────────────────────────────┤
│  AgentBucket Backend  (Go 1.22 + SQLite)          │
│  ┌────────────────────────────────────────────┐   │
│  │  Agent Bus  (peer-to-peer agent messaging)  │   │
│  └────────────────────────────────────────────┘   │
├──────────────────────────────────────────────────┤
│  Docker Sidecar Cluster  (auto-orchestrated)      │
│  ┌──────────┐ ┌──────────┐ ┌──────────┐         │
│  │ Agent 1  │ │ Agent 2  │ │ Agent N  │         │
│  │ :18043   │ │ :18239   │ │ :18020   │         │
│  │ClaudeCode│ │ClaudeCode│ │  Codex   │         │
│  └──────────┘ └──────────┘ └──────────┘         │
└──────────────────────────────────────────────────┘
```

## Quick Start

### Prerequisites

- Go 1.22+
- Node.js 20+ / pnpm (for frontend development)
- Docker (for deploying agent containers)
- AI provider tokens (auto-imported from `~/.config/ccs/providers/*.env`)

### Development

```bash
git clone git@github.com:mudern/agentbucket.git
cd agentbucket

# Install frontend dependencies
pnpm install

# Start backend
cd backend
go run ./cmd/server/
# => AgentBucket backend listening on http://127.0.0.1:8080

# Start frontend (in another terminal)
pnpm dev
# => http://127.0.0.1:5177
```

### Docker Deployment

```bash
docker-compose up -d
# => http://localhost:8080
```

The backend mounts `/var/run/docker.sock` to manage sidecar containers on the host Docker daemon — **not Docker-in-Docker**.

### Environment Variables

| Variable | Default | Description |
|---|---|---|
| `AGENTBUCKET_ADDR` | `127.0.0.1:8080` | Listen address |
| `AGENTBUCKET_DATA_DIR` | `backend/.data` | SQLite and artifact storage |
| `AGENTBUCKET_BUILD_TIMEOUT` | `300s` | Docker build timeout |
| `AGENTBUCKET_SIDECAR_HOST` | `127.0.0.1` | Sidecar reachable host (`host.docker.internal` in Docker) |
| `AGENTBUCKET_PROVIDERS_DIR` | `~/.config/ccs/providers` | CCS provider env files |
| `AGENTBUCKET_ADMIN_TOKEN` | auto-generated | Master API token for auth |

## Agent Definition

```toml
# agents/my-agent/agent.toml
id              = "my-agent"
name            = "My Agent"
description     = "Agent description"
model           = "deepseek-v4-pro[1m]"
runtime         = "claudecode"
runtime_version = "latest"
api_token       = "deepseek"
skills          = ["knowledge-base", "web-browser"]
mcps            = ["github-mcp", "filesystem-mcp"]
extra_install   = ["apk add --no-cache github-cli"]
```

| Field | Description |
|---|---|
| `id` | Unique agent identifier |
| `name` | Display name |
| `model` | AI model name |
| `runtime` | `claudecode` or `codex` |
| `api_token` | Linked AI token name |
| `skills` | Enabled skill directories |
| `mcps` | MCP server configs |
| `extra_install` | Additional Dockerfile RUN commands |

## API Overview

Full API documentation available in `.skills/agentbucket-admin/SKILL.md`.

### Agents
```bash
GET    /api/agents
POST   /api/agent-definitions/scan
```

### Deployments
```bash
GET    /api/deploy-options
POST   /api/deployments
GET    /api/deployments/{id}
POST   /api/deployments/{id}/start
POST   /api/deployments/{id}/stop
```

### Chat & Sessions
```bash
GET    /api/agents/{id}/sessions
POST   /api/agents/{id}/sessions
DELETE /api/agents/{id}/sessions/{sessionId}
GET    /api/agents/{id}/messages?sessionId=xxx
POST   /api/agents/{id}/messages       # stream: true for SSE
```

### Agent Bus
```bash
GET    /api/bus/agents
POST   /api/bus/agents/{id}/register
POST   /api/bus/agents/{id}/message
GET    /api/bus/messages?toAgent=xxx
```

### Tokens & Repos
```bash
GET    /api/ai-tokens          POST   /api/ai-tokens
GET    /api/auth-tokens        POST   /api/auth-tokens
GET    /api/repositories       POST   /api/repositories
PATCH  /api/repositories/{id}  DELETE /api/repositories/{id}
```

## Project Structure

```
AgentBucket/
├── backend/
│   ├── cmd/server/          # Go backend (HTTP + SQLite + Docker orchestration)
│   │   ├── main.go          # Entrypoint
│   │   ├── server.go        # Routes, recovery, health checker
│   │   ├── types.go         # DTO/domain structs
│   │   ├── store.go         # SQLite persistence
│   │   ├── handlers.go      # HTTP handlers
│   │   ├── deploy.go        # Docker build/run pipeline
│   │   ├── agent_scan.go    # Repository scanning + agent.toml parser
│   │   ├── chat.go          # Chat sessions, AI API calls, SSE streaming
│   │   ├── bus.go           # Agent Bus registry/messaging
│   │   └── ...
│   ├── cmd/sidecar/         # Sidecar (compiled into each deploy image)
│   ├── examples/agent-repo/ # Example agent definitions
│   ├── tokens/              # Token resolution scripts
│   └── Dockerfile           # Production Docker image
├── src/                     # React frontend
│   ├── pages/               # Page components (Agents, Chat, Deploy, etc.)
│   ├── components/          # Shared UI components (Layout, Sidebar, etc.)
│   ├── api/                 # API client layer
│   └── i18n/                # Internationalization (EN/ZH)
├── .skills/                 # Claude Code skills for development
├── docker-compose.yml       # Orchestration config
└── README.md
```

## License

MIT License
