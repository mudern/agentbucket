# AgentBucket Handoff Notes

## Current Runtime State

- Backend is running from `backend/`:
  - Command used:
    ```bash
    GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod AGENTBUCKET_ADDR=0.0.0.0:8080 AGENTBUCKET_BUILD_TIMEOUT=300s go run ./cmd/server
    ```
  - API base: `http://127.0.0.1:8080`
  - It must bind `0.0.0.0:8080` so Docker sidecars can call `http://host.docker.internal:8080`.

- Frontend is running:
  - URL: `http://127.0.0.1:5177/`
  - Started with:
    ```bash
    VITE_API_BASE=http://127.0.0.1:8080 pnpm exec vite --host 127.0.0.1
    ```

- Running Docker sidecar container:
  - Name: `agentbucket-legal-summarizer`
  - Image: `agentbucket/legal-summarizer:ea41cfe`
  - Sidecar URL: `http://127.0.0.1:18043`

## Implemented

### Backend Refactor

- `backend/cmd/server/main.go` was partially split.
- New files:
  - `backend/cmd/server/server.go`
    - server entrypoint
    - app initialization
    - route registration
  - `backend/cmd/server/types.go`
    - DTO/domain structs
    - app/store/bus types
  - `backend/cmd/server/store.go`
    - SQLite store/schema/load/save logic
  - `backend/cmd/server/handlers.go`
    - HTTP handlers for core resources, deployments and bus API
  - `backend/cmd/server/deploy.go`
    - deployment creation, Docker build/run and Dockerfile generation
  - `backend/cmd/server/agent_scan.go`
    - repository scan, `agent.toml` parsing and MCP scan
  - `backend/cmd/server/bus.go`
    - in-memory Agent bus registry/messages
  - `backend/cmd/server/chat.go`
    - chat helper, runtime CLI calls, sidecar chat calls and AI API fallback
  - `backend/cmd/server/files.go`
    - build-context file copy and tarball helpers
  - `backend/cmd/server/utils.go`, `hash.go`, `timeutil.go`
    - shared small helpers
  - `backend/cmd/sidecar/main.go`
    - real sidecar command
    - compiled by `go test ./...`
- The old embedded `const sidecarSource = "...go source..."` string was removed.
- Server now copies sidecar source from:
  - `backend/cmd/sidecar/main.go`
- This fixes the previous problem where sidecar code could not be statically checked or compiled until Docker build time.
- Added tests for the sidecar and packaging path:
  - `backend/cmd/sidecar/main_test.go`
    - verifies runtime runner selection and `/status` handler response.
  - `backend/cmd/server/deploy_test.go`
    - verifies Dockerfile generation and that build context copies the real sidecar file.
  - `backend/cmd/server/agent_scan_test.go`
    - verifies `agent.toml` parsing.
  - `backend/cmd/server/files_test.go`
    - verifies standard skill directories must contain `SKILL.md`.
- Current line counts after second refactor:
  - `backend/cmd/server/main.go`: ~63 lines
  - `backend/cmd/server/server.go`: ~68 lines
  - `backend/cmd/server/types.go`: ~216 lines
  - `backend/cmd/server/store.go`: ~353 lines
  - `backend/cmd/server/handlers.go`: ~529 lines
  - `backend/cmd/server/deploy.go`: ~216 lines
  - `backend/cmd/server/agent_scan.go`: ~163 lines
  - `backend/cmd/server/chat.go`: ~331 lines
  - `backend/cmd/sidecar/main.go`: ~284 lines
- Validation after refactor:
  ```bash
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  ```
  passes for both `cmd/server` and `cmd/sidecar`.
- Latest verification also ran:
  ```bash
  pnpm build
  ```
  and it passes. Vite still warns that the built JS chunk is larger than 500 kB.
- Next refactor step:
  - Split `handlers.go` further by resource, because it is still ~529 lines.
  - Consider moving common runtime/API code from `chat.go` into an internal package once behavior stabilizes.

### Backend

- Added Go backend under `backend/`.
- Main file: `backend/cmd/server/main.go`.
- Added SQLite dependency:
  - `github.com/mattn/go-sqlite3 v1.14.22`
- SQLite DB path:
  - `backend/.data/agentbucket.db`
- Current SQLite usage:
  - `app_state` table stores JSON state snapshot.
  - `users` table stores fake users.
  - `chat_sessions` stores Agent conversation sessions.
  - `chat_messages` stores Agent conversation messages.
  - Provider env files from `$HOME/.config/ccs/providers/*.env` are imported as AI tokens. Secrets are stored server-side and are not returned by JSON APIs.

### Agent Standard

- Agent standard doc:
  - `backend/AGENT_STANDARD.md`
- Agent definitions now use TOML:
  - `agents/<agent-id>/agent.toml`
