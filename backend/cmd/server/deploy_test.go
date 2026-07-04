package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDockerfileFor(t *testing.T) {
	dockerfile := dockerfileFor("codex", "0.142.3", []string{"apk add --no-cache ripgrep"})
	for _, want := range []string{
		"FROM node:22-alpine",
		"RUN apk add --no-cache ca-certificates bash curl git",
		"RUN apk add --no-cache ripgrep",
		"RUN npm install -g @openai/codex@0.142.3",
		"ENV AGENTBUCKET_RUNTIME=codex",
		"ENV AGENTBUCKET_RUNTIME_VERSION=0.142.3",
		"COPY sidecar/main.go .",
	} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("Dockerfile missing %q\n%s", want, dockerfile)
		}
	}
}

func TestDockerfileForOpenCode(t *testing.T) {
	dockerfile := dockerfileFor("opencode", "latest", nil)
	for _, want := range []string{
		"RUN npm install -g opencode-ai@latest",
		"ENV AGENTBUCKET_RUNTIME=opencode",
		"ENV AGENTBUCKET_RUNTIME_VERSION=latest",
	} {
		if !strings.Contains(dockerfile, want) {
			t.Fatalf("Dockerfile missing %q\n%s", want, dockerfile)
		}
	}
}

func TestWriteBuildContextCopiesRealSidecar(t *testing.T) {
	root := t.TempDir()
	repoRoot := filepath.Join(root, "repo")
	mustWriteFile(t, filepath.Join(repoRoot, "agents", "legal", "agent.toml"), `id = "legal"
name = "Legal"
runtime = "codex"
skills = ["knowledge-base"]
mcps = ["notion-mcp"]
`)
	mustWriteFile(t, filepath.Join(repoRoot, "skills", "knowledge-base", "SKILL.md"), "---\nname: knowledge-base\n---\n")
	mustWriteFile(t, filepath.Join(repoRoot, "mcp", "notion-mcp.json"), `{"id":"notion-mcp","name":"Notion MCP"}`)
	mustWriteFile(t, filepath.Join(root, "cmd", "sidecar", "main.go"), "package main\n\nfunc main() {}\n")

	app := &App{rootDir: root, dataDir: filepath.Join(root, ".data")}
	repo := Repository{ID: "repo", LocalPath: repoRoot, AgentsPath: "agents"}
	agent := Agent{ID: "legal", Path: "agents/legal/agent.toml"}
	req := DeployRequest{
		RepositoryID:   "repo",
		CommitHash:     "abc1234",
		AgentID:        "legal",
		Runtime:        "codex",
		RuntimeVersion: "latest",
		Skills:         []string{"knowledge-base"},
		MCPs:           []string{"notion-mcp"},
	}
	contextDir := filepath.Join(root, "context")
	if err := app.writeBuildContext(contextDir, repo, Commit{Hash: "abc1234"}, agent, req); err != nil {
		t.Fatal(err)
	}

	sidecar, err := os.ReadFile(filepath.Join(contextDir, "sidecar", "main.go"))
	if err != nil {
		t.Fatal(err)
	}
	if string(sidecar) != "package main\n\nfunc main() {}\n" {
		t.Fatalf("unexpected sidecar source: %q", string(sidecar))
	}
	if _, err := os.Stat(filepath.Join(contextDir, "skills", "knowledge-base", "SKILL.md")); err != nil {
		t.Fatalf("skill was not copied: %v", err)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(contextDir), "context.tar")); err != nil {
		t.Fatalf("context tar was not written: %v", err)
	}
}
