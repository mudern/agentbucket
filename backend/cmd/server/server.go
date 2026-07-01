package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func main() {
	runServer()
}

func runServer() {
	rootDir, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}
	dataDir := os.Getenv("AGENTBUCKET_DATA_DIR")
	if dataDir == "" {
		dataDir = filepath.Join(rootDir, ".data")
	}
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		log.Fatal(err)
	}

	store, err := NewStore(filepath.Join(dataDir, "agentbucket.db"), rootDir)
	if err != nil {
		log.Fatal(err)
	}

	app := &App{
		rootDir: rootDir,
		dataDir: dataDir,
		store:   store,
		bus:     newAgentBus(),
	}

	go app.startHealthChecker(30 * time.Second)

	addr := env("AGENTBUCKET_ADDR", "127.0.0.1:8080")
	log.Printf("AgentBucket backend listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(app.routes())))
}

func (app *App) routes() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", app.health)
	mux.HandleFunc("/api/current-user", app.currentUser)
	mux.HandleFunc("/api/agents", app.agents)
	mux.HandleFunc("/api/agents/", app.agentSubresource)
	mux.HandleFunc("/api/users", app.users)
	mux.HandleFunc("/api/approvals", app.approvals)
	mux.HandleFunc("/api/ai-tokens", app.aiTokens)
	mux.HandleFunc("/api/auth-tokens", app.authTokens)
	mux.HandleFunc("/api/deploy-options", app.deployOptions)
	mux.HandleFunc("/api/deployments", app.deployments)
	mux.HandleFunc("/api/repositories", app.repositories)
	mux.HandleFunc("/api/tokens/resolve", app.resolveToken)
	mux.HandleFunc("POST /api/agent-definitions/scan", app.scanAgentDefinitions)
	mux.HandleFunc("GET /api/deployments/{id}", app.deploymentByID)
	mux.HandleFunc("GET /api/deployments/{id}/status", app.deploymentStatus)
	mux.HandleFunc("POST /api/deployments/{id}/start", app.deploymentStart)
	mux.HandleFunc("POST /api/deployments/{id}/stop", app.deploymentStop)
	mux.HandleFunc("DELETE /api/deployments/{id}", app.deploymentByID)
	mux.HandleFunc("DELETE /api/ai-tokens/{id}", app.deleteAIToken)
	mux.HandleFunc("DELETE /api/auth-tokens/{id}", app.deleteAuthToken)
	mux.HandleFunc("DELETE /api/repositories/{id}", app.deleteRepository)
	mux.HandleFunc("GET /api/bus/agents", app.busAgents)
	mux.HandleFunc("POST /api/bus/agents/{agentId}/register", app.busRegister)
	mux.HandleFunc("POST /api/bus/agents/{agentId}/message", app.busSendMessage)
	mux.HandleFunc("GET /api/bus/messages", app.busMessages)
	return mux
}

func (app *App) startHealthChecker(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		state := app.store.snapshot()
		for _, d := range state.Deployments {
			if d.Status != "running" {
				continue
			}
			out, err := exec.Command("docker", "ps", "--filter", fmt.Sprintf("name=%s", d.ContainerName), "--format", "{{.Status}}").CombinedOutput()
			if err != nil || strings.TrimSpace(string(out)) == "" {
				_ = app.store.update(func(s *State) error {
					for i := range s.Deployments {
						if s.Deployments[i].ID == d.ID {
							s.Deployments[i].Status = "crashed"
							s.Deployments[i].Message = "container exited unexpectedly"
						}
					}
					return nil
				})
				log.Printf("healthcheck: deployment %s (%s) is no longer running", d.ID, d.AgentID)
			}
		}
	}
}
