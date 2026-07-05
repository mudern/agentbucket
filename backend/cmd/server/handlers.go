package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

func (app *App) stats(w http.ResponseWriter, r *http.Request) {
	state := app.store.snapshot()
	now := time.Now()
	today := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())

	// Token counts
	aiEnabled, aiDisabled := 0, 0
	for _, t := range state.AITokens {
		if t.Status == "启用" {
			aiEnabled++
		} else {
			aiDisabled++
		}
	}
	authEnabled, authDisabled := 0, 0
	for _, t := range state.AuthTokens {
		if t.Status == "启用" {
			authEnabled++
		} else {
			authDisabled++
		}
	}

	// User counts
	superAdmin, admin, user, activeUsers := 0, 0, 0, 0
	for _, u := range state.Users {
		if u.Active {
			activeUsers++
		}
		switch u.Role {
		case "super_admin":
			superAdmin++
		case "admin":
			admin++
		case "user":
			user++
		}
	}

	// Deployment stats
	var deploys []map[string]any
	running, failed, stopped, total := 0, 0, 0, 0
	todayDeploys := 0
	for _, d := range state.Deployments {
		total++
		switch d.Status {
		case "running":
			running++
		case "stopped":
			stopped++
		case "build_failed", "run_failed", "crashed":
			failed++
		}
		if d.CreatedAt.After(today) {
			todayDeploys++
		}
		deploys = append(deploys, map[string]any{
			"agentId": d.AgentID, "status": d.Status, "model": d.Model,
			"runtime": d.Runtime, "createdAt": d.CreatedAt,
		})
	}
	// Recent 10 for success rate
	recent := deploys
	if len(recent) > 10 {
		recent = recent[len(recent)-10:]
	}
	recentSuccess := 0
	for _, d := range recent {
		if d["status"] == "running" {
			recentSuccess++
		}
	}

	// Chat stats
	totalSessions, totalMessages, todayMessages := 0, 0, 0
	for _, sessions := range state.ChatSessions {
		totalSessions += len(sessions)
	}
	for _, msgs := range state.ChatMessages {
		for _, m := range msgs {
			totalMessages++
			if m.CreatedAt.After(today) {
				todayMessages++
			}
		}
	}

	// Repo sync status
	var repoStatus []map[string]any
	for _, r := range app.scanRepositories(state.Repositories) {
		commitCount := len(r.Commits)
		agentCount := 0
		if commitCount > 0 && len(r.Commits[0].Agents) > 0 {
			agentCount = len(r.Commits[0].Agents)
		}
		repoStatus = append(repoStatus, map[string]any{
			"id": r.ID, "provider": r.Provider, "status": r.Status,
			"lastSync": r.LastSync, "commits": commitCount, "agents": agentCount,
		})
	}

	// Bus status
	busAgents := app.bus.list()
	busOnline := 0
	for _, a := range busAgents {
		if a.Status == "online" {
			busOnline++
		}
	}

	// System info
	dockerAvailable := true
	if _, err := exec.LookPath("docker"); err != nil {
		dockerAvailable = false
	}

	writeJSON(w, http.StatusOK, map[string]any{
		"tokens": map[string]any{
			"ai":   map[string]any{"enabled": aiEnabled, "disabled": aiDisabled},
			"auth": map[string]any{"enabled": authEnabled, "disabled": authDisabled},
		},
		"users": map[string]any{
			"total": len(state.Users), "active": activeUsers,
			"superAdmin": superAdmin, "admin": admin, "user": user,
		},
		"deployments": map[string]any{
			"total": total, "running": running, "failed": failed, "stopped": stopped,
			"today": todayDeploys, "recentSuccessRate": recentSuccess,
		},
		"chat": map[string]any{
			"totalSessions": totalSessions, "totalMessages": totalMessages,
			"todayMessages": todayMessages,
		},
		"repositories": repoStatus,
		"bus": map[string]any{
			"total": len(busAgents), "online": busOnline,
		},
		"system": map[string]any{
			"version": "1.0.0", "dockerAvailable": dockerAvailable,
			"goVersion": "1.22+",
		},
	})
}

