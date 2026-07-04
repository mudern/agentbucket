<p align="center">
  <img src="public/agentbucket-logo-mark-transparent.png" alt="Buckie, la mascotte AgentBucket" width="96" />
  <br/>
  <img src="public/agentbucket-logo-mark.svg" alt="AgentBucket" width="360" />
</p>

<p align="center">
  Plan de contrôle AI pour agents définis par dépôt, déploiements Docker, orchestration sidecar et opérations API-first.
</p>

<p align="center">
  <a href="README.md">English</a> |
  <a href="README.zh.md">中文</a> |
  <a href="README.fr.md">Français</a> |
  <a href="README.ja.md">日本語</a> |
  <a href="README.de.md">Deutsch</a> |
  <a href="README.ko.md">한국어</a> |
  <a href="README.es.md">Español</a> |
  <a href="README.ar.md">العربية</a> |
  <a href="README.pt.md">Português</a> |
  <a href="README.it.md">Italiano</a>
</p>

AgentBucket scanne les définitions d'agents depuis des dépôts Git, les package avec des skills standard et des configurations MCP dans des images Docker, exécute un sidecar dans chaque conteneur, et expose une interface web et des API curl-friendly pour le déploiement, le chat, la résolution de tokens et la messagerie agent-à-agent.

## Démarrage Rapide

```bash
# Backend
cd backend && go run ./cmd/server

# Frontend
pnpm dev --host 0.0.0.0 --port 5173
```

## Docker

```bash
docker pull ghcr.io/mudern/agentbucket:latest
docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/mudern/agentbucket:latest
```

Voir [README.md](README.md) pour la documentation complète.
