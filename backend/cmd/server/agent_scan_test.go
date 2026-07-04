package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// --- TOML Parsing ---

func TestParseAgentManifestTOML(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.toml")
	mustWriteFile(t, path, `id = "legal-summarizer"
name = "Legal Summarizer"
description = "Summarize legal docs"
model = "GPT-4.1"
runtime = "codex"
runtime_version = "0.142.3"
api_token = "deepseek"
skills = ["knowledge-base", "document-parser"]
mcps = ["notion-mcp"]
extra_install = ["apk add --no-cache ripgrep"]
`)
	agent, err := parseAgentManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if agent.ID != "legal-summarizer" {
		t.Fatalf("id = %q", agent.ID)
	}
	if agent.RuntimeVersion != "0.142.3" {
		t.Fatalf("runtime version = %q", agent.RuntimeVersion)
	}
	if got := len(agent.Skills); got != 2 {
		t.Fatalf("skills len = %d", got)
	}
	if got := len(agent.ExtraInstall); got != 1 {
		t.Fatalf("extra install len = %d", got)
	}
}

func TestParseAgentManifestMinimal(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.toml")
	mustWriteFile(t, path, `id = "minimal"
name = "Minimal Agent"
`)
	agent, err := parseAgentManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if agent.Runtime != "" {
		t.Fatalf("expected empty runtime, got %q", agent.Runtime)
	}
	// parseAgentManifest doesn't set defaults; scanAgents does
}

func TestParseAgentManifestEmptySkills(t *testing.T) {
	path := filepath.Join(t.TempDir(), "agent.toml")
	mustWriteFile(t, path, `id = "no-skills"
name = "No Skills"
skills = []
`)
	agent, err := parseAgentManifest(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(agent.Skills) != 0 {
		t.Fatalf("expected 0 skills, got %d", len(agent.Skills))
	}
}

// --- Skill Copying ---

func TestCopySelectedSkillsRequiresSkillMD(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "skills")
	dst := filepath.Join(root, "out")
	mustWriteFile(t, filepath.Join(src, "valid", "SKILL.md"), "---\nname: valid\n---\n")
	if err := copySelectedSkills(src, dst, []string{"valid"}); err != nil {
		t.Fatalf("valid skill copy failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "valid", "SKILL.md")); err != nil {
		t.Fatalf("copied skill missing: %v", err)
	}
	if err := copySelectedSkills(src, dst, []string{"missing"}); err == nil {
		t.Fatal("expected missing SKILL.md error")
	}
}

func TestCopySelectedSkillsEmptySelection(t *testing.T) {
	root := t.TempDir()
	if err := copySelectedSkills(filepath.Join(root, "skills"), filepath.Join(root, "out"), nil); err != nil {
		t.Fatalf("empty selection should succeed: %v", err)
	}
}

// --- Git-based Agent Scanning ---

func TestScanAgentsWithoutGit(t *testing.T) {
	dir := t.TempDir()
	repo := Repository{LocalPath: dir, AgentsPath: "agents"}
	mustWriteFile(t, filepath.Join(dir, "agents", "test-agent", "agent.toml"),
		`id = "test-agent"
name = "Test Agent"
model = "test-model"
runtime = "codex"
`)
	agents := scanAgents(repo)
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent, got %d", len(agents))
	}
	if agents[0].ID != "test-agent" {
		t.Fatalf("expected test-agent, got %q", agents[0].ID)
	}
	if agents[0].Path != "agents/test-agent/agent.toml" {
		t.Fatalf("expected path, got %q", agents[0].Path)
	}
}

