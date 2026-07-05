package main

import (
	"bytes"
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
			name:        "gemini explicit version",
			config:      Config{Runtime: "gemini", RuntimeVersion: "latest", Model: "gemini-test"},
			wantName:    "gemini",
			wantVersion: "latest",
			wantCommand: []string{"gemini", "-m", "gemini-test", "-p"},
			wantChat:    []string{"gemini", "-m", "gemini-test", "-p", "hello runtime"},
		},
		{
			name:        "reasonix explicit version",
			config:      Config{Runtime: "reasonix", RuntimeVersion: "latest", Model: "reasonix-test"},
			wantName:    "reasonix",
			wantVersion: "latest",
			wantCommand: []string{"reasonix", "run", "--model", "reasonix-test"},
			wantChat:    []string{"reasonix", "run", "--model", "reasonix-test", "hello runtime"},
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

func TestHealthAndRegisterHandlers(t *testing.T) {
	previous := config
	t.Cleanup(func() { config = previous })
	config = Config{
		AgentID:        "support-bot",
		Runtime:        "gemini",
		RuntimeVersion: "latest",
		Model:          "gemini-test",
		Skills:         []string{"agentbucket-comms"},
		MCPs:           []string{"filesystem"},
	}

	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{name: "health", handler: health, path: "/health"},
		{name: "register", handler: registerAgent, path: "/bus/register"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tt.handler(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d", recorder.Code)
			}
			var body map[string]any
			if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
				t.Fatal(err)
			}
			if body["agent"] != "support-bot" || body["runtime"] != "gemini" {
				t.Fatalf("unexpected body: %#v", body)
			}
		})
	}
}

func TestWithCORSHandlesPreflight(t *testing.T) {
	handler := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/health", nil))
	if recorder.Code != http.StatusNoContent {
		t.Fatalf("status = %d, want 204", recorder.Code)
	}
	if recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("missing CORS header: %#v", recorder.Header())
	}
}

func TestGetTokenProxiesAgentIdentity(t *testing.T) {
	previous := config
	t.Cleanup(func() { config = previous })
	config = Config{AgentID: "agent-a"}

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/tokens/resolve" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("X-Agent-ID"); got != "agent-a" {
			t.Fatalf("X-Agent-ID = %q", got)
		}
		var payload map[string]any
		if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
			t.Fatal(err)
		}
		if payload["tokenId"].(float64) != 101 {
			t.Fatalf("payload = %#v", payload)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusAccepted)
		_, _ = w.Write([]byte(`{"token":"proxied"}`))
	}))
	defer upstream.Close()
	t.Setenv("AGENTBUCKET_URL", upstream.URL)

	recorder := httptest.NewRecorder()
	getToken(recorder, httptest.NewRequest(http.MethodPost, "/tokens/get", bytes.NewBufferString(`{"tokenId":101}`)))
	if recorder.Code != http.StatusAccepted {
		t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if !strings.Contains(recorder.Body.String(), "proxied") {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}

func TestGetTokenRejectsBadJSON(t *testing.T) {
	recorder := httptest.NewRecorder()
	getToken(recorder, httptest.NewRequest(http.MethodPost, "/tokens/get", strings.NewReader("{")))
	if recorder.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", recorder.Code)
	}
}

func TestHandleChatRejectsInvalidRequests(t *testing.T) {
	for _, tt := range []struct {
		name       string
		method     string
		body       string
		wantStatus int
	}{
		{name: "method", method: http.MethodGet, wantStatus: http.StatusMethodNotAllowed},
		{name: "empty message", method: http.MethodPost, body: `{"message":""}`, wantStatus: http.StatusBadRequest},
		{name: "bad json", method: http.MethodPost, body: `{`, wantStatus: http.StatusBadRequest},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			handleChat(recorder, httptest.NewRequest(tt.method, "/agent/chat", strings.NewReader(tt.body)))
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d", recorder.Code, tt.wantStatus)
			}
		})
	}
}

func TestStopAgentWithoutProcess(t *testing.T) {
	agentMu.Lock()
	agentCmd = nil
	agentMu.Unlock()

	recorder := httptest.NewRecorder()
	stopAgent(recorder, httptest.NewRequest(http.MethodPost, "/agent/stop", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	if !strings.Contains(recorder.Body.String(), `"ok":true`) {
		t.Fatalf("unexpected body: %s", recorder.Body.String())
	}
}
