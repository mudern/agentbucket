package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
)

// --- Deploy Options ---

func TestDeployOptionsIncludesSupportedRuntimes(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}
	recorder := httptest.NewRecorder()
	app.deployOptions(recorder, httptest.NewRequest(http.MethodGet, "/api/deploy-options", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}
	var body DeployOptions
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"codex", "claudecode", "opencode"} {
		if !containsString(body.Runtimes, want) {
			t.Fatalf("runtimes = %#v, missing %q", body.Runtimes, want)
		}
	}
}

// --- Agent List: Only Deployed ---

func TestAgentsOnlyReturnsDeployed(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "agents", "deployed", "agent.toml"),
		`id = "deployed"
name = "Deployed Agent"
`)
	mustWriteFile(t, filepath.Join(dir, "agents", "offline", "agent.toml"),
		`id = "offline"
name = "Offline Agent"
`)

	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	// Create a running deployment for one agent
	_ = store.update(func(s *State) error {
		s.Repositories = []Repository{{
			ID: "test", Provider: "Local", LocalPath: dir, AgentsPath: "agents", Status: "启用",
		}}
		s.Deployments = []Deployment{{
			ID: "dep-1", AgentID: "deployed", Status: "running",
			Model: "test-model", Runtime: "codex", APITokenID: 1,
		}}
		s.AITokens = []AIToken{{ID: 1, Name: "test-token", Status: "启用"}}
		return nil
	})

	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}
	recorder := httptest.NewRecorder()
	app.agents(recorder, httptest.NewRequest(http.MethodGet, "/api/agents", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status = %d", recorder.Code)
	}
	var agents []Agent
	if err := json.Unmarshal(recorder.Body.Bytes(), &agents); err != nil {
		t.Fatal(err)
	}
	if len(agents) != 1 {
		t.Fatalf("expected 1 deployed agent, got %d", len(agents))
	}
	if agents[0].ID != "deployed" {
		t.Fatalf("expected deployed agent, got %q", agents[0].ID)
	}
	if agents[0].Status != "已部署" {
		t.Fatalf("expected status '已部署', got %q", agents[0].Status)
	}
}

func TestAgentsDeploymentConfigOverrides(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "agents", "test", "agent.toml"),
		`id = "test"
name = "Test"
model = "old-model"
runtime = "claudecode"
api_token = "old-token"
`)

	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	_ = store.update(func(s *State) error {
		s.Repositories = []Repository{{
			ID: "test", Provider: "Local", LocalPath: dir, AgentsPath: "agents", Status: "启用",
		}}
		s.Deployments = []Deployment{{
			ID: "dep-1", AgentID: "test", Status: "running",
			Model: "new-model", Runtime: "codex", APITokenID: 1,
		}}
		s.AITokens = []AIToken{{ID: 1, Name: "new-token", Status: "启用"}}
		return nil
	})

	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}
	recorder := httptest.NewRecorder()
	app.agents(recorder, httptest.NewRequest(http.MethodGet, "/api/agents", nil))

	var agents []Agent
	json.Unmarshal(recorder.Body.Bytes(), &agents)
	if len(agents) != 1 {
		t.Fatal("expected 1 agent")
	}
	if agents[0].Model != "new-model" {
		t.Fatalf("expected deployment model 'new-model', got %q", agents[0].Model)
	}
	if agents[0].Runtime != "codex" {
		t.Fatalf("expected deployment runtime 'codex', got %q", agents[0].Runtime)
	}
	if agents[0].APIToken != "new-token" {
		t.Fatalf("expected resolved token 'new-token', got %q", agents[0].APIToken)
	}
}

// --- SSE Parsing ---

func TestScanSSEData(t *testing.T) {
	input := strings.Join([]string{
		"event: content_block_delta",
		"data: first",
		"",
		": keepalive",
		"data: second with spaces   ",
		"data:",
		"data: third",
		"",
	}, "\n")

	var got []string
	if err := scanSSEData(strings.NewReader(input), func(data string) {
		got = append(got, data)
	}); err != nil {
		t.Fatalf("scanSSEData returned error: %v", err)
	}

	want := []string{"first", "second with spaces", "third"}
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestAnthropicTextDelta(t *testing.T) {
	data := `{"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`
	if got := anthropicTextDelta(data); got != "hello" {
		t.Fatalf("expected text delta, got %q", got)
	}
	if got := anthropicTextDelta(`{"type":"message_start"}`); got != "" {
		t.Fatalf("expected non-delta event to be ignored, got %q", got)
	}
	if got := anthropicTextDelta("[DONE]"); got != "" {
		t.Fatalf("expected DONE to be ignored, got %q", got)
	}
}

func TestSSEError(t *testing.T) {
	recorder := httptest.NewRecorder()
	sseError(recorder, "test error message")

	body := recorder.Body.String()
	if !strings.Contains(body, `"error"`) {
		t.Fatalf("expected error in SSE body, got: %s", body)
	}
	if !strings.Contains(body, "test error message") {
		t.Fatalf("expected error message in body, got: %s", body)
	}
}

// --- Password Hashing ---

func TestHashPasswordSalted(t *testing.T) {
	h1 := hashPassword("admin123")
	h2 := hashPassword("admin123")
	// Each call should produce a different salt → different hash
	if h1 == h2 {
		t.Fatal("expected different salts for each hashPassword call")
	}
	// Both should verify
	if !verifyPassword("admin123", h1) {
		t.Fatal("h1 should verify")
	}
	if !verifyPassword("admin123", h2) {
		t.Fatal("h2 should verify")
	}
	// Wrong password should fail
	if verifyPassword("wrong", h1) {
		t.Fatal("wrong password should not verify")
	}
}

func TestVerifyPasswordLegacy(t *testing.T) {
	legacy := hashPasswordLegacy("admin123")
	if !verifyPassword("admin123", legacy) {
		t.Fatal("legacy hash should verify")
	}
}

// --- Helpers ---

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
