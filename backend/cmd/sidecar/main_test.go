package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestRunnerFor(t *testing.T) {
	t.Setenv("AGENTBUCKET_RUNTIME_VERSION", "stable")

	tests := []struct {
		name        string
		config      Config
		wantName    string
		wantVersion string
		wantCommand []string
		wantChat    []string
	}{
		{
			name:        "codex explicit version",
			config:      Config{Runtime: "codex", RuntimeVersion: "0.142.3", Model: "gpt-test"},
			wantName:    "codex",
			wantVersion: "0.142.3",
			wantCommand: []string{"codex", "exec", "--model", "gpt-test"},
			wantChat:    []string{"codex", "exec", "--model", "gpt-test", "hello runtime"},
		},
		{
			name:        "claudecode env version",
			config:      Config{Runtime: "claudecode", Model: "sonnet-test"},
			wantName:    "claudecode",
			wantVersion: "stable",
			wantCommand: []string{"claude", "-p"},
			wantChat:    []string{"claude", "-p", "hello runtime"},
		},
		{
			name:        "opencode explicit version",
			config:      Config{Runtime: "opencode", RuntimeVersion: "latest", Model: "qwen-test"},
			wantName:    "opencode",
			wantVersion: "latest",
			wantCommand: []string{"opencode", "run", "--model", "qwen-test"},
			wantChat:    []string{"opencode", "run", "--model", "qwen-test", "hello runtime"},
		},
		{
			name:        "unknown runtime falls back to codex",
			config:      Config{Runtime: "unknown", Model: "fallback-model"},
			wantName:    "codex",
			wantVersion: "stable",
			wantCommand: []string{"codex", "exec", "--model", "fallback-model"},
			wantChat:    []string{"codex", "exec", "--model", "fallback-model", "hello runtime"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			runner := runnerFor(tt.config)
			if runner.Name() != tt.wantName {
				t.Fatalf("runner name = %q, want %q", runner.Name(), tt.wantName)
			}
			if runner.Version() != tt.wantVersion {
				t.Fatalf("runner version = %q, want %q", runner.Version(), tt.wantVersion)
			}
			cmd := runner.Command(tt.config)
			joined := strings.Join(cmd.Args, " ")
			for _, part := range tt.wantCommand {
				if !strings.Contains(joined, part) {
					t.Fatalf("command %q does not contain %q", joined, part)
				}
			}
			if cmd.Dir != "/app/agent" {
				t.Fatalf("cmd dir = %q, want /app/agent", cmd.Dir)
			}
			chatCmd := runner.ChatCommand(tt.config, "hello runtime")
			chatJoined := strings.Join(chatCmd.Args, " ")
			for _, part := range tt.wantChat {
				if !strings.Contains(chatJoined, part) {
					t.Fatalf("chat command %q does not contain %q", chatJoined, part)
				}
			}
			if chatCmd.Dir != "/app/agent" {
				t.Fatalf("chat cmd dir = %q, want /app/agent", chatCmd.Dir)
			}
		})
	}
}

func TestStatusHandler(t *testing.T) {
	previous := config
	t.Cleanup(func() { config = previous })
	config = Config{
		AgentID:        "legal-summarizer",
		Runtime:        "codex",
		RuntimeVersion: "latest",
		Model:          "GPT-4.1",
		Skills:         []string{"knowledge-base"},
		MCPs:           []string{"notion-mcp"},
		AuthTokens:     []int{101},
	}

	recorder := httptest.NewRecorder()
	status(recorder, httptest.NewRequest(http.MethodGet, "/status", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}
	var body map[string]any
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	if body["agent"] != "legal-summarizer" {
		t.Fatalf("agent = %v", body["agent"])
	}
	if body["runtime"] != "codex" {
		t.Fatalf("runtime = %v", body["runtime"])
	}
	if body["online"] != false {
		t.Fatalf("online = %v, want false", body["online"])
	}
}
