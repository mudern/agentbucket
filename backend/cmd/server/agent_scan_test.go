package main

import (
	"path/filepath"
	"testing"
)

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
