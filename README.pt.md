<p align="center"><img src="public/agentbucket-logo-mark-transparent.png" width="96" /><br/><img src="public/agentbucket-logo-mark.svg" width="360" /></p>
<p align="center">Plano de controle de agentes AI para agentes definidos por repositório, implantações Docker, orquestração sidecar e operações API-first.</p>
<p align="center"><a href="README.md">English</a> | <a href="README.zh.md">中文</a> | <a href="README.fr.md">Français</a> | <a href="README.ja.md">日本語</a> | <a href="README.de.md">Deutsch</a> | <a href="README.ko.md">한국어</a> | <a href="README.es.md">Español</a> | <a href="README.ar.md">العربية</a> | <a href="README.pt.md">Português</a> | <a href="README.it.md">Italiano</a></p>
## Início Rápido
```bash
cd backend && go run ./cmd/server
pnpm dev --host 0.0.0.0 --port 5173
```
## Docker
```bash
docker pull ghcr.io/mudern/agentbucket:latest
docker run -p 8080:8080 -v /var/run/docker.sock:/var/run/docker.sock ghcr.io/mudern/agentbucket:latest
```
Veja [README.md](README.md) para documentação completa.
