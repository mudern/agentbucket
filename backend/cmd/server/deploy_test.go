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

func TestDockerfileForGeminiAndReasonix(t *testing.T) {
	tests := []struct {
		runtime string
		install string
		env     string
	}{
		{runtime: "gemini", install: "RUN npm install -g @google/gemini-cli@latest", env: "ENV AGENTBUCKET_RUNTIME=gemini"},
		{runtime: "reasonix", install: "RUN npm install -g reasonix@latest", env: "ENV AGENTBUCKET_RUNTIME=reasonix"},
	}
	for _, tt := range tests {
		t.Run(tt.runtime, func(t *testing.T) {
			dockerfile := dockerfileFor(tt.runtime, "latest", nil)
			for _, want := range []string{tt.install, tt.env} {
				if !strings.Contains(dockerfile, want) {
					t.Fatalf("Dockerfile missing %q\n%s", want, dockerfile)
				}
			}
		})
	}
}

func TestDeploymentEnvInjectsAuthorizedTokens(t *testing.T) {
	app := &App{store: &Store{state: State{
		AITokens: []AIToken{
			{ID: 7, Name: "gemini-prod", Status: "启用", Secret: "ai-secret", BaseURL: "https://ai.example", Model: "gemini-test"},
		},
		AuthTokens: []AuthToken{
			{ID: 101, Name: "GitHub Token", Secret: "ghp_secret", Status: "启用"},
			{ID: 102, Name: "Disabled Token", Secret: "disabled", Status: "停用"},
			{ID: 103, Name: "Other Token", Secret: "other", Status: "启用"},
		},
	}}}
	env := app.deploymentEnv(Deployment{
		AgentID:        "agent-a",
		APITokenID:     7,
		Model:          "fallback-model",
		Runtime:        "gemini",
		RuntimeVersion: "latest",
		AuthTokens:     []int{101, 102},
	})
	joined := strings.Join(env, "\n")
	for _, want := range []string{
		"AGENTBUCKET_AI_TOKEN=ai-secret",
		"GEMINI_API_KEY=ai-secret",
		"GOOGLE_API_KEY=ai-secret",
		"AGENTBUCKET_AUTH_TOKEN_101=ghp_secret",
		"AGENTBUCKET_AUTH_TOKEN_GITHUB_TOKEN=ghp_secret",
		`"101":"ghp_secret"`,
		`"GitHub Token":"ghp_secret"`,
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("env missing %q\n%s", want, joined)
		}
	}
	for _, notWant := range []string{"disabled", "other"} {
		if strings.Contains(joined, notWant) {
			t.Fatalf("env should not contain %q\n%s", notWant, joined)
		}
	}
}

func TestDockerRunArgsIncludesGeneratedEnvironment(t *testing.T) {
	app := &App{store: &Store{state: State{
		AITokens:   []AIToken{{ID: 9, Name: "openai-prod", Status: "启用", Secret: "sk-test", BaseURL: "https://api.example", Model: "gpt-test"}},
		AuthTokens: []AuthToken{{ID: 42, Name: "Deploy Key", Secret: "deploy-secret", Status: "启用"}},
	}}}
	args := app.dockerRunArgs(Deployment{
		AgentID:        "agent-a",
		APITokenID:     9,
		Model:          "fallback-model",
		Runtime:        "codex",
		RuntimeVersion: "latest",
		AuthTokens:     []int{42},
		ContainerName:  "agentbucket-agent-a",
		HostPort:       18088,
		ImageTag:       "agentbucket/agent-a:abc123",
	})
	joined := strings.Join(args, "\n")
	for _, want := range []string{
		"run",
		"--name\nagentbucket-agent-a",
		"-p\n127.0.0.1:18088:8088",
		"AGENTBUCKET_RUNTIME=codex",
		"OPENAI_API_KEY=sk-test",
		"AGENTBUCKET_AUTH_TOKEN_DEPLOY_KEY=deploy-secret",
		"agentbucket/agent-a:abc123",
	} {
		if !strings.Contains(joined, want) {
			t.Fatalf("docker args missing %q\n%s", want, joined)
		}
	}
}

func TestAuthTokenEnvNameSanitization(t *testing.T) {
	if got := envName(" GitHub Token / Prod "); got != "GITHUB_TOKEN_PROD" {
		t.Fatalf("envName = %q", got)
	}
	if got := envName("!!!"); got != "" {
		t.Fatalf("envName for punctuation = %q, want empty", got)
	}
	if got := firstNonEmpty("", "model-a", "model-b"); got != "model-a" {
		t.Fatalf("firstNonEmpty = %q", got)
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
