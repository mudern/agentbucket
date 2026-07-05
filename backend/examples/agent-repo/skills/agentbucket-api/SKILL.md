---
name: agentbucket-api
description: Use this skill whenever the user wants to manage AgentBucket by API or curl: deploy agents, add repositories, add AI tokens, add auth tokens, inspect deployments, send messages to agents, or query sidecar status without using the web UI.
---

# AgentBucket API

Use AgentBucket as an API-first control plane. Prefer curl-friendly JSON payloads and return concise command sequences the user can run.

## Setup

Default API base:

```bash
export AGENTBUCKET_API="${AGENTBUCKET_API:-http://127.0.0.1:8080}"
```

When local proxy variables interfere with localhost, use:

```bash
curl --noproxy '*'
```

## Health

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/health"
```

## Agents

List all agents (scanned from configured repositories):

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/agents"
```

Re-scan agent definitions from repositories:

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/agent-definitions/scan"
```

## Deploy Options

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/deploy-options"
```

Use this to discover repositories, commits, agents, runtimes, models, AI tokens, auth tokens, and MCP servers.

Current runtimes are `codex`, `claudecode`, `opencode`, `gemini`, and `reasonix`.

## Add Repository

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/repositories" \
  -H 'Content-Type: application/json' \
  -d '{
    "id": "local-agents",
    "provider": "Local",
    "url": "file:///abs/path/to/repo",
    "branch": "main",
    "agentsPath": "agents",
    "localPath": "/abs/path/to/repo",
    "status": "启用"
  }'
```

## AI Tokens

AI tokens are created through the AgentBucket API/UI and stored server-side in the database. Secrets are redacted on list responses.

List AI tokens (secrets are redacted):

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/ai-tokens"
```

Add AI token:

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/ai-tokens" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "deepseek",
    "provider": "DeepSeek",
    "scope": "team",
    "baseUrl": "https://api.deepseek.com/anthropic",
    "model": "deepseek-v4-pro",
    "secret": "REDACTED",
    "status": "启用"
  }'
```

## Auth Tokens

List auth tokens:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/auth-tokens"
```

Add auth token:

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/auth-tokens" \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "Test Public API",
    "description": "测试外部公开 API",
    "secret": "test_public_secret",
    "status": "启用"
  }'
```

## Resolve Token

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/tokens/resolve" \
  -H 'Content-Type: application/json' \
  -H 'X-Agent-ID: legal-summarizer' \
  -d '{"tokenId":101,"param":"smoke"}'
```

## Deploy Agent

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/deployments" \
  -H 'Content-Type: application/json' \
  -d '{
    "repositoryId": "agentbucket-example",
    "commitHash": "ea41cfe",
    "agentId": "legal-summarizer",
    "apiTokenId": 1,
    "model": "deepseek-v4-pro[1m]",
    "runtime": "claudecode",
    "runtimeVersion": "latest",
    "skills": ["knowledge-base", "document-parser"],
    "mcps": ["notion-mcp", "filesystem-mcp"],
    "authTokens": []
  }'
```

Response includes: `id`, `status`, `imageTag`, `containerName`, `sidecarUrl`, `hostPort`, `buildContext`, `createdAt`.

## Deployments

List all deployments:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/deployments"
```

Get single deployment:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/deployments/dep-legal-summarizer-1782826339"
```

Get deployment status:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/deployments/dep-legal-summarizer-1782826339/status"
```

Start a stopped deployment:

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/deployments/dep-legal-summarizer-1782826339/start"
```

Stop a running deployment:

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/deployments/dep-legal-summarizer-1782826339/stop"
```

## Sidecar

Use the `sidecarUrl` from a deployment response:

```bash
export SIDECAR_URL="http://127.0.0.1:18043"
```

Status:

```bash
curl --noproxy '*' -sS "$SIDECAR_URL/status"
```

Start runtime:

```bash
curl --noproxy '*' -sS -X POST "$SIDECAR_URL/agent/start"
```

Stop runtime:

```bash
curl --noproxy '*' -sS -X POST "$SIDECAR_URL/agent/stop"
```

Get authorized token through sidecar:

```bash
curl --noproxy '*' -sS -X POST "$SIDECAR_URL/tokens/get" \
  -H 'Content-Type: application/json' \
  -d '{"tokenId":101,"param":"smoke"}'
```

Chat through deployed sidecar:

```bash
curl --noproxy '*' -sS -X POST "$SIDECAR_URL/agent/chat" \
  -H 'Content-Type: application/json' \
  -d '{"message":"你好，介绍一下你能做什么"}'
```

## Chat With Agent (via backend API)

List sessions:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/agents/legal-summarizer/sessions"
```

Create session:

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/agents/legal-summarizer/sessions" \
  -H 'Content-Type: application/json' \
  -d '{"title":"部署问题排查"}'
```

Send message (backend routes to AI API or deployed sidecar):

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_API/api/agents/legal-summarizer/messages" \
  -H 'Content-Type: application/json' \
  -d '{
    "sessionId": "default",
    "content": "你好，列出当前可用 skill"
  }'
```

Read messages:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/agents/legal-summarizer/messages?sessionId=default"
```

## Users and Approvals

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/users"
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/approvals"
curl --noproxy '*' -sS "$AGENTBUCKET_API/api/current-user"
```

## Agent Standard

Agents use `agent.toml`:

```toml
id = "legal-summarizer"
name = "法务文档总结"
description = "法务文档总结与风险条款提取 Agent，使用 DeepSeek V4 模型。"
model = "deepseek-v4-pro[1m]"
runtime = "claudecode"
runtime_version = "latest"
api_token = "deepseek"
skills = ["knowledge-base", "document-parser"]
mcps = ["notion-mcp", "filesystem-mcp"]
```

Available runtimes: `codex`, `claudecode`, `opencode`, `gemini`, `reasonix`

AI token provider names are user-defined and come from the tokens stored in AgentBucket.

Skills must be standard skill directories: `skills/<skill-id>/SKILL.md`

MCP configs are JSON files in `mcp/<mcp-id>.json`