- Example repo:
  - `backend/examples/agent-repo`
- Example agents:
  - `legal-summarizer`
  - `release-writer`
- Example `agent.toml` fields:
  - `id`
  - `name`
  - `description`
  - `model`
  - `runtime`
  - `runtime_version`
  - `api_token`
  - `skills`
  - `mcps`

### Skill Standard

- Skills must be standard skill directories:
  - `skills/<skill-id>/SKILL.md`
- Example skills:
  - `knowledge-base`
  - `document-parser`
  - `git-reader`
  - `web-browser`
- Deploy packaging validates `SKILL.md` exists for every selected skill.

### Backend API

Implemented endpoints:

- `GET /health`
- `GET /api/current-user`
- `GET /api/agents`
- `GET /api/users`
- `GET /api/approvals`
- `GET /api/ai-tokens`
- `POST /api/ai-tokens`
- `GET /api/auth-tokens`
- `POST /api/auth-tokens`
- `GET /api/deploy-options`
- `GET /api/repositories`
- `POST /api/repositories`
- `GET /api/deployments`
- `POST /api/deployments`
- `POST /api/tokens/resolve`
- `GET /api/agents/{agentId}/sessions`
- `GET /api/agents/{agentId}/messages?sessionId=...`
- `POST /api/agents/{agentId}/messages`

### Deploy Flow

`POST /api/deployments`:

- Finds repo/commit/agent from scanned local repo.
- Validates runtime:
  - `codex`
  - `claudecode`
- Copies selected agent directory into Docker build context.
- Copies selected standard skill directories.
- Copies MCP configs.
- Writes:
  - `agentbucket.config.json`
  - `sidecar/main.go`
  - `Dockerfile`
  - `context.tar`
- Builds Docker image.
- Runs Docker container with sidecar port mapping.

### Runtime Abstraction

The generated sidecar now contains a small runtime abstraction:

- `RuntimeRunner`
- `CodexRunner`
- `ClaudeCodeRunner`

Dockerfile installs runtime CLI:

- `codex`: `npm install -g @openai/codex@<version>`
- `claudecode`: `npm install -g @anthropic-ai/claude-code@<version>`

Important: Dockerfile was changed to use `node:20-alpine` as the final image. The earlier `alpine + apk add nodejs npm` path timed out badly.

### Sidecar Server

Generated sidecar exposes:

- `GET /health`
- `GET /status`
- `POST /agent/start`
- `POST /agent/stop`
- `POST /bus/register`
- `POST /tokens/get`

Verified:

- `GET http://127.0.0.1:18043/status` returns:
  - `online: true`
  - `runtime: codex`
  - `runtimeVersion: latest`
  - `skills`
  - `mcps`
  - `authTokens`
- `POST /agent/start` starts a runtime process and makes `online: true`.

### Token/Auth Test Flow

Seed auth tokens include:

- `101` Test Public API, enabled.
- `102` Test Admin API, enabled.
- `103` Test Disabled API, disabled.

Verified:

```bash
curl --noproxy '*' -sS -X POST http://127.0.0.1:18043/tokens/get \
  -H 'Content-Type: application/json' \
  -d '{"tokenId":101,"param":"smoke"}'
```

Returns a test token.

Disabled token test:

```bash
curl --noproxy '*' -sS -X POST http://127.0.0.1:18043/tokens/get \
  -H 'Content-Type: application/json' \
  -d '{"tokenId":103,"param":"smoke"}'
```

Returns:

```json
{"error":"token is disabled"}
```

### Frontend

- `src/api/index.js` now calls real backend API at:
  - `VITE_API_BASE` or default `http://127.0.0.1:8080`
- `DeployPage.jsx` posts to `/api/deployments`.
- `DeployPage.jsx` supports runtime version.
- `AgentChatPage.jsx` now loads sessions/messages from the backend and can send messages from the textarea.
- `agentbucket-api-skill/SKILL.md` documents curl/API workflows for agents or users that do not want to open the UI.
- `pnpm build` passes.

## Verified Commands

Backend tests:

```bash
cd backend
GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
```

Passed.

Frontend build:

```bash
pnpm build
```

Passed.

Successful deploy result:

- Deployment ID: `dep-legal-summarizer-1782826339`
- Status: `running`
- Image: `agentbucket/legal-summarizer:ea41cfe`
- Container: `agentbucket-legal-summarizer`
- Sidecar URL: `http://127.0.0.1:18043`

## Current User Requirements Not Fully Done

### 1. AI-native API Design

User requested all APIs be designed more AI-native and easy to use via curl/agents.

Need to add/clean endpoints such as:

