# AgentBucket Administration Skill

Use this skill to manage AgentBucket entirely through the API — no frontend required. All examples use `curl`, suitable for automation scripts, CI/CD pipelines, or AI agents.

## Prerequisites

Set the API base URL. Adjust for your deployment:

```bash
# Local development
export AB_URL="http://127.0.0.1:8080"

# Docker deployment
export AB_URL="http://localhost:8080"

# Curl flag for local proxy environments
export AB_CURL="curl --noproxy '*' -sS"
```

## Quick Health Check

```bash
$AB_CURL $AB_URL/health
# {"ok":true}
```

---

## Repository Management

### List repositories

```bash
$AB_CURL $AB_URL/api/repositories
```

### Bind a local repository

```bash
$AB_CURL -X POST $AB_URL/api/repositories \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "example-agents",
    "provider": "Local",
    "localPath": "/home/user/agent-repos/my-agents",
    "branch": "main",
    "agentsPath": "agents"
  }'
```

### Bind a GitHub repository

```bash
$AB_CURL -X POST $AB_URL/api/repositories \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "my-agent-repo",
    "provider": "GitHub",
    "url": "https://github.com/user/my-agent-repo",
    "localPath": "/home/user/repos/my-agent-repo",
    "branch": "main",
    "agentsPath": "agents",
    "enabled": true
  }'
```

### Sync a GitHub repository

```bash
$AB_CURL -X POST $AB_URL/api/repositories/example-repo/sync
```

### Update repository settings

```bash
$AB_CURL -X PATCH $AB_URL/api/repositories/example-repo \
  -H 'Content-Type: application/json' \
  -d '{"enabled": true, "branch": "develop"}'
```

### Delete a repository

```bash
$AB_CURL -X DELETE $AB_URL/api/repositories/example-repo
```

---

## Agent Scanning

### List scanned agents

```bash
$AB_CURL $AB_URL/api/agents
```

### Re-scan agents from repositories

```bash
$AB_CURL -X POST $AB_URL/api/agent-definitions/scan
```

---

## AI Token Management

### List AI tokens

```bash
$AB_CURL $AB_URL/api/ai-tokens
```

### Add an AI token

```bash
$AB_CURL -X POST $AB_URL/api/ai-tokens \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "my-token",
    "provider": "ANTHROPIC",
    "apiKey": "sk-ant-...",
    "baseUrl": "https://api.anthropic.com",
    "model": "claude-sonnet-4-20250514",
    "scope": "general"
  }'
```

### Update an AI token

```bash
$AB_CURL -X PATCH $AB_URL/api/ai-tokens/1 \
  -H 'Content-Type: application/json' \
  -d '{"status": "禁用"}'
```

### Delete an AI token

```bash
$AB_CURL -X DELETE $AB_URL/api/ai-tokens/1
```

---

## Auth Token Management

Auth tokens are secrets that deployed agents can retrieve at runtime via the sidecar `/tokens/get` endpoint.

### List auth tokens

```bash
$AB_CURL $AB_URL/api/auth-tokens
```

### Add an auth token

```bash
$AB_CURL -X POST $AB_URL/api/auth-tokens \
  -H 'Content-Type: application/json' \
  -d '{
    "name": "GitHub Token",
    "accessTarget": "api.github.com",
    "functionName": "github_token.py",
    "status": "启用",
    "description": "GitHub API access token for release-writer agent"
  }'
```

### Update an auth token

```bash
$AB_CURL -X PATCH $AB_URL/api/auth-tokens/101 \
  -H 'Content-Type: application/json' \
  -d '{"status": "禁用"}'
```

### Delete an auth token

```bash
$AB_CURL -X DELETE $AB_URL/api/auth-tokens/104
```

---

## Deployment

### View deploy options

Shows available repositories, commits, agents, runtimes, models, AI tokens, auth tokens:

```bash
$AB_CURL $AB_URL/api/deploy-options
```

### Deploy an agent

```bash
$AB_CURL -X POST $AB_URL/api/deployments \
  -H 'Content-Type: application/json' \
  -d '{
    "repositoryId": "example-repo",
    "agentId": "legal-summarizer",
    "runtime": "claudecode",
    "runtimeVersion": "latest",
    "model": "deepseek-v4-pro",
    "apiTokenId": "1",
    "skills": ["knowledge-base", "document-parser"],
    "mcps": ["filesystem-mcp"],
    "authTokens": [101, 104]
  }'
```

### List all deployments

```bash
$AB_CURL $AB_URL/api/deployments
```

### Get a single deployment

```bash
$AB_CURL $AB_URL/api/deployments/dep-legal-summarizer-1234567890
```

### Get deployment status

```bash
$AB_CURL $AB_URL/api/deployments/dep-legal-summarizer-1234567890/status
```

### Stop a deployment

```bash
$AB_CURL -X POST $AB_URL/api/deployments/dep-legal-summarizer-1234567890/stop
```

### Start a stopped deployment

```bash
$AB_CURL -X POST $AB_URL/api/deployments/dep-legal-summarizer-1234567890/start
```

### Delete a deployment

```bash
$AB_CURL -X DELETE $AB_URL/api/deployments/dep-legal-summarizer-1234567890
```

---

## Agent Chat

### List sessions for an agent

```bash
$AB_CURL $AB_URL/api/agents/legal-summarizer/sessions
```

### Create a new session

```bash
$AB_CURL -X POST $AB_URL/api/agents/legal-summarizer/sessions \
  -H 'Content-Type: application/json' \
  -d '{"title": "Contract Review"}'
```

