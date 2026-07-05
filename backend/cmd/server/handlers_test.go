package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"path/filepath"
	"strings"
	"testing"
	"time"
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
	_ = store.update(func(state *State) error {
		state.AITokens = []AIToken{{ID: 1, Name: "secret-token", Provider: "TEST", Status: "启用", Model: "model-a", Secret: "hidden-secret"}}
		return nil
	})
	recorder := httptest.NewRecorder()
	app.deployOptions(recorder, httptest.NewRequest(http.MethodGet, "/api/deploy-options", nil))

	if recorder.Code != http.StatusOK {
		t.Fatalf("status code = %d, want 200", recorder.Code)
	}
	var body DeployOptions
	if err := json.Unmarshal(recorder.Body.Bytes(), &body); err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"codex", "claudecode", "opencode", "gemini", "reasonix"} {
		if !containsString(body.Runtimes, want) {
			t.Fatalf("runtimes = %#v, missing %q", body.Runtimes, want)
		}
	}
	if len(body.AITokens) != 1 || body.AITokens[0].Secret != "" {
		t.Fatalf("deploy options leaked AI token secret: %#v", body.AITokens)
	}
}

func TestAuthTokensCreateDefaultsAndListRedactsSecret(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}
	body := bytes.NewBufferString(`{"name":"Deploy Key","description":"read only","secret":"secret-value"}`)
	recorder := httptest.NewRecorder()
	app.authTokens(recorder, httptest.NewRequest(http.MethodPost, "/api/auth-tokens", body))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var created AuthToken
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.ID == 0 || created.Status != "启用" || created.UpdatedAt == "" {
		t.Fatalf("created token missing defaults: %#v", created)
	}

	recorder = httptest.NewRecorder()
	app.authTokens(recorder, httptest.NewRequest(http.MethodGet, "/api/auth-tokens", nil))
	if recorder.Code != http.StatusOK {
		t.Fatalf("list status = %d", recorder.Code)
	}
	var tokens []AuthToken
	if err := json.Unmarshal(recorder.Body.Bytes(), &tokens); err != nil {
		t.Fatal(err)
	}
	for _, token := range tokens {
		if token.Secret != "" {
			t.Fatalf("auth token %d leaked secret %q", token.ID, token.Secret)
		}
	}
}

func TestResolveTokenAuthorizationMatrix(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	_ = store.update(func(s *State) error {
		s.AuthTokens = []AuthToken{
			{ID: 1, Name: "Allowed", Description: "allowed token", Secret: "allowed-secret", Status: "启用"},
			{ID: 2, Name: "Disabled", Secret: "disabled-secret", Status: "停用"},
			{ID: 3, Name: "Unassigned", Secret: "unassigned-secret", Status: "启用"},
		}
		s.Deployments = []Deployment{
			{ID: "dep-running", AgentID: "agent-a", Status: "running", AuthTokens: []int{1, 2}},
			{ID: "dep-stopped", AgentID: "agent-b", Status: "stopped", AuthTokens: []int{1}},
		}
		return nil
	})
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	tests := []struct {
		name       string
		agentID    string
		body       string
		wantStatus int
		wantToken  string
	}{
		{name: "missing agent", body: `{"tokenId":1}`, wantStatus: http.StatusUnauthorized},
		{name: "bad json", agentID: "agent-a", body: `{`, wantStatus: http.StatusBadRequest},
		{name: "not found", agentID: "agent-a", body: `{"tokenId":404}`, wantStatus: http.StatusNotFound},
		{name: "disabled", agentID: "agent-a", body: `{"tokenId":2}`, wantStatus: http.StatusForbidden},
		{name: "not assigned", agentID: "agent-a", body: `{"tokenId":3}`, wantStatus: http.StatusForbidden},
		{name: "stopped deployment", agentID: "agent-b", body: `{"tokenId":1}`, wantStatus: http.StatusForbidden},
		{name: "allowed", agentID: "agent-a", body: `{"tokenId":1}`, wantStatus: http.StatusOK, wantToken: "allowed-secret"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest(http.MethodPost, "/api/tokens/resolve", strings.NewReader(tt.body))
			if tt.agentID != "" {
				req.Header.Set("X-Agent-ID", tt.agentID)
			}
			recorder := httptest.NewRecorder()
			app.resolveToken(recorder, req)
			if recorder.Code != tt.wantStatus {
				t.Fatalf("status = %d, want %d, body = %s", recorder.Code, tt.wantStatus, recorder.Body.String())
			}
			if tt.wantToken != "" && !strings.Contains(recorder.Body.String(), tt.wantToken) {
				t.Fatalf("response missing token %q: %s", tt.wantToken, recorder.Body.String())
			}
		})
	}
}