- `POST /api/agent-definitions/scan`
- `POST /api/deployments`
- `GET /api/deployments/:id`
- `POST /api/deployments/:id/start`
- `POST /api/deployments/:id/stop`
- `GET /api/deployments/:id/status`
- `POST /api/ai-tokens`
- `POST /api/auth-tokens`
- `POST /api/repositories`
- `POST /api/agents/:agentId/sessions`
- `GET /api/agents/:agentId/sessions`
- `POST /api/agents/:agentId/messages`
- `GET /api/agents/:agentId/messages?sessionId=...`

Keep JSON payloads simple and script-friendly.

### 2. Agent Chat Is Implemented As A Local Conversation Loop

Implemented:

- `GET /api/agents/{agentId}/sessions`
- `GET /api/agents/{agentId}/messages?sessionId=default`
- `POST /api/agents/{agentId}/messages`
- Frontend textarea sends messages and appends user/assistant responses.
- Sessions and messages are persisted in SQLite-backed state and mirrored into dedicated tables.

Current assistant response is still a structured AgentBucket response, not a full runtime-streamed Codex/Claude answer. It includes runtime, online state, skills and MCPs. Next step is to pipe user messages through the sidecar runtime and stream output back.

### 3. Session Management Is SQLite-native

Current schema:

```sql
CREATE TABLE IF NOT EXISTS chat_sessions (
  id TEXT PRIMARY KEY,
  agent_id TEXT NOT NULL,
  title TEXT NOT NULL,
  preview TEXT NOT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS chat_messages (
  id TEXT PRIMARY KEY,
  session_id TEXT NOT NULL,
  agent_id TEXT NOT NULL,
  role TEXT NOT NULL,
  content TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE INDEX IF NOT EXISTS idx_chat_sessions_agent_updated
  ON chat_sessions(agent_id, updated_at DESC);

CREATE INDEX IF NOT EXISTS idx_chat_messages_agent_session_created
  ON chat_messages(agent_id, session_id, created_at ASC);
```

### 4. Skill Package For AgentBucket APIs

User requested a companion skill so agents can use AgentBucket APIs without opening the UI.

Location:

- `agentbucket-api-skill/SKILL.md`

Note: `.codex` is read-only in this workspace, so the skill was created in the repo instead of installed into `.codex/skills`.

This skill should include:

- API base setup.
- Curl examples.
- Deploy new Agent.
- Add repository.
- Add AI token.
- Add auth token.
- Scan agents.
- Query deployment status.
- Start/stop deployment.
- Send chat message.
- Resolve token through sidecar.

Use real endpoint names after API cleanup.

### 5. Runtime Management Needs More Realism

Currently sidecar starts:

- `codex exec --model <model> "AgentBucket sidecar online"`
- `claude -p "AgentBucket sidecar online"`

This proves runtime CLI installation and process management, but not yet a full interactive agent message bridge.

Need to design:

- How chat messages are routed into runtime.
- How runtime stdout/stderr is captured.
- How streaming responses work.
- How bus registration is represented in backend.

## Important Environment Notes

- Go needs writable caches:

```bash
GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
```

- Without those env vars, Go may fail due read-only filesystem.

- Curl often needs:

```bash
curl --noproxy '*'
```

because local proxy settings try to route localhost through `127.0.0.1:7890`.

- Docker build can be slow due network. `node:20-alpine` fixed the worst timeout compared to installing node/npm on raw Alpine.

## Files Changed/Added

Backend:

- `backend/go.mod`
- `backend/go.sum`
- `backend/cmd/server/main.go`
- `backend/AGENT_STANDARD.md`
- `backend/examples/agent-repo/...`

Frontend:

- `src/api/index.js`
- `src/pages/DeployPage.jsx`

Docs:

- `README.md`

Handoff:

- `temp.md`

## 2026-06-30 Session Progress

### Cleaned Up Mock Data

- Removed mock AITokens from `seedState`. AITokens are now auto-imported from `~/.config/ccs/providers/` (deepseek, glm, kimi, minimax) with real API keys.
- Removed mock AuthTokens. Auth tokens are now empty by default - add via API or UI.
- Removed mock approvals.
- Removed unused `src/api/mockData.js`.

### Added Missing API Endpoints

- `POST /api/agent-definitions/scan` - Re-scan agent definitions from repositories.
- `GET /api/deployments/{id}` - Get a single deployment by ID.
- `GET /api/deployments/{id}/status` - Get deployment status summary.
- `POST /api/deployments/{id}/start` - Start a stopped deployment container.
- `POST /api/deployments/{id}/stop` - Stop a running deployment container.
- `POST /api/agents/{agentId}/sessions` - Create a new chat session with a title.

### Real AI Chat Implementation

- Replaced the hardcoded `buildAssistantMessage` with `callAIAPI` that makes real HTTP calls to Anthropic-compatible AI APIs.
- Message routing logic:
  - If agent has a running sidecar deployment, tries to call `{sidecarUrl}/agent/chat` first.
  - Falls back to calling the AI API directly using the agent's configured API token/base URL/model.
