<p align="center">
  <img src="public/agentbucket-logo-mark-transparent.png" alt="AgentBucket" width="96" />
  <br/>
  <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="360" />
</p>

<p align="center">저장소 정의 에이전트, Docker 배포, Sidecar 오케스트레이션, API-first 운영을 위한 AI 에이전트 제어 평면.</p>

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

AgentBucket은 Git 저장소에서 에이전트 정의를 스캔하고, 표준 스킬 및 MCP 구성과 함께 Docker 이미지로 패키징하며, 각 컨테이너에서 Sidecar를 실행하고, 배포, 채팅, 토큰 해결 및 에이전트 간 메시징을 위한 Web UI와 curl 친화적 API를 제공합니다.

## 기능

- `agents/<agent-id>/agent.toml` でエージェントを定義
- `skills/<skill-id>/SKILL.md` から標準スキルを検証・パッケージ
- `mcp/*.json` から MCP 設定をパッケージ
- デプロイごとに Docker イメージをビルドし Go Sidecar を注入
- `claudecode`, `codex`, `opencode` ランタイムをサポート
- SQLite にユーザー、セッション、メッセージ、デプロイ、リポジトリ、状態を保存
- Sidecar または Anthropic 互換 API を通じた SSE ストリーミングチャット
- エージェント検出とメッセージパッシングのためのバス
- エージェントレベルの認可による Sidecar 経由のトークン解決

## 빠른 시작

### 요구 사항

- Go 1.22+, Node.js 20+, pnpm 11+, Docker (선택)

### 백엔드 시작

```bash
cd backend
AGENTBUCKET_ADDR=0.0.0.0:8080 go run ./cmd/server
```

### 프론트엔드 시작

```bash
VITE_API_BASE=http://127.0.0.1:8080 pnpm dev --host 0.0.0.0 --port 5173
```

## Docker Compose

```bash
docker pull ghcr.io/mudern/agentbucket:latest
docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/mudern/agentbucket:latest
```

## 에이전트 매니페스트

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

## API 개요

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

## 프로젝트 구조

```text
backend/cmd/server/    Go backend
backend/cmd/sidecar/   Sidecar source
src/                   React frontend
```

## MIT 라이선스

전체 문서는 [README.md](README.md)를 참조하세요.