func (app *App) health(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) currentUser(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, app.store.snapshot().CurrentUser)
}

func (app *App) users(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, app.store.snapshot().Users)
}

func (app *App) approvals(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, app.store.snapshot().Approvals)
	case http.MethodPost:
		var a Approval
		if err := json.NewDecoder(r.Body).Decode(&a); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		_ = app.store.update(func(s *State) error {
			maxID := 0
			for _, existing := range s.Approvals {
				if existing.ID > maxID {
					maxID = existing.ID
				}
			}
			a.ID = maxID + 1
			a.Status = "待审批"
			s.Approvals = append(s.Approvals, a)
			return nil
		})
		writeJSON(w, http.StatusCreated, a)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) approvalAction(w http.ResponseWriter, r *http.Request) {
	idStr := r.PathValue("id")
	action := r.PathValue("action")
	id := 0
	fmt.Sscanf(idStr, "%d", &id)
	var result struct {
		Action string `json:"action"`
	}
	if err := json.NewDecoder(r.Body).Decode(&result); err != nil {
		result.Action = action
	}
	found := false
	_ = app.store.update(func(s *State) error {
		for i := range s.Approvals {
			if s.Approvals[i].ID == id {
				switch result.Action {
				case "approve":
					s.Approvals[i].Status = "已通过"
					s.Approvals[i].Reviewer = "admin"
				case "reject":
					s.Approvals[i].Status = "已拒绝"
					s.Approvals[i].Reviewer = "admin"
				}
				found = true
				return nil
			}
		}
		return nil
	})
	if !found {
		writeError(w, http.StatusNotFound, fmt.Errorf("approval %q not found", idStr))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "action": result.Action})
}