- Fixed DeepSeek response parsing (handles `thinking` + `text` content blocks).
- Fixed session ID uniqueness (removed hardcoded "default" session ID, now uses hash-based unique IDs per agent).

### Sidecar Updates

- Added `/agent/chat` endpoint to the embedded sidecar source.
- Sidecar chat endpoint takes `{"message":"..."}` and executes the runtime CLI (codex/claude) with the user message, returns stdout.

### Agent Definitions Updated

Updated existing agents and added new ones using real CCS providers:
- `legal-summarizer` - DeepSeek V4 Pro
- `release-writer` - GLM 4.5
- `code-reviewer` (new) - MiniMax M2
- `support-bot` (new) - Kimi Latest

All agents use `claudecode` runtime and reference real CCS token names.

### Frontend Updates

- Added `createAgentSession` API function.
- Added session creation UI in `AgentChatPage`: input field + "+" button in the sidebar.
- Sessions refresh automatically after creation and after sending messages.
- Added `whitespace-pre-wrap` to message content for proper formatting of AI responses.
- Removed the `WizardModal` import (no longer used in AgentChatPage).

### Skill Package Updated

- Updated `agentbucket-api-skill/SKILL.md` with all real endpoint names, including new deployment lifecycle endpoints, session creation, and sidecar chat.

### Verified

- Backend compiles and all endpoints return expected responses.
- Frontend builds successfully.
- Chat with DeepSeek agent returns real AI responses.
- Chat with GLM agent returns real AI responses.
- Session creation via POST works.
- Agent scan returns all 4 agents with correct TOML configs.
- Deployment status endpoint correctly returns 404 for nonexistent deployments.

## 2026-06-30 Session 2 Progress

### Frontend Redesign

- **Markdown rendering**: Installed `react-markdown` + `remark-gfm` + `@tailwindcss/typography`. AI responses now render with proper formatting, code blocks, lists, tables, etc.
- **Interactive options**: AI can ask questions via `[QUESTION: prompt | option1 | option2]` format. Frontend parses this and renders clickable buttons. Clicking an option sends it as the next message.
- **Polished UI**:
  - Message bubbles with colored avatars ("你" for user, "AI" gradient for assistant)
  - Timestamps on each message
  - Typing indicator with animated dots while waiting for AI response
  - Auto-scroll to latest message
  - Session sidebar shows skills/MCPs at the bottom
  - Empty state with icon and hint text
  - Disabled input when no session selected
  - Error display with styling
  - Focus ring on input area
- Created `tailwind.config.js` with typography plugin.

### Auth Tokens

- Seeded 5 auth tokens in `seedState`:
  - 101: Test Public API (enabled) - available to all deployed agents
  - 102: Test Admin API (enabled) - only for explicitly authorized agents
  - 103: Test Disabled API (disabled) - always returns 403
  - 104: GitHub Token (enabled)
  - 105: Internal DB (enabled)
- Token resolution through sidecar respects agent authorization (only agents in deployments with matching auth token IDs can access).

### Agent Bus

- Added in-memory shared bus (`AgentBus`) to the backend.
- Endpoints:
  - `GET /api/bus/agents` - List all registered agents
  - `POST /api/bus/agents/{agentId}/register` - Register an agent on the bus
  - `POST /api/bus/agents/{agentId}/message` - Send a message to another agent
  - `GET /api/bus/messages?toAgent=xxx` - Query bus messages (filterable by recipient)
- Bus retains last 200 messages.
- Agents can discover each other and communicate peer-to-peer.

### Session Message Limit

- Enforced 20-roundtrip limit per session (40 messages total = 20 user + 20 assistant).
- Atomic check inside `store.update` to prevent race conditions.
- Returns HTTP 403 with Chinese error message when limit reached.

### Interaction Protocol

- Backend appends system prompt to user messages telling AI it can use `[QUESTION: text | optionA | optionB]` format.
- Frontend `MessageBubble` component parses this format and renders interactive buttons.
- Clicking an option calls `handleSend(opt)` which sends it as a user message.

### Communication Skill

- Created `agentbucket-comms` skill at `backend/examples/agent-repo/skills/agentbucket-comms/SKILL.md`.
- Documents bus registration, message sending, token resolution, and agent discovery flows.

### Files Changed/Added

- `backend/cmd/server/main.go` - Bus, auth tokens, session limits, interaction protocol
- `src/pages/AgentChatPage.jsx` - Full redesign with markdown, options, polished UI
- `src/api/index.js` - Added `createAgentSession`
- `tailwind.config.js` - Created with typography plugin
- `backend/examples/agent-repo/skills/agentbucket-comms/SKILL.md` - New communication skill
- `package.json` - Added react-markdown, remark-gfm, @tailwindcss/typography

