<p align="center">
  <img src="public/agentbucket-logo-mark-transparent.png" alt="AgentBucket" width="96" />
  <br/>
  <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="360" />
</p>

<p align="center">リポジトリ定義エージェント、Docker デプロイ、Sidecar オーケストレーション、API ファースト運用のための AI エージェントコントロールプレーン。</p>

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

AgentBucket は Git リポジトリからエージェント定義をスキャンし、標準スキルと MCP 設定を含む Docker イメージにパッケージ化し、各コンテナで Sidecar を実行し、デプロイ、チャット、トークン解決、エージェント間メッセージングのための Web UI と curl 対応 API を提供します。

## 機能

- `agents/<agent-id>/agent.toml` でエージェントを定義
- `skills/<skill-id>/SKILL.md` から標準スキルを検証・パッケージ
- `mcp/*.json` から MCP 設定をパッケージ
- デプロイごとに Docker イメージをビルドし Go Sidecar を注入
- `claudecode`, `codex`, `opencode` ランタイムをサポート
- SQLite にユーザー、セッション、メッセージ、デプロイ、リポジトリ、状態を保存
- Sidecar または Anthropic 互換 API を通じた SSE ストリーミングチャット
- エージェント検出とメッセージパッシングのためのバス
- エージェントレベルの認可による Sidecar 経由のトークン解決

## クイックスタート

### 要件

- Go 1.22+, Node.js 20+, pnpm 11+, Docker（オプション）

### バックエンドの起動

```bash
cd backend
AGENTBUCKET_ADDR=0.0.0.0:8080 go run ./cmd/server
```

### フロントエンドの起動

```bash
VITE_API_BASE=http://127.0.0.1:8080 pnpm dev --host 0.0.0.0 --port 5173
```

## Docker Compose

```bash
docker pull ghcr.io/mudern/agentbucket:latest
docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/mudern/agentbucket:latest
```

## エージェントマニフェスト

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

## API 概要

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

## プロジェクト構成

```text
backend/cmd/server/    Go backend
backend/cmd/sidecar/   Sidecar source
src/                   React frontend
```

## MIT ライセンス

完全なドキュメントは [README.md](README.md) を参照してください。
