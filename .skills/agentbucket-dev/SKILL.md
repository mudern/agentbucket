# AgentBucket Development Skill

This skill helps developers understand, build, test, and contribute to the AgentBucket codebase.

## Project Structure

```
AgentBucket/
├── backend/                      # Go backend (API server + sidecar)
│   ├── cmd/server/               # Main server code
│   │   ├── main.go               # Entrypoint (~63 lines)
│   │   ├── server.go             # Route registration, container recovery, health checker
│   │   ├── types.go              # All DTO/domain structs
│   │   ├── store.go              # SQLite persistence layer
│   │   ├── handlers.go           # HTTP handlers (529 lines, needs splitting)
│   │   ├── deploy.go             # Docker build/run pipeline
│   │   ├── agent_scan.go         # Repository agent.toml scanning
│   │   ├── chat.go               # Chat/session CRUD, AI API calls, SSE streaming
│   │   ├── bus.go                # In-memory Agent Bus (registration, messaging)
│   │   ├── files.go              # Build-context file copy helpers
│   │   ├── utils.go, hash.go, timeutil.go  # Shared helpers
│   ├── cmd/sidecar/              # Sidecar (compiled into each Docker deployment)
│   │   ├── main.go               # Runtime runner, HTTP endpoints
│   │   ├── main_test.go          # Sidecar tests
│   ├── examples/agent-repo/      # Example agent definitions
│   ├── tokens/                   # Token resolution scripts
│   ├── .data/                    # SQLite DB + deployment build contexts
│   ├── go.mod, go.sum
│   ├── Dockerfile                # Production Docker build
│   ├── AGENT_STANDARD.md         # Agent definition specification
├── src/                          # React frontend
│   ├── pages/                    # Page components
│   │   ├── AgentsPage.jsx        # Agent listing with search/filter
│   │   ├── AgentChatPage.jsx     # Full-featured chat with sessions
│   │   ├── DeployPage.jsx        # 5-step deploy wizard
│   │   ├── RepositoriesPage.jsx  # Repository CRUD
│   │   ├── AiTokensPage.jsx      # AI token management
│   │   ├── AuthTokensPage.jsx    # Auth token management
│   │   ├── UsersPage.jsx         # User listing
│   │   ├── ApprovalsPage.jsx     # Approval listing
│   │   ├── AuthPage.jsx          # Login/register forms
│   ├── api/index.js              # API client
│   ├── components/               # Shared components
│   ├── hooks/                    # Custom React hooks
│   ├── data.js                   # Navigation/sidebar config
│   ├── i18n/                     # Internationalization
├── .skills/                      # Claude Code skills
├── docker-compose.yml            # Full stack orchestration
├── .dockerignore
├── package.json
├── vite.config.js
├── tailwind.config.js
```

## Architecture

### Backend (`backend/cmd/server/`)

- **Language**: Go 1.22
- **Database**: SQLite via `github.com/mattn/go-sqlite3` v1.14.22
- **Pattern**: Single-binary HTTP server with in-process SQLite
- **State**: `app_state` JSON blob + relational tables (`chat_sessions`, `chat_messages`, `bus_messages`, `users`)
- **Key env vars**:
  - `AGENTBUCKET_ADDR` — listen address (default `127.0.0.1:8080`, use `0.0.0.0:8080` for Docker)
  - `AGENTBUCKET_DATA_DIR` — data directory (default `<cwd>/.data`, use `/data` in Docker)
  - `AGENTBUCKET_SIDECAR_HOST` — host for sidecar URLs (default `127.0.0.1`, use `host.docker.internal` in Docker)
  - `AGENTBUCKET_BUILD_TIMEOUT` — Docker build timeout (default `300s`)
  - `AGENTBUCKET_PROVIDERS_DIR` — CCS provider env files directory

### Sidecar (`backend/cmd/sidecar/`)

- Compiled at deployment time into each Docker image
- Runs inside each deployed agent container
- Exposes: `/health`, `/status`, `/agent/start`, `/agent/stop`, `/agent/chat`, `/bus/register`, `/tokens/get`
- Auto-registers on Agent Bus at startup (retries 10x, 2s interval)
- Uses `AGENTBUCKET_URL` env var (default `http://host.docker.internal:8080`)

### Frontend

- **Framework**: React 18 + Vite 5
- **Styling**: Tailwind CSS 3 + `@tailwindcss/typography`
- **Markdown**: `react-markdown` + `rehype-highlight` (syntax highlighting)
- **Routing**: `react-router-dom` v6
- **I18n**: React context with EN/ZH support

### Deployment Pipeline