## 2026-06-30 Session 3 Progress

### Fixed Model Names

- Kimi: `kimi-latest` → `kimi-k2.6` (verified working)
- MiniMax: `minimax-m2` → `MiniMax-M2.5` (API balance issue, code correct)
- GLM: `glm-4.5` → `glm-5-turbo` (matches CCS config)

### Fixed Session Limit

- Changed from **20 rounds per session** to **20 sessions per agent** as user intended
- Limit enforced atomically in `store.update` to prevent race conditions
- Returns HTTP 403 "会话数已达上限（20 个）" when exceeded
- Applies to both explicit `POST /api/agents/{id}/sessions` and auto-created sessions

### Fixed Frontend Overflow

- Added `overflow-hidden`, `break-words`, `min-w-0 max-w-[75%]` to message containers
- Code blocks: `prose-pre:max-w-[calc(75vw-6rem)] prose-pre:overflow-x-auto`
- Images: `prose-img:max-w-full`

### Added Code Syntax Highlighting

- Installed `rehype-highlight` + `highlight.js`
- Imported `highlight.js/styles/github-dark.css` theme
- Integrated `rehypeHighlight` plugin into ReactMarkdown pipeline
- Code blocks now have proper syntax highlighting in dark theme

### Added Back Button

- Agent chat page header now has "← 返回" button linking to `/` (Agents page)
- Uses `Link` from react-router-dom

### Injected Bus/Skills/MCPs Context into AI

- `callAIAPI` now builds a rich system prompt including:
  - Agent name, ID, runtime
  - Skills list
  - MCP configurations
  - Bus communication instructions (discovery, register, send/receive messages)
  - QUESTION format availability (only when needed)
- AI now knows it can discover and communicate with other agents on the bus
- AI can use bus endpoints to send messages to other agents

### Removed Forced QUESTION Prompt

- System prompt now says "如果需要用户做出选择或确认" (if you need the user to choose)
- Question format is available but not forced on every response

## 2026-06-30 Session 4 Progress

### Frontend Optimistic Update

- User messages now appear instantly in chat (before API response)
- Temporary message with `temp-{timestamp}` ID shown immediately
- Typing indicator shown while waiting for AI response
- Temp message replaced with real messages from API response
- On error, temp message removed and error shown

### Sidecar Auto-Registration on Bus

- Sidecar `main()` now starts a goroutine that auto-registers the agent on the bus
- Retries up to 10 times with 2s intervals (handles cold start race)
- Calls `POST /api/bus/agents/{agentId}/register` with agent name, status, endpoint
- Uses `AGENTBUCKET_URL` env var (defaults to `http://host.docker.internal:8080`)
- When a Docker deployment starts, the agent automatically appears on the bus

### Real Bus Context in AI Prompt

- AI system prompt now includes the ACTUAL bus agent list (not instructions to call API)
- `callAIAPI` takes `app *App` parameter to access `app.bus.list()`
- Bus agents are injected into the context prompt as a formatted list
- AI excludes itself from the list
- Verified: AI correctly reports real agents (no hallucination)

## 2026-07-01 Chat Layout Polish

### Agent Chat Page Layout

- Reworked `src/pages/AgentChatPage.jsx` into one bounded white workspace with a single compact top bar.
- Added a collapsible conversation sidebar controlled from the top bar.
- Top bar now shows back navigation, session sidebar toggle, current Agent, current session, model, runtime, and status.
- "新建会话" in the top bar opens the session sidebar and focuses the new session input.
- Removed the second nested chat header and the demo-like empty-state icon so the message area has more room.
- Verified with `pnpm build`; build passes with the existing Vite chunk-size warning.

## 2026-07-01 Deploy Capability Picker

### Deploy Page Configuration UX

- Replaced the flat API Token / Skill / MCP / Auth Token checkbox lists in `src/pages/DeployPage.jsx`.
- The "配置能力" step now shows compact summary cards and opens a searchable picker modal per capability type.
- Picker supports selected-first ordering, search, single-select API Token, and multi-select bulk actions for Skill/MCP/Auth Token.
- Added touched-state handling for Skill/MCP so users can intentionally clear all selections instead of falling back to Agent defaults.
- Verified with `pnpm build`; build passes with the existing Vite chunk-size warning.

## 2026-07-01 Backend SSE Robustness

### Chat Stream Parsing

- Continued from the handoff's runtime/chat TODOs.
- Replaced chunk-based SSE parsing in `backend/cmd/server/chat.go` with `scanSSEData`, a line-based scanner shared by sidecar chat forwarding and direct AI API streaming.
- Added `anthropicTextDelta` helper so Anthropic-compatible stream events are parsed in one place.
- Added `backend/cmd/server/chat_test.go` covering SSE data scanning and Anthropic text delta extraction.
- Verified with:
  ```bash
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  ```
- Restarted the backend on `0.0.0.0:8080` so the fix is active for the running frontend.

