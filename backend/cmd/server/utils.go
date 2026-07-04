package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
)

var masterToken string

func getMasterToken() string {
	if masterToken == "" {
		masterToken = env("AGENTBUCKET_ADMIN_TOKEN", fmt.Sprintf("ab-admin-%d", time.Now().Unix()))
	}
	return masterToken
}

func withAuth(store *Store, next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Skip auth for health check, current user, deploy options, login, agents (read-only UI)
		path := r.URL.Path
		if path == "/health" || path == "/api/deploy-options" || path == "/api/login" || path == "/api/current-user" || path == "/api/agents" || path == "/api/deployments" || strings.HasPrefix(path, "/api/deployments/") {
			next.ServeHTTP(w, r)
			return
		}
		// Skip OPTIONS preflight
		if r.Method == http.MethodOptions {
			next.ServeHTTP(w, r)
			return
		}
		token := r.Header.Get("Authorization")
		if strings.HasPrefix(token, "Bearer ") {
			token = strings.TrimPrefix(token, "Bearer ")
		} else {
			token = r.Header.Get("X-API-Key")
		}
		if token == "" {
			writeError(w, http.StatusUnauthorized, fmt.Errorf("missing authorization"))
			return
		}
		if token == getMasterToken() {
			next.ServeHTTP(w, r)
			return
		}
		// Check against users table
		if store != nil && store.db != nil {
			var count int
			if err := store.db.QueryRow(`SELECT COUNT(*) FROM users WHERE token = ? AND active = 1`, token).Scan(&count); err == nil && count > 0 {
				next.ServeHTTP(w, r)
				return
			}
		}
		writeError(w, http.StatusForbidden, fmt.Errorf("invalid token"))
	})
}

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