1. User selects repo → commit → agent in DeployPage
2. Backend scans local repo for `agent.toml` manifests
3. `createDeployment()` generates Docker build context with:
   - Agent directory
   - Selected skills
   - MCP configs
   - `sidecar/main.go` (compiled at Docker build time)
   - `agentbucket.config.json`
   - `Dockerfile`
4. Docker builds image, runs container with port mapping
5. Sidecar starts, registers on Agent Bus
6. Backend health checker monitors container status

## Development Commands

### Backend

```bash
cd backend

# Run server
GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod \
  AGENTBUCKET_ADDR=0.0.0.0:8080 AGENTBUCKET_BUILD_TIMEOUT=300s \
  go run ./cmd/server

# Run tests
GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod \
  go test ./...

# Run specific test
GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod \
  go test -v -run TestFunctionName ./cmd/server/
```

### Frontend

```bash
# Install dependencies
pnpm install

# Dev server
VITE_API_BASE=http://127.0.0.1:8080 pnpm exec vite --host 127.0.0.1

# Production build
pnpm build
```

### Docker

```bash
# Build and start full stack
docker-compose up -d

# View logs
docker-compose logs -f

# Stop
docker-compose down
```

### Curl

```bash
# Always use --noproxy '*' due to local proxy settings
curl --noproxy '*' -sS http://127.0.0.1:8080/health

# Deploy an agent
curl --noproxy '*' -sS -X POST http://127.0.0.1:8080/api/deployments \
  -H 'Content-Type: application/json' \
  -d '{"repositoryId":"example-repo","agentId":"legal-summarizer","runtime":"claudecode"}'
```

## Key Design Decisions

1. **DooD not DinD**: When running in Docker, the backend mounts the host's `/var/run/docker.sock`. All `docker build`/`docker run` commands execute on the host Docker daemon. Sidecar containers are peers on the host, not nested.

2. **Sidecar compilation at deploy time**: The sidecar Go source is copied into the Docker build context and compiled inside the Dockerfile (`FROM golang:1.22-alpine AS sidecar-build`). This means sidecar code changes take effect on the next deployment.

3. **JSON blob + relational hybrid**: Core mutable state is stored as a JSON blob in `app_state` for atomic reads/writes. Chat sessions and bus messages use relational tables for querying and indexing.

4. **Agent Bus is in-memory with SQLite audit log**: Bus messages are stored in an in-memory ring buffer (200 message cap) for fast access, with SQLite persistence as an audit log.

5. **SSE streaming for direct AI API calls**: When `stream: true` is passed to the chat endpoint, the backend opens an SSE (Server-Sent Events) connection to the AI API and forwards chunks to the frontend in real-time.

6. **Runtime abstraction**: `RuntimeRunner` interface with `CodexRunner` and `ClaudeCodeRunner` implementations. Each runner knows how to construct the CLI command for its respective runtime.

## Coding Conventions

- **Go**: Follow standard Go idioms. All code lives in `package main` (single executable). Error handling uses `fmt.Errorf` with `%w` wrapping where appropriate.
- **Frontend**: Functional components with hooks. API calls through `src/api/index.js`. Shared state through component composition (no global store). CSS via Tailwind utility classes.
- **Naming**: Use Chinese for user-facing status strings (e.g., `"启用"`, `"禁用"`), English for field names and IDs. Error messages are currently Chinese — planned to externalize into i18n.
- **Commits**: Imperative mood summaries with `Co-Authored-By: Claude Opus 4.6` trailer.

## Adding a New API Endpoint

1. Add any new types to `types.go`
2. Add the handler function in the appropriate file
3. Register the route in `server.go` → `routes()` using Go 1.22's pattern syntax:
   - `mux.HandleFunc("GET /api/resource", handler)` — method-specific
   - `mux.HandleFunc("/api/resource", handler)` — all methods
   - `mux.HandleFunc("GET /api/resource/{id}", handler)` — path params via `r.PathValue("id")`
4. Add a corresponding frontend API function in `src/api/index.js`
5. Add a test in the appropriate `*_test.go` file

## Adding a New Frontend Page

1. Create the page component in `src/pages/`
2. Add the route in `src/App.jsx`
3. Add the nav item in `src/data.js`
4. Add any new API functions to `src/api/index.js`

## Testing

- Backend tests use Go's standard `testing` package
- Sidecar tests verify runtime runner selection and HTTP handler responses
- Deploy tests verify Dockerfile generation and build context file copying
- Agent scan tests verify `agent.toml` parsing with various valid/invalid inputs
- Frontend has no automated tests yet (manual testing via Playwright or browser)