## 2026-07-01 README Refresh

### Project Documentation

- Rewrote `README.md` to match the current Go backend + Vite frontend + Docker sidecar architecture.
- Restored the brand header using both `public/agentbucket-logo-mark-transparent.png` for Buckie and `public/agentbucket-logo-mark.svg` for the AgentBucket wordmark.
- Restored clickable badges for Go, React, Vite, Tailwind, SQLite, and Docker.
- Replaced the old fixed-width ASCII architecture diagram with a Mermaid flowchart to avoid alignment issues.
- Added more accurate local dev commands, including `NO_PROXY`, Go cache env vars, `VITE_API_BASE`, and the note that Vite may choose a different port.
- Added sections for repository layout, `agent.toml`, deployment flow, runtime/chat routing, API overview, sidecar API, development checks, and current limitations.
- Fixed API documentation reference to point at `agentbucket-api-skill/SKILL.md`.

## 2026-07-01 OpenCode Runtime

### Runtime Support

- Added `opencode` as a supported runtime alongside `codex` and `claudecode`.
- Added `backend/cmd/server/runtime.go` with shared runtime enumeration/validation.
- `GET /api/deploy-options` now returns `["codex","claudecode","opencode"]`.
- Deployment validation accepts `opencode`.
- Generated deployment Dockerfiles install `opencode-ai@<runtimeVersion>` and set `AGENTBUCKET_RUNTIME=opencode`.
- Sidecar now includes `OpenCodeRunner`, using `opencode run --model <model>`.
- Local backend runtime fallback also attempts `opencode run --model <model>`.
- Updated `README.md` and `backend/AGENT_STANDARD.md` to document `opencode`.
- Added tests for the opencode Dockerfile path and sidecar runner.
- Verified with:
  ```bash
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  pnpm build
  ```
- Restarted the backend on `0.0.0.0:8080`; deploy options now include `opencode`.

## 2026-07-01 Sidecar Runtime Command Cleanup

### Runtime Command Abstraction

- Split sidecar runtime commands into startup command and chat command.
- `RuntimeRunner` now exposes `Command(config)` for `/agent/start` and `ChatCommand(config, message)` for `/agent/chat`.
- Removed the fragile `/agent/chat` behavior that mutated `cmd.Args` by replacing the final placeholder argument.
- Added sidecar tests that verify startup and chat commands for `codex`, `claudecode`, `opencode`, and unknown-runtime fallback.
- Verified with:
  ```bash
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  ```

## 2026-07-01 API Skill Sync

### Curl Documentation

- Updated `agentbucket-api-skill/SKILL.md` to mention `opencode` in deploy options and Agent standard runtime lists.
- Updated `.skills/agentbucket-admin/SKILL.md` to match current API payload fields:
  - repositories use `id`, `url`, `localPath`, `agentsPath`, and `status`
  - AI token creation uses `secret`
  - deployment examples include `commitHash`
  - `apiTokenId` is numeric
- Updated sidecar streaming note: SSE now works through direct AI API streaming and sidecar chat forwarding.

## 2026-07-01 Deploy Options Runtime Test

### Runtime Regression Coverage

- Added `backend/cmd/server/handlers_test.go`.
- Test calls `deployOptions` directly and asserts `codex`, `claudecode`, and `opencode` are returned.
- Verified with:
  ```bash
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  ```

## 2026-07-01 Chinese README Refresh

### README.zh.md

- Rewrote `README.zh.md` to match the refreshed English README structure.
- Added Buckie transparent PNG and AgentBucket SVG wordmark at the top.
- Restored clickable badges.
- Replaced the old ASCII architecture diagram with Mermaid.
- Synced local dev commands, proxy/cache notes, runtime list including `opencode`, deployment flow, API pointers, and current limitations.

## 2026-07-01 Deploy Runtime Guidance

### Frontend Runtime Selection

- Added runtime-specific helper text under the runtime selector in `src/pages/DeployPage.jsx`.
- Added i18n keys in `src/i18n/zh.js` and `src/i18n/en.js` for:
  - `codex`
  - `claudecode`
  - `opencode`
  - fallback runtime description
- Verified with:
  ```bash
  pnpm build
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  ```

## 2026-07-01 Login Fix

### Auth Persistence And Password UX

- Fixed login failure caused by old SQLite `users` table missing `password_hash`.
- Added `password_hash` migration in `backend/cmd/server/store.go`.
- Added `ensureUserPasswordHashes` so existing seeded users regain default test passwords:
  - `Luna` / `Alex`: `admin123`
  - `Ivy` / `Noah`: `user123`