func TestBasicResourceHandlers(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	for _, tt := range []struct {
		name    string
		handler http.HandlerFunc
		path    string
	}{
		{name: "health", handler: app.health, path: "/health"},
		{name: "current user", handler: app.currentUser, path: "/api/current-user"},
		{name: "users", handler: app.users, path: "/api/users"},
		{name: "approvals", handler: app.approvals, path: "/api/approvals"},
		{name: "stats", handler: app.stats, path: "/api/stats"},
	} {
		t.Run(tt.name, func(t *testing.T) {
			recorder := httptest.NewRecorder()
			tt.handler(recorder, httptest.NewRequest(http.MethodGet, tt.path, nil))
			if recorder.Code != http.StatusOK {
				t.Fatalf("status = %d, body = %s", recorder.Code, recorder.Body.String())
			}
			if !strings.Contains(recorder.Header().Get("Content-Type"), "application/json") {
				t.Fatalf("expected json content type, got %q", recorder.Header().Get("Content-Type"))
			}
		})
	}
}

func TestAITokenCreatePatchDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	recorder := httptest.NewRecorder()
	app.aiTokens(recorder, httptest.NewRequest(http.MethodPost, "/api/ai-tokens", bytes.NewBufferString(`{
		"name":"OpenAI",
		"provider":"OPENAI",
		"secret":"sk-test",
		"baseUrl":"https://api.example",
		"model":"gpt-test"
	}`)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	var created AIToken
	if err := json.Unmarshal(recorder.Body.Bytes(), &created); err != nil {
		t.Fatal(err)
	}
	if created.Status != "启用" || created.Scope != "manual" || created.Usage != "unused" {
		t.Fatalf("created token missing defaults: %#v", created)
	}
	if created.Secret != "" {
		t.Fatalf("created token leaked secret")
	}

	tokenID := created.ID
	if token, ok := findAIToken(store.snapshot().AITokens, tokenID); !ok || token.Secret != "sk-test" {
		t.Fatalf("stored token missing secret: %#v", token)
	}
	recorder = httptest.NewRecorder()
	app.aiTokens(recorder, httptest.NewRequest(http.MethodGet, "/api/ai-tokens", nil))
	if strings.Contains(recorder.Body.String(), "sk-test") {
		t.Fatalf("list response leaked secret: %s", recorder.Body.String())
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/ai-tokens/id", strings.NewReader(`{"status":"停用"}`))
	req.SetPathValue("id", fmt.Sprint(tokenID))
	recorder = httptest.NewRecorder()
	app.patchAIToken(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if token, ok := findAIToken(store.snapshot().AITokens, tokenID); !ok || token.Status != "停用" {
		t.Fatalf("patched token not found or wrong status: %#v", store.snapshot().AITokens)
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/ai-tokens/id", nil)
	req.SetPathValue("id", fmt.Sprint(tokenID))
	recorder = httptest.NewRecorder()
	app.deleteAIToken(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if _, ok := findAIToken(store.snapshot().AITokens, tokenID); ok {
		t.Fatalf("token was not deleted: %#v", store.snapshot().AITokens)
	}
}

func TestRepositoriesLocalCreatePatchDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	recorder := httptest.NewRecorder()
	app.repositories(recorder, httptest.NewRequest(http.MethodPost, "/api/repositories", bytes.NewBufferString(`{
		"id":"repo-a",
		"provider":"Local",
		"localPath":"/tmp/repo-a"
	}`)))
	if recorder.Code != http.StatusCreated {
		t.Fatalf("create status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	repo := store.snapshot().Repositories[len(store.snapshot().Repositories)-1]
	if repo.Branch != "main" || repo.AgentsPath != "agents" || repo.Status != "启用" {
		t.Fatalf("repo defaults not applied: %#v", repo)
	}

	req := httptest.NewRequest(http.MethodPatch, "/api/repositories/repo-a", strings.NewReader(`{"status":"停用"}`))
	req.SetPathValue("id", "repo-a")
	recorder = httptest.NewRecorder()
	app.patchRepository(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("patch status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodDelete, "/api/repositories/repo-a", nil)
	req.SetPathValue("id", "repo-a")
	recorder = httptest.NewRecorder()
	app.deleteRepository(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
}

func TestBusHTTPHandlers(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	req := httptest.NewRequest(http.MethodPost, "/api/bus/agents/agent-a/register", strings.NewReader(`{"name":"Agent A","status":"online"}`))
	req.SetPathValue("agentId", "agent-a")
	recorder := httptest.NewRecorder()
	app.busRegister(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("register status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/bus/agents/agent-a/message", strings.NewReader(`{"toAgent":"agent-b","content":"hello"}`))
	req.SetPathValue("agentId", "agent-a")
	recorder = httptest.NewRecorder()
	app.busSendMessage(recorder, req)
	if recorder.Code != http.StatusOK {
		t.Fatalf("message status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	app.busMessages(recorder, httptest.NewRequest(http.MethodGet, "/api/bus/messages?toAgent=agent-b", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "hello") {
		t.Fatalf("messages response = %d %s", recorder.Code, recorder.Body.String())
	}

	recorder = httptest.NewRecorder()
	app.busAgents(recorder, httptest.NewRequest(http.MethodGet, "/api/bus/agents", nil))
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "agent-a") {
		t.Fatalf("agents response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestSessionHandlersCreateRenameDelete(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	recorder := httptest.NewRecorder()
	app.agentSessions(recorder, httptest.NewRequest(http.MethodGet, "/api/agents/a1/sessions", nil), "a1")
	if recorder.Code != http.StatusOK {
		t.Fatalf("get status = %d", recorder.Code)
	}
	var sessions []ChatSession
	if err := json.Unmarshal(recorder.Body.Bytes(), &sessions); err != nil {
		t.Fatal(err)
	}
	if len(sessions) != 1 {
		t.Fatalf("expected default session, got %#v", sessions)
	}

	recorder = httptest.NewRecorder()
	app.renameSession(recorder, httptest.NewRequest(http.MethodPost, "/api/agents/a1/sessions/"+sessions[0].ID, strings.NewReader(`{"title":"renamed"}`)), "a1", sessions[0].ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("rename status = %d, body = %s", recorder.Code, recorder.Body.String())
	}

	req := httptest.NewRequest(http.MethodDelete, "/api/agents/a1/sessions/"+sessions[0].ID, nil)
	recorder = httptest.NewRecorder()
	app.deleteSession(recorder, req, "a1", sessions[0].ID)
	if recorder.Code != http.StatusOK {
		t.Fatalf("delete status = %d, body = %s", recorder.Code, recorder.Body.String())
	}
	if got := len(store.snapshot().ChatSessions["a1"]); got != 0 {
		t.Fatalf("session still present, count = %d", got)
	}
}

func TestDeploymentStatusAndStopNotRunning(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()
	_ = store.update(func(s *State) error {
		s.Deployments = []Deployment{{
			ID: "dep-1", AgentID: "agent-a", Runtime: "codex", Status: "stopped",
			Message: "already stopped", SidecarURL: "http://127.0.0.1:18000", CreatedAt: time.Now(),
		}}
		return nil
	})
	app := &App{rootDir: dir, dataDir: dir, store: store, bus: newAgentBus()}

	req := httptest.NewRequest(http.MethodGet, "/api/deployments/dep-1/status", nil)
	req.SetPathValue("id", "dep-1")
	recorder := httptest.NewRecorder()
	app.deploymentStatus(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "already stopped") {
		t.Fatalf("status response = %d %s", recorder.Code, recorder.Body.String())
	}

	req = httptest.NewRequest(http.MethodPost, "/api/deployments/dep-1/stop", nil)
	req.SetPathValue("id", "dep-1")
	recorder = httptest.NewRecorder()
	app.deploymentStop(recorder, req)
	if recorder.Code != http.StatusOK || !strings.Contains(recorder.Body.String(), "not_running") {
		t.Fatalf("stop response = %d %s", recorder.Code, recorder.Body.String())
	}
}

func TestAuthAndCORSMiddleware(t *testing.T) {
	previous := masterToken
	masterToken = ""
	t.Cleanup(func() { masterToken = previous })
	t.Setenv("AGENTBUCKET_ADMIN_TOKEN", "test-master")

	protected := withAuth(nil, http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = w.Write([]byte("ok"))
	}))
	recorder := httptest.NewRecorder()
	protected.ServeHTTP(recorder, httptest.NewRequest(http.MethodGet, "/api/users", nil))
	if recorder.Code != http.StatusUnauthorized {
		t.Fatalf("missing auth status = %d", recorder.Code)
	}
	req := httptest.NewRequest(http.MethodGet, "/api/users", nil)
	req.Header.Set("Authorization", "Bearer test-master")
	recorder = httptest.NewRecorder()
	protected.ServeHTTP(recorder, req)
	if recorder.Code != http.StatusOK || recorder.Body.String() != "ok" {
		t.Fatalf("authorized response = %d %s", recorder.Code, recorder.Body.String())
	}

	cors := withCORS(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusTeapot)
	}))
	recorder = httptest.NewRecorder()
	cors.ServeHTTP(recorder, httptest.NewRequest(http.MethodOptions, "/api/users", nil))
	if recorder.Code != http.StatusNoContent || recorder.Header().Get("Access-Control-Allow-Origin") != "*" {
		t.Fatalf("cors preflight = %d headers=%#v", recorder.Code, recorder.Header())
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

func findAIToken(tokens []AIToken, id int) (AIToken, bool) {
	for _, token := range tokens {
		if token.ID == id {
			return token, true
		}
	}
	return AIToken{}, false
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
