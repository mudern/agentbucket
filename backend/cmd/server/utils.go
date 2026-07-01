package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]any{"error": err.Error()})
}

func env(key, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return fallback
}

func buildTimeout() time.Duration {
	value := env("AGENTBUCKET_BUILD_TIMEOUT", "180s")
	duration, err := time.ParseDuration(value)
	if err != nil {
		return 180 * time.Second
	}
	return duration
}

func executeTokenScript(rootDir string, token *AuthToken, param string) (string, error) {
	if token.Script == "" {
		return "", fmt.Errorf("no script configured")
	}
	scriptPath := filepath.Join(rootDir, token.Script)
	if _, err := os.Stat(scriptPath); err != nil {
		return "", fmt.Errorf("script not found: %s", token.Script)
	}
	cmd := exec.Command("python3", scriptPath, param)
	cmd.Dir = rootDir
	out, err := cmd.CombinedOutput()
	if err != nil {
		return "", fmt.Errorf("script error: %v: %s", err, strings.TrimSpace(string(out)))
	}
	result := strings.TrimSpace(string(out))
	if result == "" {
		return "", fmt.Errorf("script returned empty output")
	}
	return result, nil
}