- Updated user save/load to persist `password_hash`.
- Added backend test coverage in `backend/cmd/server/store_test.go`.
- Added login page password visibility toggle and "记住登录状态".
- Updated auth persistence to support both `localStorage` and `sessionStorage`.
- Verified:
  ```bash
  cd backend
  GOCACHE=/tmp/agentbucket-go-cache GOMODCACHE=/tmp/agentbucket-go-mod go test ./...
  pnpm build
  curl --noproxy '*' -sS -X POST http://127.0.0.1:8080/api/login \
    -H 'Content-Type: application/json' \
    -d '{"username":"Luna","password":"admin123"}'
  ```
- Restarted backend on `0.0.0.0:8080`; `Luna/admin123` now logs in successfully.

## 2026-07-01 Session — P2 Completion + Docker + Core Feature Sweep

### Commit Summary (7 commits pushed)

1. **`60e88d7` — P2: Bus SQLite persistence + deploy capability picker UX**
   - `bus.go`: `post()` writes to `bus_messages` SQLite table alongside in-memory ring buffer
   - `store.go`: `bus_messages` table schema + index
   - `DeployPage.jsx`: flat checkbox lists → `CapabilityCard` summaries + `CapabilityPickerModal` with search, single/multi-select, touched-state tracking

2. **`50b58e4` — Docker deployment support (DooD, not DinD)**
   - `backend/Dockerfile`: multi-stage build (Go + frontend → Alpine with docker-cli)
   - `docker-compose.yml`: mounts `/var/run/docker.sock`, data volume, CCS providers
   - `deploy.go`: `sidecarHost()` uses `AGENTBUCKET_SIDECAR_HOST` env var
   - `server.go`: SPA static file serving when `dist/` exists
   - `store.go`: `AGENTBUCKET_PROVIDERS_DIR` env var for CCS path
   - `.dockerignore`: exclude node_modules, .data, .git

3. **`af74a32` — i18n support + all pages real logic + token scripts**
   - `src/i18n/`: LanguageProvider context, zh.js/en.js translation files with full keyset
   - `Sidebar.jsx`: language switcher (中文/EN), nav labels via `t()`
   - `AiTokensPage.jsx`: working create form with state management + `createAiToken` API
   - `UsersPage.jsx`: role change (admin/user) and enable/disable via `PATCH /api/users/{id}`
   - `ApprovalsPage.jsx`: approve/reject via `POST /api/approvals/{id}/{action}`
   - `AgentChatPage.jsx`: session delete button with confirmation
   - Backend: `PATCH /api/users/{id}`, `POST /api/approvals/{id}/{action}`, `POST /api/approvals`
   - `backend/tokens/`: real Python scripts (test_public, test_admin, github_token, internal_db, notion_token)
   - `.skills/`: agentbucket-dev (development reference) + agentbucket-admin (curl-based management)

4. **`62e05bc` — Deployment progress monitoring page**
   - `DeployProgressPage.jsx`: status cards with color coding, build logs, agent info, auto-polls every 5s
   - Route `/deploy/progress`, nav entry in `data.js`, i18n key mapping in Sidebar

5. **`be4708f` — SSE streaming support for sidecar-deployed agents**
   - Sidecar `handleChat`: split into `handleStreamChat` (SSE via stdout pipe) + `handleOneShotChat`
   - Backend `streamAgentMessage`: checks for running sidecar deployment first, forwards SSE chunks
   - Falls back to direct AI API streaming if sidecar unavailable

6. **`6324cc8` — Real Git integration**
   - `agent_scan.go`: `scanCommits()` uses `git log --format=%H||%s||%aI -20` for real commit history
   - `handlers.go`: auto-clone on GitHub repo bind (goroutine, non-blocking)
   - Falls back to fake commit if `.git` directory doesn't exist

7. **`48fc847` — Tests + i18n sweep + deployment progress fixes**
   - `bus_test.go`: 7 tests (register, post, ring buffer cap 200, persistence, filter)
   - `store_test.go`: 11 tests (CRUD, sessions, 20-limit, approvals, persistence across reopen)
   - Full i18n conversion across all remaining pages (DeployPage 35+, AgentChatPage 20+, AgentsPage, AuthTokensPage, RepositoriesPage, Sidebar)
   - DeployProgressPage: agent filter bar with count badges, clickable agent names for per-agent view
   - `deploy.go`: build log now combines docker build output + container ID on success

8. **`540046b` — Backend auth + graceful shutdown + bus pruning + README**
   - Token-based auth middleware (`Authorization: Bearer` or `X-API-Key`), master token from `AGENTBUCKET_ADMIN_TOKEN`
   - `POST /api/login` endpoint, frontend attaches token on all requests
   - Graceful shutdown: SIGINT handler stops running containers
   - Bus messages pruned to last 1000 rows hourly
   - README.md rewritten (professional tone, flat badges, architecture diagram, EN/ZH links)
   - README.zh.md created (full Chinese translation)