### Get messages in a session

Replace `SESSION_ID` with the actual session UUID:

```bash
$AB_CURL "$AB_URL/api/agents/legal-summarizer/messages?sessionId=SESSION_ID"
```

### Send a message (non-streaming)

```bash
$AB_CURL -X POST $AB_URL/api/agents/legal-summarizer/messages \
  -H 'Content-Type: application/json' \
  -d '{
    "sessionId": "SESSION_ID",
    "content": "Please review this contract clause...",
    "stream": false
  }'
```

### Send a message (SSE streaming)

```bash
$AB_CURL -X POST $AB_URL/api/agents/legal-summarizer/messages \
  -H 'Content-Type: application/json' \
  -d '{
    "sessionId": "SESSION_ID",
    "content": "Explain the key points of GDPR compliance",
    "stream": true
  }'
```

### Delete a session

```bash
$AB_CURL -X DELETE $AB_URL/api/agents/legal-summarizer/sessions/SESSION_ID
```

---

## Agent Bus

The Agent Bus allows deployed agents to discover and communicate with each other.

### List registered agents

```bash
$AB_CURL $AB_URL/api/bus/agents
```

### Register an agent on the bus

```bash
$AB_CURL -X POST $AB_URL/api/bus/agents/my-agent/register \
  -H 'Content-Type: application/json' \
  -d '{"name": "My Agent", "status": "online", "endpoint": "http://localhost:18000"}'
```

### Send a message to another agent

```bash
$AB_CURL -X POST $AB_URL/api/bus/agents/my-agent/message \
  -H 'Content-Type: application/json' \
  -d '{
    "toAgent": "legal-summarizer",
    "content": "Please analyze these documents and send results back"
  }'
```

### Query bus messages

```bash
# All messages
$AB_CURL $AB_URL/api/bus/messages

# Messages for a specific agent
$AB_CURL "$AB_URL/api/bus/messages?toAgent=legal-summarizer"
```

---

## Sidecar Direct Access

When a deployment is running, you can interact with its sidecar directly:

```bash
# Check sidecar health
$AB_CURL http://127.0.0.1:18043/health

# Get sidecar status
$AB_CURL http://127.0.0.1:18043/status

# Start the agent runtime
$AB_CURL -X POST http://127.0.0.1:18043/agent/start

# Stop the agent runtime
$AB_CURL -X POST http://127.0.0.1:18043/agent/stop

# Chat directly with the sidecar
$AB_CURL -X POST http://127.0.0.1:18043/agent/chat \
  -H 'Content-Type: application/json' \
  -d '{"message": "What is your purpose?"}'

# Resolve an auth token
$AB_CURL -X POST http://127.0.0.1:18043/tokens/get \
  -H 'Content-Type: application/json' \
  -d '{"tokenId": 101, "param": "my-request"}'
```

---

## Common Workflows

### Full deployment lifecycle

```bash
# 1. Bind a repository
$AB_CURL -X POST $AB_URL/api/repositories \
  -H 'Content-Type: application/json' \
  -d '{"name":"my-repo","provider":"Local","localPath":"/path/to/repo","branch":"main","agentsPath":"agents"}'

# 2. Scan for agents
$AB_CURL -X POST $AB_URL/api/agent-definitions/scan

# 3. Check what was found and get the agent ID
$AB_CURL $AB_URL/api/agents

# 4. Deploy
$AB_CURL -X POST $AB_URL/api/deployments \
  -H 'Content-Type: application/json' \
  -d '{"repositoryId":"my-repo","agentId":"my-agent","runtime":"claudecode"}'

# 5. Wait ~30s, then check status
$AB_CURL $AB_URL/api/deployments/dep-my-agent-TIMESTAMP/status

# 6. Start chatting
$AB_CURL -X POST $AB_URL/api/agents/my-agent/sessions \
  -H 'Content-Type: application/json' \
  -d '{"title":"First chat"}'

$AB_CURL -X POST $AB_URL/api/agents/my-agent/messages \
  -H 'Content-Type: application/json' \
  -d '{"sessionId":"SESSION_ID","content":"Hello!","stream":false}'

# 7. Clean up
$AB_CURL -X POST $AB_URL/api/deployments/dep-my-agent-TIMESTAMP/stop
$AB_CURL -X DELETE $AB_URL/api/deployments/dep-my-agent-TIMESTAMP
```

### Multi-agent collaboration via bus

```bash
# Agent 1: legal-summarizer asks Agent 2: code-reviewer
$AB_CURL -X POST $AB_URL/api/bus/agents/legal-summarizer/message \
  -H 'Content-Type: application/json' \
  -d '{"toAgent":"code-reviewer","content":"I found a privacy clause issue. Can you cross-check?"}'

# Agent 2: code-reviewer checks its messages
$AB_CURL "$AB_URL/api/bus/messages?toAgent=code-reviewer"

# Agent 2 responds
$AB_CURL -X POST $AB_URL/api/bus/agents/code-reviewer/message \
  -H 'Content-Type: application/json' \
  -d '{"toAgent":"legal-summarizer","content":"Confirmed. The clause violates GDPR Article 5."}'
```

## Notes

- Deployments may take 30-120 seconds depending on Docker build speed
- SSE streaming only works for direct AI API calls currently (sidecar streaming planned)
- Session limit is 20 per agent — use `DELETE` to remove old sessions
- Bus messages in memory are capped at 200; SQLite audit log may grow unbounded
- All API keys/secrets are stored server-side; the API never returns `apiKey` values in JSON responses
