package main

import (
	"encoding/json"
	"fmt"
	"net/http"
)

func (app *App) resolveToken(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	agentID := r.Header.Get("X-Agent-ID")
	if agentID == "" {
		writeError(w, http.StatusUnauthorized, fmt.Errorf("missing X-Agent-ID"))
		return
	}
	var req TokenResolveRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	state := app.store.snapshot()
	var token *AuthToken
	for i := range state.AuthTokens {
		if state.AuthTokens[i].ID == req.TokenID {
			token = &state.AuthTokens[i]
			break
		}
	}
	if token == nil {
		writeError(w, http.StatusNotFound, fmt.Errorf("token not found"))
		return
	}
	if token.Status != "启用" {
		writeError(w, http.StatusForbidden, fmt.Errorf("token is disabled"))
		return
	}
	allowed := false
	for _, deployment := range state.Deployments {
		if deployment.AgentID != agentID || deployment.Status != "running" {
			continue
		}
		for _, tokenID := range deployment.AuthTokens {
			if tokenID == req.TokenID {
				allowed = true
				break
			}
		}
	}
	if !allowed {
		writeError(w, http.StatusForbidden, fmt.Errorf("agent %s is not allowed to access token %d", agentID, req.TokenID))
		return
	}
	resolved, err := executeTokenScript(app.rootDir, token, req.Param)
	if err != nil {
		resolved = fmt.Sprintf("test-token-%d-%s-%s (script error: %v)", token.ID, slug(agentID), shortHash(req.Param), err)
	}
	writeJSON(w, http.StatusOK, map[string]any{
		"tokenId":      token.ID,
		"name":         token.Name,
		"accessTarget": token.AccessTarget,
		"param":        req.Param,
		"token":        resolved,
	})
}
