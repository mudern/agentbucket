package main

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

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

func containsString(values []string, want string) bool {
	for _, value := range values {
		if value == want {
			return true
		}
	}
	return false
}