func (app *App) patchUser(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updates map[string]any
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	found := false
	_ = app.store.update(func(s *State) error {
		for i := range s.Users {
			if fmt.Sprintf("%d", s.Users[i].ID) == id {
				if v, ok := updates["role"].(string); ok {
					s.Users[i].Role = v
				}
				if v, ok := updates["active"].(bool); ok {
					s.Users[i].Active = v
				}
				found = true
				return nil
			}
		}
		return nil
	})
	if !found {
		writeError(w, http.StatusNotFound, fmt.Errorf("user %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) aiTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, app.store.snapshot().AITokens)
	case http.MethodPost:
		var token AIToken
		if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if token.Name == "" || token.Provider == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("name and provider are required"))
			return
		}
		if token.Status == "" {
			token.Status = "启用"
		}
		if token.Scope == "" {
			token.Scope = "manual"
		}
		if token.Usage == "" {
			token.Usage = "unused"
		}
		if err := app.store.update(func(state *State) error {
			maxID := 0
			for _, item := range state.AITokens {
				if item.ID > maxID {
					maxID = item.ID
				}
			}
			token.ID = maxID + 1
			state.AITokens = append(state.AITokens, token)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, token)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) authTokens(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		tokens := app.store.snapshot().AuthTokens
		safe := make([]AuthToken, len(tokens))
		for i, t := range tokens {
			safe[i] = t
			safe[i].Secret = ""
		}
		writeJSON(w, http.StatusOK, safe)
	case http.MethodPost:
		var token AuthToken
		if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if token.Name == "" || token.Secret == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("name and secret are required"))
			return
		}
		if token.Status == "" {
			token.Status = "启用"
		}
		if token.UpdatedAt == "" {
			token.UpdatedAt = time.Now().Format(time.RFC3339)
		}
		if err := app.store.update(func(state *State) error {
			maxID := 0
			for _, item := range state.AuthTokens {
				if item.ID > maxID {
					maxID = item.ID
				}
			}
			token.ID = maxID + 1
			state.AuthTokens = append(state.AuthTokens, token)
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, token)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) agents(w http.ResponseWriter, r *http.Request) {
	state := app.store.snapshot()
	seen := map[string]bool{}
	agents := make([]Agent, 0)
	for _, repo := range app.scanRepositories(state.Repositories) {
		if len(repo.Commits) == 0 {
			continue
		}
		for _, agent := range repo.Commits[0].Agents {
			if seen[agent.ID] {
				continue
			}
			seen[agent.ID] = true
			// Only show agents that have a running deployment
			var deployment *Deployment
			for i := range state.Deployments {
				if state.Deployments[i].AgentID == agent.ID && state.Deployments[i].Status == "running" {
					deployment = &state.Deployments[i]
					break
				}
			}
			if deployment == nil {
				continue // skip agents without a running deployment
			}
			agent.Status = "已部署"
			// Override agent definition with deployment-time choices
			if deployment.Model != "" {
				agent.Model = deployment.Model
			}
			if deployment.Runtime != "" {
				agent.Runtime = deployment.Runtime
			}
			if len(deployment.Skills) > 0 {
				agent.Skills = deployment.Skills
			}
			if len(deployment.MCPs) > 0 {
				agent.MCPs = deployment.MCPs
			}
			// Resolve API token name from deployment's apiTokenId
			if deployment.APITokenID > 0 {
				for _, t := range state.AITokens {
					if t.ID == deployment.APITokenID {
						agent.APIToken = t.Name
						break
					}
				}
			}
			agent.Tags = []string{agent.Runtime, agent.Model}
			agent.UpdatedAt = "刚刚"
			agents = append(agents, agent)
		}
	}
	writeJSON(w, http.StatusOK, agents)
}

func (app *App) agentSubresource(w http.ResponseWriter, r *http.Request) {
	parts := strings.Split(strings.TrimPrefix(r.URL.Path, "/api/agents/"), "/")
	if len(parts) < 2 || parts[0] == "" {
		writeError(w, http.StatusNotFound, fmt.Errorf("agent subresource not found"))
		return
	}
	agentID := parts[0]
	resource := parts[1]
	switch resource {
	case "sessions":
		if len(parts) >= 3 && parts[2] != "" && r.Method == http.MethodDelete {
			app.deleteSession(w, r, agentID, parts[2])
			return
		}
		app.agentSessions(w, r, agentID)
	case "messages":
		app.agentMessages(w, r, agentID)
	default:
		writeError(w, http.StatusNotFound, fmt.Errorf("agent subresource not found"))
	}
}

func (app *App) agentSessions(w http.ResponseWriter, r *http.Request, agentID string) {
	switch r.Method {
	case http.MethodGet:
		state := app.store.snapshot()
		sessions := state.ChatSessions[agentID]
		if len(sessions) == 0 {
			sessions = []ChatSession{newChatSession(agentID, "默认会话")}
			_ = app.store.update(func(state *State) error {
				ensureChatMaps(state)
				state.ChatSessions[agentID] = sessions
				return nil
			})
		}
		writeJSON(w, http.StatusOK, sessions)
	case http.MethodPost:
		var req struct {
			Title string `json:"title"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(req.Title) == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("title is required"))
			return
		}
		session := newChatSession(agentID, req.Title)
		if err := app.store.update(func(state *State) error {
			ensureChatMaps(state)
			if len(state.ChatSessions[agentID]) >= 20 {
				return fmt.Errorf("会话数已达上限（20 个），请删除旧会话后重试")
			}
			state.ChatSessions[agentID] = append([]ChatSession{session}, state.ChatSessions[agentID]...)
			return nil
		}); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "已达上限") {
				status = http.StatusForbidden
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusCreated, session)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) agentMessages(w http.ResponseWriter, r *http.Request, agentID string) {
	switch r.Method {
	case http.MethodGet:
		sessionID := r.URL.Query().Get("sessionId")
		state := app.store.snapshot()
		sessions := state.ChatSessions[agentID]
		if sessionID == "" && len(sessions) > 0 {
			sessionID = sessions[0].ID
		}
		messages := state.ChatMessages[chatKey(agentID, sessionID)]
		if messages == nil {
			messages = []ChatMessage{}
		}
		writeJSON(w, http.StatusOK, messages)
	case http.MethodPost:
		var req struct {
			SessionID string `json:"sessionId"`
			Content   string `json:"content"`
			Stream    bool   `json:"stream"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if strings.TrimSpace(req.Content) == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("content is required"))
			return
		}
		sessionID := req.SessionID
		if sessionID == "" {
			sessionID = newChatSession(agentID, "默认会话").ID
		}
		if req.Stream {
			app.streamAgentMessage(w, agentID, req.Content, sessionID)
			return
		}
		now := time.Now()
		userMessage := ChatMessage{
			ID:        fmt.Sprintf("msg-%d-user", now.UnixNano()),
			SessionID: sessionID,
			AgentID:   agentID,
			Role:      "user",
			Content:   req.Content,
			CreatedAt: now,
		}
		assistantMessage := app.buildAssistantMessage(agentID, sessionID, req.Content)
		if err := app.store.update(func(state *State) error {
			ensureChatMaps(state)
			sessions := state.ChatSessions[agentID]
			found := false
			for i := range sessions {
				if sessions[i].ID == sessionID {
					sessions[i].Preview = req.Content
					sessions[i].UpdatedAt = now
					found = true
					break
				}
			}
			if !found {
				if len(sessions) >= 20 {
					return fmt.Errorf("会话数已达上限（20 个），请删除旧会话后重试")
				}
				title := req.Content
				if len([]rune(title)) > 20 {
					title = string([]rune(title)[:20])
				}
				sessions = append([]ChatSession{newChatSession(agentID, title)}, sessions...)
				sessions[0].ID = sessionID
				sessions[0].Preview = req.Content
			}
			state.ChatSessions[agentID] = sessions
			key := chatKey(agentID, sessionID)
			state.ChatMessages[key] = append(state.ChatMessages[key], userMessage, assistantMessage)
			return nil
		}); err != nil {
			status := http.StatusInternalServerError
			if strings.Contains(err.Error(), "已达上限") {
				status = http.StatusForbidden
			}
			writeError(w, status, err)
			return
		}
		writeJSON(w, http.StatusCreated, []ChatMessage{userMessage, assistantMessage})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) deployOptions(w http.ResponseWriter, r *http.Request) {
	state := app.store.snapshot()
	models := []string{}
	seen := map[string]bool{}
	for _, token := range state.AITokens {
		if token.Status == "启用" && token.Model != "" && !seen[token.Model] {
			models = append(models, token.Model)
			seen[token.Model] = true
		}
	}
	writeJSON(w, http.StatusOK, DeployOptions{
		Repositories: app.scanRepositories(state.Repositories),
		Models:       models,
		Runtimes:     supportedRuntimes(),
		RuntimeTags:  []string{"latest", "stable", "nightly"},
		MCPServers:   scanMCPServers(state.Repositories),
		AITokens:     state.AITokens,
		AuthTokens:   state.AuthTokens,
	})
}

func (app *App) repositories(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		writeJSON(w, http.StatusOK, app.scanRepositories(app.store.snapshot().Repositories))
	case http.MethodPost:
		var repo Repository
		if err := json.NewDecoder(r.Body).Decode(&repo); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if repo.ID == "" {
			repo.ID = slug(repo.URL)
		}
		if repo.Branch == "" {
			repo.Branch = "main"
		}
		if repo.AgentsPath == "" {
			repo.AgentsPath = "agents"
		}
		if repo.Status == "" {
			repo.Status = "启用"
		}
		// Auto-clone for GitHub repositories
		// Auto-clone for remote repos (not Local): GitHub, GitLab, etc.
		if repo.Provider != "Local" && repo.URL != "" {
			if repo.LocalPath == "" {
				repo.LocalPath = filepath.Join(app.dataDir, "repos", slug(repo.URL))
			}
			if _, err := os.Stat(filepath.Join(repo.LocalPath, ".git")); os.IsNotExist(err) {
				go func() {
					os.MkdirAll(filepath.Dir(repo.LocalPath), 0o755)
					cmd := exec.Command("git", "clone", "-b", repo.Branch, repo.URL, repo.LocalPath)
					out, err := cmd.CombinedOutput()
					if err != nil {
						log.Printf("git clone failed for %s: %v - %s", repo.ID, err, string(out))
					} else {
						log.Printf("git clone succeeded for %s", repo.ID)
					}
				}()
			}
		}
		if err := app.store.update(func(state *State) error {
			found := false
			for i := range state.Repositories {
				if state.Repositories[i].ID == repo.ID {
					state.Repositories[i] = repo
					found = true
					break
				}
			}
			if !found {
				state.Repositories = append(state.Repositories, repo)
			}
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, repo)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) listBranches(w http.ResponseWriter, r *http.Request) {
	var req struct {
		URL string `json:"url"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.URL == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("url is required"))
		return
	}
	cmd := exec.Command("git", "ls-remote", "--heads", req.URL)
	out, err := cmd.CombinedOutput()
	if err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("failed to list branches: %v — %s", err, string(out)))
		return
	}
	var branches []string
	for _, line := range strings.Split(string(out), "\n") {
		line = strings.TrimSpace(line)
		if line == "" {
			continue
		}
		// Format: <hash>\trefs/heads/<branch>
		parts := strings.Split(line, "\t")
		if len(parts) == 2 {
			branch := strings.TrimPrefix(parts[1], "refs/heads/")
			branches = append(branches, branch)
		}
	}
	if branches == nil {
		branches = []string{}
	}
	writeJSON(w, http.StatusOK, map[string]any{"branches": branches})
}

func (app *App) deployments(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet:
		deployments := app.store.snapshot().Deployments
		if deployments == nil {
			deployments = []Deployment{}
		}
		writeJSON(w, http.StatusOK, deployments)
	case http.MethodPost:
		var req DeployRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		log.Printf("[DEPLOY] REQUEST: agentId=%q model=%q runtime=%q apiTokenId=%d skills=%v mcps=%v",
			req.AgentID, req.Model, req.Runtime, req.APITokenID, req.Skills, req.MCPs)
		deployment, err := app.createDeployment(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := app.store.update(func(state *State) error {
			// Replace existing deployment for the same agent (overwrite)
			replaced := false
			for i := range state.Deployments {
				if state.Deployments[i].AgentID == deployment.AgentID {
					state.Deployments[i] = deployment
					replaced = true
					break
				}
			}
			if !replaced {
				state.Deployments = append([]Deployment{deployment}, state.Deployments...)
			}
			return nil
		}); err != nil {
			writeError(w, http.StatusInternalServerError, err)
			return
		}
		writeJSON(w, http.StatusCreated, deployment)
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) scanAgentDefinitions(w http.ResponseWriter, r *http.Request) {
	state := app.store.snapshot()
	repos := app.scanRepositories(state.Repositories)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "repositories": repos})
}

func (app *App) deploymentByID(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	switch r.Method {
	case http.MethodGet:
		deployments := app.store.snapshot().Deployments
		for i := range deployments {
			if deployments[i].ID == id {
				writeJSON(w, http.StatusOK, deployments[i])
				return
			}
		}
		writeError(w, http.StatusNotFound, fmt.Errorf("deployment %q not found", id))
	case http.MethodDelete:
		var removed Deployment
		if err := app.store.update(func(state *State) error {
			for i, d := range state.Deployments {
				if d.ID == id {
					removed = d
					_ = exec.Command("docker", "rm", "-f", d.ContainerName).Run()
					state.Deployments = append(state.Deployments[:i], state.Deployments[i+1:]...)
					return nil
				}
			}
			return fmt.Errorf("deployment %q not found", id)
		}); err != nil {
			writeError(w, http.StatusNotFound, err)
			return
		}
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "deleted": removed.ID})
	default:
		w.WriteHeader(http.StatusMethodNotAllowed)
	}
}

func (app *App) deploymentsStream(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, fmt.Errorf("streaming not supported"))
		return
	}
	ctx := r.Context()
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			data, _ := json.Marshal(app.store.snapshot().Deployments)
			fmt.Fprintf(w, "data: %s\n\n", data)
			flusher.Flush()
		}
	}
}

func (app *App) deploymentHealth(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	state := app.store.snapshot()
	for i := range state.Deployments {
		if state.Deployments[i].ID == id && state.Deployments[i].SidecarURL != "" {
			resp, err := http.Get(state.Deployments[i].SidecarURL + "/health")
			if err != nil {
				writeJSON(w, http.StatusOK, map[string]any{"ok": false, "error": err.Error()})
				return
			}
			defer resp.Body.Close()
			var result map[string]any
			json.NewDecoder(resp.Body).Decode(&result)
			writeJSON(w, http.StatusOK, result)
			return
		}
	}
	writeError(w, http.StatusNotFound, fmt.Errorf("deployment not found"))
}

func (app *App) deploymentStatus(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var target *Deployment
	state := app.store.snapshot()
	for i := range state.Deployments {
		if state.Deployments[i].ID == id {
			target = &state.Deployments[i]
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("deployment %q not found", id))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"id":         target.ID,
		"agentId":    target.AgentID,
		"runtime":    target.Runtime,
		"status":     target.Status,
		"message":    target.Message,
		"sidecarUrl": target.SidecarURL,
	})
}

func (app *App) deploymentRedeploy(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	state := app.store.snapshot()
	var target *Deployment
	for i := range state.Deployments {
		if state.Deployments[i].ID == id {
			target = &state.Deployments[i]
			break
		}
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("deployment not found"))
		return
	}
	req := DeployRequest{
		RepositoryID:   target.RepositoryID,
		CommitHash:     target.CommitHash,
		AgentID:        target.AgentID,
		APITokenID:     target.APITokenID,
		Model:          target.Model,
		Runtime:        target.Runtime,
		RuntimeVersion: target.RuntimeVersion,
		Skills:         target.Skills,
		MCPs:           target.MCPs,
		AuthTokens:     target.AuthTokens,
	}
	deployment, err := app.createDeployment(req)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	// Replace old deployment
	_ = app.store.update(func(s *State) error {
		for i := range s.Deployments {
			if s.Deployments[i].ID == id {
				s.Deployments[i] = deployment
				return nil
			}
		}
		s.Deployments = append([]Deployment{deployment}, s.Deployments...)
		return nil
	})
	writeJSON(w, http.StatusCreated, deployment)
}

func (app *App) deploymentStart(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var target *Deployment
	idx := -1
	if err := app.store.update(func(state *State) error {
		for i := range state.Deployments {
			if state.Deployments[i].ID == id {
				target = &state.Deployments[i]
				idx = i
				break
			}
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("deployment %q not found", id))
		return
	}
	if target.Status == "running" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "already_running"})
		return
	}
	if _, err := exec.LookPath("docker"); err != nil {
		writeError(w, http.StatusBadRequest, fmt.Errorf("docker not found"))
		return
	}
	_ = exec.Command("docker", "rm", "-f", target.ContainerName).Run()
	run := exec.Command("docker", app.dockerRunArgs(*target)...)
	out, err := run.CombinedOutput()
	if err != nil {
		if err2 := app.store.update(func(state *State) error {
			deployments := state.Deployments
			for i := range deployments {
				if deployments[i].ID == id {
					deployments[i].Status = "run_failed"
					deployments[i].Message = string(out)
				}
			}
			state.Deployments = deployments
			return nil
		}); err2 != nil {
			log.Printf("failed to update deployment status: %v", err2)
		}
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	_ = app.store.update(func(state *State) error {
		if idx < len(state.Deployments) && state.Deployments[idx].ID == id {
			state.Deployments[idx].Status = "running"
			state.Deployments[idx].Message = strings.TrimSpace(string(out))
		}
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "running"})
}

func (app *App) deploymentStop(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var target *Deployment
	idx := -1
	if err := app.store.update(func(state *State) error {
		for i := range state.Deployments {
			if state.Deployments[i].ID == id {
				target = &state.Deployments[i]
				idx = i
				break
			}
		}
		return nil
	}); err != nil {
		writeError(w, http.StatusInternalServerError, err)
		return
	}
	if target == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("deployment %q not found", id))
		return
	}
	if target.Status != "running" {
		writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "not_running"})
		return
	}
	_ = exec.Command("docker", "stop", target.ContainerName).Run()
	_ = app.store.update(func(state *State) error {
		if idx < len(state.Deployments) && state.Deployments[idx].ID == id {
			state.Deployments[idx].Status = "stopped"
			state.Deployments[idx].Message = "container stopped"
		}
		return nil
	})
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "status": "stopped"})
}

func (app *App) renameSession(w http.ResponseWriter, r *http.Request, agentID string, sessionID string) {
	var req struct {
		Title string `json:"title"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Title == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("title is required"))
		return
	}
	found := false
	_ = app.store.update(func(state *State) error {
		sessions := state.ChatSessions[agentID]
		for i := range sessions {
			if sessions[i].ID == sessionID {
				sessions[i].Title = req.Title
				found = true
				break
			}
		}
		return nil
	})
	if !found {
		writeError(w, http.StatusNotFound, fmt.Errorf("session not found"))
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) deleteSession(w http.ResponseWriter, r *http.Request, agentID string, sessionID string) {
	if r.Method != http.MethodDelete {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	if err := app.store.update(func(state *State) error {
		ensureChatMaps(state)
		sessions := state.ChatSessions[agentID]
		for i, s := range sessions {
			if s.ID == sessionID {
				state.ChatSessions[agentID] = append(sessions[:i], sessions[i+1:]...)
				delete(state.ChatMessages, chatKey(agentID, sessionID))
				return nil
			}
		}
		return fmt.Errorf("session %q not found", sessionID)
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) deleteAIToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := app.store.update(func(state *State) error {
		for i, t := range state.AITokens {
			if fmt.Sprintf("%d", t.ID) == id {
				state.AITokens = append(state.AITokens[:i], state.AITokens[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("token not found")
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) deleteAuthToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := app.store.update(func(state *State) error {
		for i, t := range state.AuthTokens {
			if fmt.Sprintf("%d", t.ID) == id {
				state.AuthTokens = append(state.AuthTokens[:i], state.AuthTokens[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("token not found")
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) deleteRepository(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	if err := app.store.update(func(state *State) error {
		for i, repo := range state.Repositories {
			if repo.ID == id {
				state.Repositories = append(state.Repositories[:i], state.Repositories[i+1:]...)
				return nil
			}
		}
		return fmt.Errorf("repository %q not found", id)
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) patchRepository(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := app.store.update(func(state *State) error {
		for i := range state.Repositories {
			if state.Repositories[i].ID == id {
				if v, ok := updates["status"]; ok {
					state.Repositories[i].Status = v
				}
				return nil
			}
		}
		return fmt.Errorf("repository %q not found", id)
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) patchAIToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := app.store.update(func(state *State) error {
		for i := range state.AITokens {
			if fmt.Sprintf("%d", state.AITokens[i].ID) == id {
				if v, ok := updates["status"]; ok {
					state.AITokens[i].Status = v
				}
				return nil
			}
		}
		return fmt.Errorf("token not found")
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}

func (app *App) patchAuthToken(w http.ResponseWriter, r *http.Request) {
	id := r.PathValue("id")
	var updates map[string]string
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if err := app.store.update(func(state *State) error {
		for i := range state.AuthTokens {
			if fmt.Sprintf("%d", state.AuthTokens[i].ID) == id {
				if v, ok := updates["status"]; ok {
					state.AuthTokens[i].Status = v
				}
				return nil
			}
		}
		return fmt.Errorf("token not found")
	}); err != nil {
		writeError(w, http.StatusNotFound, err)
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"ok": true})
}
