<p align="center">
  <img src="public/agentbucket-logo-mark-transparent.png" alt="AgentBucket" width="96" />
  <br/>
  <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="360" />
</p>

<p align="center">Pannello di controllo AI per agenti definiti da repository, deployment Docker, orchestrazione sidecar e operazioni API-first.</p>

<p align="center">
  <a href="README.md">English</a> |
  <a href="README.zh.md">中文</a> |
  <a href="README.fr.md">Francais</a> |
  <a href="README.ja.md">日本語</a> |
  <a href="README.de.md">Deutsch</a> |
  <a href="README.ko.md">한국어</a> |
  <a href="README.es.md">Espanol</a> |
  <a href="README.ar.md">العربية</a> |
  <a href="README.pt.md">Portugues</a> |
  <a href="README.it.md">Italiano</a>
</p>

AgentBucket scansiona le definizioni degli agenti dai repository Git, impacchetta gli agenti selezionati con skill standard e configurazioni MCP in immagini Docker, esegue un sidecar in ogni contenitore ed espone un'interfaccia web e API compatibili con curl per deployment, chat, risoluzione dei token e messaggistica tra agenti.

## Funzionalità

- `agents/<agent-id>/agent.toml` でエージェントを定義
- `skills/<skill-id>/SKILL.md` から標準スキルを検証・パッケージ
- `mcp/*.json` から MCP 設定をパッケージ
- デプロイごとに Docker イメージをビルドし Go Sidecar を注入
- `claudecode`, `codex`, `opencode` ランタイムをサポート
- SQLite にユーザー、セッション、メッセージ、デプロイ、リポジトリ、状態を保存
- Sidecar または Anthropic 互換 API を通じた SSE ストリーミングチャット
- エージェント検出とメッセージパッシングのためのバス
- エージェントレベルの認可による Sidecar 経由のトークン解決

## Avvio Rapido

### Requisiti

- Go 1.22+, Node.js 20+, pnpm 11+, Docker (opzionale)

### Avviare il Backend

```bash
cd backend
AGENTBUCKET_ADDR=0.0.0.0:8080 go run ./cmd/server
```

### Avviare il Frontend

```bash
VITE_API_BASE=http://127.0.0.1:8080 pnpm dev --host 0.0.0.0 --port 5173
```

## Docker Compose

```bash
docker pull ghcr.io/mudern/agentbucket:latest
docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/mudern/agentbucket:latest
```

## Manifesto dell'Agente

```toml
id = "my-agent"
name = "My Agent"
description = "What this agent does"
model = "deepseek-v4-pro[1m]"
runtime = "claudecode"
runtime_version = "latest"
api_token = "deepseek"
skills = ["git-reader"]
mcps = ["filesystem-mcp"]
```

## Panoramica API

```text
GET  /health
GET  /api/current-user
GET  /api/agents
POST /api/deployments
GET  /api/deployments
GET  /api/agents/<built-in function id>/sessions
POST /api/agents/<built-in function id>/messages
GET  /api/ai-tokens
GET  /api/auth-tokens
POST /api/tokens/resolve
```

## Struttura del Progetto

```text
backend/cmd/server/    Go backend
backend/cmd/sidecar/   Sidecar source
src/                   React frontend
```

## Licenza MIT

Vedi [README.md](README.md) per la documentazione completa in inglese.
