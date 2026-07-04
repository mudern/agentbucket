<p align="center">
  <img src="public/agentbucket-logo-mark-transparent.png" alt="Buckie" width="96" />
  <br/>
  <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="360" />
</p>

<p align="center">
  リポジトリ定義のエージェント、Docker デプロイ、Sidecar オーケストレーション、API ファースト運用のための AI エージェントコントロールプレーン。
</p>

<p align="center">
  <a href="README.md">English</a> | <a href="README.zh.md">中文</a> | <a href="README.fr.md">Français</a> | <a href="README.ja.md">日本語</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.ar.md">العربية</a> | <a href="README.pt.md">Português</a> | <a href="README.it.md">Italiano</a>
</p>

## クイックスタート

```bash
cd backend && go run ./cmd/server
pnpm dev --host 0.0.0.0 --port 5173
```

## Docker

```bash
docker pull ghcr.io/mudern/agentbucket:latest
docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/mudern/agentbucket:latest
```

詳細は [README.md](README.md) を参照。