func TestScanAgentsWithGitRepo(t *testing.T) {
	dir := t.TempDir()
	// Setup a real git repo
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@agentbucket.dev")
	runGit(t, dir, "config", "user.name", "Test")

	mustWriteFile(t, filepath.Join(dir, "agents", "hello-agent", "agent.toml"),
		`id = "hello-agent"
name = "Hello"
model = "test-model"
runtime = "claudecode"
`)
	runGit(t, dir, "add", "agents/hello-agent/agent.toml")
	runGit(t, dir, "commit", "-m", "add hello agent")

	repo := Repository{LocalPath: dir, AgentsPath: "agents"}
	agents := scanAgents(repo)
	if len(agents) != 1 {
		t.Fatalf("expected 1 agent from git tree, got %d", len(agents))
	}
	if agents[0].ID != "hello-agent" {
		t.Fatalf("expected hello-agent, got %q", agents[0].ID)
	}
}

func TestScanAgentsGitIgnoresUncommitted(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@agentbucket.dev")
	runGit(t, dir, "config", "user.name", "Test")

	// Commit one agent
	mustWriteFile(t, filepath.Join(dir, "agents", "committed", "agent.toml"),
		`id = "committed"
name = "Committed Agent"
`)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add committed")

	// Write an uncommitted agent
	mustWriteFile(t, filepath.Join(dir, "agents", "uncommitted", "agent.toml"),
		`id = "uncommitted"
name = "Uncommitted Agent"
`)
	// NOTE: intentionally NOT git add + commit

	repo := Repository{LocalPath: dir, AgentsPath: "agents"}
	agents := scanAgents(repo)
	if len(agents) != 1 {
		ids := make([]string, len(agents))
		for i, a := range agents {
			ids[i] = a.ID
		}
		t.Fatalf("expected only 1 committed agent, got %d: %v", len(agents), ids)
	}
	if agents[0].ID != "committed" {
		t.Fatalf("expected committed agent, got %q", agents[0].ID)
	}
}

func TestScanAgentsMultipleCommits(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@agentbucket.dev")
	runGit(t, dir, "config", "user.name", "Test")

	// Commit 1: add agent A
	mustWriteFile(t, filepath.Join(dir, "agents", "agent-a", "agent.toml"),
		`id = "agent-a"
name = "Agent A"
`)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add agent A")

	// Commit 2: add agent B
	mustWriteFile(t, filepath.Join(dir, "agents", "agent-b", "agent.toml"),
		`id = "agent-b"
name = "Agent B"
`)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "add agent B")

	repo := Repository{LocalPath: dir, AgentsPath: "agents"}
	commits := scanCommits(repo)
	if len(commits) < 2 {
		t.Fatalf("expected at least 2 commits, got %d", len(commits))
	}
	// Latest commit (HEAD) should have both agents
	if len(commits[0].Agents) != 2 {
		t.Fatalf("expected 2 agents at HEAD, got %d", len(commits[0].Agents))
	}
}

func TestScanRepositoriesMultipleCommits(t *testing.T) {
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@agentbucket.dev")
	runGit(t, dir, "config", "user.name", "Test")

	mustWriteFile(t, filepath.Join(dir, "agents", "multi-agent", "agent.toml"),
		`id = "multi-agent"
name = "Multi"
`)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "v1")

	// Second commit (modify agent.toml)
	mustWriteFile(t, filepath.Join(dir, "agents", "multi-agent", "agent.toml"),
		`id = "multi-agent"
name = "Multi V2"
model = "v2-model"
`)
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-m", "v2")

	app := &App{rootDir: dir, dataDir: filepath.Join(dir, ".data")}
	repos := app.scanRepositories([]Repository{{
		ID: "test", Provider: "Local", LocalPath: dir, AgentsPath: "agents",
	}})

	if len(repos) != 1 {
		t.Fatal("expected 1 repo")
	}
	if len(repos[0].Commits) < 2 {
		t.Fatalf("expected at least 2 commits, got %d", len(repos[0].Commits))
	}
	// HEAD agent from git tree should have latest name
	headAgents := repos[0].Commits[0].Agents
	if len(headAgents) != 1 || headAgents[0].Name != "Multi V2" {
		t.Fatalf("expected 'Multi V2' at HEAD, got %q", headAgents[0].Name)
	}
	// Second commit's agents (same as HEAD since they share scanAgents result)
}

// --- Helpers ---

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, string(out))
	}
}
