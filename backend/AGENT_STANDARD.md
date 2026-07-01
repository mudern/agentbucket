# AgentBucket Agent Standard

An Agent is defined by `agent.toml` inside one agent directory:

```toml
id = "legal-summarizer"
name = "Legal Summarizer"
description = "Summarize legal documents."
model = "GPT-4.1"
runtime = "codex"
runtime_version = "latest"
api_token = "DeepSeek Shared"
skills = ["knowledge-base", "document-parser"]
mcps = ["notion-mcp", "filesystem-mcp"]
```

Required fields:

- `id`: stable agent id, used for deployment and container naming.
- `name`: display name.
- `runtime`: currently `codex`, `claudecode`, or `opencode`.
- `skills`: list of standard skill directory ids.
- `mcps`: list of MCP config ids.

Skill standard:

- Each skill must be a directory under `skills/`.
- Each skill directory must contain `SKILL.md`.
- `SKILL.md` should include frontmatter with at least `name` and `description`.

Repository layout:

```text
agents/
  legal-summarizer/
    agent.toml
skills/
  knowledge-base/
    SKILL.md
mcp/
  github-mcp.json
```