### Current Test Suite

```
Backend: 22 tests passing
  server/  bus_test.go:          7 tests (register, post, ring buffer, persistence)
  server/  store_test.go:       11 tests (CRUD, sessions, limits, approvals)
  server/  deploy_test.go:       2 tests (Dockerfile template, build context)
  server/  agent_scan_test.go:   1 test  (TOML manifest parsing)
  server/  files_test.go:        1 test  (skill copy requires SKILL.md)
  sidecar/ main_test.go:         2 tests (runner selection, status handler)

Frontend: pnpm build passes (chunk-size warning only)
```

### Key Architecture Notes

- **DooD not DinD**: Backend mounts host `/var/run/docker.sock`. Sidecar containers are peers on the host.
- **SSE flow**: Frontend → Backend → Sidecar (stdout pipe streaming) → Runtime CLI (codex/claude)
- **Auth**: Token-based middleware, master token from env var or auto-generated, login returns token
- **i18n**: React context + dynamic import, zh/en locale files, language persisted to localStorage
- **Git**: Real `git log` for commit history (20 entries), auto-clone on GitHub repo bind

---

## Remaining / Future Work

### P1 — Features

| # | Item | Notes |
|---|---|---|
| 1 | **Sidecar persistent chat sessions** | Each chat message spawns a new runtime CLI process. No conversation history in the runtime. Options: (a) inject history into each one-shot call, or (b) keep a persistent stdin/stdout pipe across messages |
| 2 | **Deployment async build with progress** | Current `POST /api/deployments` is synchronous (blocks up to 300s). Move Docker build to goroutine, add `GET /api/deployments/{id}/log` for live progress streaming, update progress page for real-time display |
| 3 | **Agent-specific commit deployments** | `scanRepositories` returns real commits but all commits share current HEAD agent definitions. Need per-commit agent snapshot (git archive or worktree checkout) for deploying from specific historical commits |
| 4 | **Multi-tenancy / RBAC** | Current role system (`super_admin`, `admin`, `user`) is only enforced on frontend nav visibility. Backend endpoints are open (auth middleware protects everything equally). Need per-endpoint role checks |

### P2 — UX Polish

| # | Item | Notes |
|---|---|---|
| 5 | **Auth tokens page create form** | WizardModal has placeholder steps but no real file upload or script parsing. Need actual Python script upload → store → auto-populate function name/argument |
| 6 | **Vite chunk splitting** | Single 580KB JS chunk. Use `manualChunks` or `dynamic import()` for route-based code splitting |
| 7 | **Agent status canonicalization** | Status values use Chinese strings (`"启用"`, `"停用"`, `"已部署"`) as canonical data. Should be English enum values (`"enabled"`, `"disabled"`, `"deployed"`) translated only at UI layer |
| 8 | **Password hashing** | Login endpoint compares plaintext username only (no password check). Seed users have no password field. Needs bcrypt/scrypt + user registration flow |
| 9 | **Docker port collision detection** | `hostPortFor()` maps agent IDs to ports 18000+ via hash but no check for port-in-use. Add port availability check before container run |

### P3 — Infrastructure

| # | Item | Notes |
|---|---|---|
| 10 | **Frontend tests** | Zero automated frontend tests. Add Playwright or Vitest + React Testing Library for key flows (login, deploy wizard, chat SSE) |
| 11 | **API rate limiting** | No rate limiting on any endpoint. Chat and deploy endpoints are CPU/memory heavy |
| 12 | **Session token cleanup** | Tokens stored in localStorage with no expiry. Add JWT-based tokens with refresh |
| 13 | **Docker image cleanup** | Old deployment images accumulate. Add `docker image prune` on deployment delete or periodic cleanup |
| 14 | **Health check on deploy** | After container starts, no verification that sidecar `/health` responds. Add health check before marking `running` status |
| 15 | **Log rotation** | Build logs stored inline in `deployments[].message`. Could grow large. Consider file-based log storage with rotation |

### Refactor Candidates

| # | Item | Notes |
|---|---|---|
| 16 | **Split `handlers.go` (529 lines)** | By resource: `handlers_deploy.go`, `handlers_tokens.go`, `handlers_chat.go`, `handlers_repos.go` |
| 17 | **Extract `chat.go` runtime logic** | Move AI API client, SSE streaming, and context building into an `internal/` package |
| 18 | **Replace hand-rolled TOML parser** | `parseSimpleTOML` in `agent_scan.go` doesn't handle tables, nested values, or spec edge cases. Consider `github.com/BurntSushi/toml` |
| 19 | **Per-agent API endpoint** | `getAgentById` in frontend fetches full agent list and filters. Add `GET /api/agents/{id}` backend endpoint |
