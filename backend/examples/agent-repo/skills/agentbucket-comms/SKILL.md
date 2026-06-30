---
name: agentbucket-comms
description: Agent bucket agent-to-agent communication and token resolution. Use this skill when agents need to discover other agents on the bus, send messages to peers, query the shared message bus, or resolve authorized tokens through the AgentBucket control plane.
---

# AgentBucket Communication Skill

This skill enables agents to communicate with each other on the AgentBucket shared bus and resolve authorized tokens through the sidecar.

## API Base

```bash
export AGENTBUCKET_URL="${AGENTBUCKET_URL:-http://host.docker.internal:8080}"
export SIDECAR_URL="${SIDECAR_URL:-http://localhost:8088}"
```

## List Agents on the Bus

Discover all agents currently registered on the shared bus:

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_URL/api/bus/agents"
```

Returns an array of agents with `agentId`, `name`, `status`, `endpoint`, `lastSeen`.

## Register This Agent on the Bus

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_URL/api/bus/agents/$AGENTBUCKET_AGENT_ID/register" \
  -H 'Content-Type: application/json' \
  -d "{\"name\":\"$(hostname)\",\"status\":\"online\",\"endpoint\":\"$SIDECAR_URL\"}"
```

## Send a Message to Another Agent

```bash
curl --noproxy '*' -sS -X POST "$AGENTBUCKET_URL/api/bus/agents/$AGENTBUCKET_AGENT_ID/message" \
  -H 'Content-Type: application/json' \
  -d '{"toAgent":"legal-summarizer","content":"Hello, can you summarize the latest deployment status?"}'
```

The message is stored on the shared bus. The target agent can poll for messages.

## Read Bus Messages

```bash
curl --noproxy '*' -sS "$AGENTBUCKET_URL/api/bus/messages"
```

Returns all messages on the bus (up to 200 messages).

## Resolve an Authorized Token

Resolve a token through the sidecar. The sidecar proxies the request to the main backend with the agent's identity:

```bash
curl --noproxy '*' -sS -X POST "$SIDECAR_URL/tokens/get" \
  -H 'Content-Type: application/json' \
  -d '{"tokenId":101,"param":"your-param"}'
```

Token IDs:
- `101` - Test Public API (available to all deployed agents)
- `102` - Test Admin API (only available to agents with explicit authorization)
- `103` - Test Disabled API (always fails)
- `104` - GitHub Token
- `105` - Internal DB

The response includes the resolved token value.

## Check Sidecar Status

```bash
curl --noproxy '*' -sS "$SIDECAR_URL/status"
```

Returns runtime status, online state, skills, MCPs, and auth token assignments.

## Conversation Protocol

When another agent sends a message via the bus:
1. Poll `GET /api/bus/messages` to check for messages addressed to this agent
2. Process the message and send a response via `POST /api/bus/agents/{fromAgent}/message`
3. Messages are ephemeral - the bus retains the last 200 messages

## Agent Discovery Flow

1. Register: `POST /api/bus/agents/{myId}/register`
2. Discover: `GET /api/bus/agents`
3. Communicate: `POST /api/bus/agents/{myId}/message`

## Token Authorization Flow

Tokens are attached to agent deployments during the deployment step. When an agent requests a token:
1. Sidecar forwards `POST /tokens/get` to `$AGENTBUCKET_URL/api/tokens/resolve` with `X-Agent-ID` header
2. Backend checks: token exists, token is enabled, agent is authorized to access this token
3. If authorized, returns the token value
4. If unauthorized, returns HTTP 403

This means some agents can access certain tokens and others cannot, depending on deployment configuration.
