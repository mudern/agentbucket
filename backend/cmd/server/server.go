package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"os/signal"
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

	app.recoverRunningContainers()
	go app.startHealthChecker(30 * time.Second)
	go app.startGitSyncer(5 * time.Minute)

	addr := env("AGENTBUCKET_ADDR", "127.0.0.1:8080")

	// Prune bus_messages periodically
	go func() {
		for {
			time.Sleep(1 * time.Hour)
			if app.store.db != nil {
				_, _ = app.store.db.Exec(`DELETE FROM bus_messages WHERE id NOT IN (SELECT id FROM (SELECT id FROM bus_messages ORDER BY created_at DESC LIMIT 1000))`)
			}
		}
	}()

	// Graceful shutdown
	server := &http.Server{Addr: addr, Handler: withCORS(withAuth(app.store, app.routes()))}
	go func() {
		sigCh := make(chan os.Signal, 1)
		signal.Notify(sigCh, os.Interrupt)
		<-sigCh
		log.Println("shutting down...")
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		server.Shutdown(shutdownCtx)
		// Stop running containers
		state := app.store.snapshot()
		for _, d := range state.Deployments {
			if d.Status == "running" {
				exec.Command("docker", "stop", d.ContainerName).Run()
				log.Printf("stopped container: %s", d.ContainerName)
			}
		}
		os.Exit(0)
	}()

	log.Printf("AgentBucket backend listening on http://%s", addr)
	log.Fatal(server.ListenAndServe())
}

func (app *App) routes() http.Handler {
	mux := http.NewServeMux()

	// Serve frontend static files when dist/ directory exists (Docker / single-binary mode)
	distDir := filepath.Join(app.rootDir, "dist")
	hasDist := dirExists(distDir)
	mux.HandleFunc("/health", app.health)
	mux.HandleFunc("POST /api/login", app.login)
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
	mux.HandleFunc("POST /api/repositories/{id}/sync", app.syncRepository)
	mux.HandleFunc("PATCH /api/repositories/{id}", app.patchRepository)
	mux.HandleFunc("PATCH /api/ai-tokens/{id}", app.patchAIToken)
	mux.HandleFunc("PATCH /api/auth-tokens/{id}", app.patchAuthToken)
	mux.HandleFunc("GET /api/bus/agents", app.busAgents)
	mux.HandleFunc("POST /api/bus/agents/{agentId}/register", app.busRegister)
	mux.HandleFunc("POST /api/bus/agents/{agentId}/message", app.busSendMessage)
	mux.HandleFunc("GET /api/bus/messages", app.busMessages)
	mux.HandleFunc("PATCH /api/users/{id}", app.patchUser)
	mux.HandleFunc("POST /api/approvals/{id}/{action}", app.approvalAction)
	if hasDist {
		fs := http.FileServer(http.Dir(distDir))
		mux.HandleFunc("GET /", func(w http.ResponseWriter, r *http.Request) {
			path := r.URL.Path
			if filepath.Ext(path) != "" {
				fs.ServeHTTP(w, r)
			} else {
				http.ServeFile(w, r, filepath.Join(distDir, "index.html"))
			}
		})
	}
	return mux
}

func (app *App) recoverRunningContainers() {
	out, err := exec.Command("docker", "ps", "--filter", "name=agentbucket-", "--format", "{{.Names}}\t{{.Image}}\t{{.Ports}}").CombinedOutput()
	if err != nil {
		return
	}
	state := app.store.snapshot()
	existing := map[string]bool{}
	for _, d := range state.Deployments {
		existing[d.ContainerName] = true
	}
	recovered := 0
	for _, line := range strings.Split(strings.TrimSpace(string(out)), "\n") {
		if line == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		if len(parts) < 3 {
			continue
		}
		containerName := parts[0]
		image := parts[1]
		if existing[containerName] {
			continue
		}
		// Extract agent ID from container name: agentbucket-{slug}
		agentSlug := strings.TrimPrefix(containerName, "agentbucket-")
		// Extract port from "0.0.0.0:PORT->8088/tcp" format
		port := 0
		portStr := parts[2]
		if idx := strings.Index(portStr, "->"); idx > 0 {
			start := strings.LastIndex(portStr[:idx], ":")
			if start >= 0 {
				fmt.Sscanf(portStr[start+1:idx], "%d", &port)
			}
		}
		if port == 0 {
			port = hostPortFor(agentSlug)
		}
		// Match agent from scanned repos
		repos := app.scanRepositories(app.store.snapshot().Repositories)
		var agentID string
		for _, repo := range repos {
			for _, commit := range repo.Commits {
				for _, agent := range commit.Agents {
					if slug(agent.ID) == agentSlug {
						agentID = agent.ID
						break
					}
				}
			}
		}
		if agentID == "" {
			continue
		}
		d := Deployment{
			ID:            fmt.Sprintf("dep-%s-recovered", agentID),
			AgentID:       agentID,
			ImageTag:      image,
			ContainerName: containerName,
			Status:        "running",
			HostPort:      port,
			SidecarURL:    fmt.Sprintf("http://%s:%d", sidecarHost(), port),
			CreatedAt:     time.Now(),
		}
		_ = app.store.update(func(s *State) error {
			s.Deployments = append(s.Deployments, d)
			return nil
		})
		recovered++
		log.Printf("recovered container: %s -> %s (port %d)", containerName, agentID, port)
	}
	if recovered > 0 {
		log.Printf("recovered %d running containers", recovered)
	}
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

func (app *App) startGitSyncer(interval time.Duration) {
	// Run once immediately
	app.syncAllGitRepos()
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for range ticker.C {
		app.syncAllGitRepos()
	}
}

func (app *App) syncAllGitRepos() {
	state := app.store.snapshot()
	for _, repo := range state.Repositories {
		if repo.Provider != "GitHub" {
			continue
		}
		app.syncGitRepo(&repo)
	}
}

func (app *App) syncGitRepo(repo *Repository) {
	gitDir := filepath.Join(repo.LocalPath, ".git")
	if _, err := os.Stat(gitDir); os.IsNotExist(err) {
		return
	}
	cmd := exec.Command("git", "-C", repo.LocalPath, "pull", "origin", repo.Branch)
	out, err := cmd.CombinedOutput()
	now := time.Now().Format(time.RFC3339)
	if err != nil {
		log.Printf("git sync failed for %s: %v - %s", repo.ID, err, strings.TrimSpace(string(out)))
		_ = app.store.update(func(s *State) error {
			for i := range s.Repositories {
				if s.Repositories[i].ID == repo.ID {
					s.Repositories[i].LastSync = now + " (failed)"
				}
			}
			return nil
		})
		return
	}
	_ = app.store.update(func(s *State) error {
		for i := range s.Repositories {
			if s.Repositories[i].ID == repo.ID {
				s.Repositories[i].LastSync = now
			}
		}
		return nil
	})
	log.Printf("git sync: %s updated", repo.ID)
}

func dirExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && info.IsDir()
}

func (app *App) syncRepository(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	state := app.store.snapshot()
	var target *Repository
	for i := range state.Repositories {
		if state.Repositories[i].ID == id {
			target = &state.Repositories[i]
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("repository %q not found", id))
		return
	}
	app.syncGitRepo(target)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "lastSync": time.Now().Format(time.RFC3339)})
}
