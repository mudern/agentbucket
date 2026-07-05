package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"time"
)

func ensureChatMaps(state *State) {
	if state.ChatSessions == nil {
		state.ChatSessions = map[string][]ChatSession{}
	}
	if state.ChatMessages == nil {
		state.ChatMessages = map[string][]ChatMessage{}
	}
}

func newChatSession(agentID string, title string) ChatSession {
	now := time.Now()
	id := "session-" + shortHash(agentID+"-"+title+"-"+now.Format(time.RFC3339Nano))
	return ChatSession{
		ID:        id,
		AgentID:   agentID,
		Title:     title,
		Preview:   "",
		CreatedAt: now,
		UpdatedAt: now,
	}
}

func chatKey(agentID string, sessionID string) string {
	return agentID + "::" + sessionID
}

// resolveAgentConfig merges deployment overrides with agent definition defaults.
// Deploy-time choices (token, model, runtime) take precedence over agent.toml values.
func (app *App) resolveAgentConfig(agent Agent, tokens []AIToken, deployments []Deployment) (tokenName string, model string, runtime string) {
	tokenName = agent.APIToken
	model = agent.Model
	runtime = agent.Runtime
	// If there's a deployment for this agent, use its configuration
	for _, d := range deployments {
		if d.AgentID == agent.ID {
			if d.Runtime != "" {
				runtime = d.Runtime
			}
			if d.Model != "" {
				model = d.Model
			}
			// Resolve apiTokenId to token name
			if d.APITokenID > 0 {
				for _, t := range tokens {
					if t.ID == d.APITokenID {
						tokenName = t.Name
						model = t.Model // Token's model takes highest priority
						break
					}
				}
			}
			break
		}
	}
	return
}

func (app *App) buildAssistantMessage(agentID string, sessionID string, userContent string) ChatMessage {
	now := time.Now()
	state := app.store.snapshot()

	var agent Agent
	found := false
	for _, repo := range app.scanRepositories(state.Repositories) {
		if len(repo.Commits) == 0 {
			continue
		}
		for _, candidate := range repo.Commits[0].Agents {
			if candidate.ID == agentID {
				agent = candidate
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		return ChatMessage{
			ID: fmt.Sprintf("msg-%d-assistant", now.UnixNano()), SessionID: sessionID, AgentID: agentID,
			Role: "assistant", Content: "未找到 Agent 定义。", CreatedAt: now,
		}
	}

	// Merge deployment overrides: user-chosen token/model/runtime take precedence over agent.toml
	resolvedToken, resolvedModel, resolvedRuntime := app.resolveAgentConfig(agent, state.AITokens, state.Deployments)
	agent.APIToken = resolvedToken
	agent.Model = resolvedModel
	agent.Runtime = resolvedRuntime

	// Auto-register on the bus so other agents can discover this one
	app.bus.register(BusAgent{
		AgentID:  agentID,
		Name:     agent.Name,
		Status:   "online",
		Endpoint: fmt.Sprintf("http://127.0.0.1:%d", hostPortFor(agentID)),
	})

	// Check for running sidecar deployment first
	var sidecarURL string
	for i := range state.Deployments {
		if state.Deployments[i].AgentID == agentID && state.Deployments[i].Status == "running" {
			sidecarURL = state.Deployments[i].SidecarURL
			break
		}
	}

	// If sidecar is running, try to route through it
	// Try runtime CLI on host first (for local chat bound to runtime)
	if content, ok := callRuntimeCLI(agent, userContent, state.AITokens); ok {
		return ChatMessage{
			ID: fmt.Sprintf("msg-%d-assistant", now.UnixNano()), SessionID: sessionID, AgentID: agentID,
			Role: "assistant", Content: content, CreatedAt: now,
		}
	}

	// Check for running sidecar deployment
	if sidecarURL != "" {
		if content, ok := callSidecarChat(sidecarURL, userContent); ok {
			return ChatMessage{
				ID: fmt.Sprintf("msg-%d-assistant", now.UnixNano()), SessionID: sessionID, AgentID: agentID,
				Role: "assistant", Content: content, CreatedAt: now,
			}
		}
	}

	// Fallback: call AI API directly
	content := callAIAPI(app, agent, userContent, state.AITokens)
	return ChatMessage{
		ID: fmt.Sprintf("msg-%d-assistant", now.UnixNano()), SessionID: sessionID, AgentID: agentID,
		Role: "assistant", Content: content, CreatedAt: now,
	}
}

func callRuntimeCLI(agent Agent, userContent string, tokens []AIToken) (string, bool) {
	var baseURL, authToken, model string
	tokenName := agent.APIToken
	for _, t := range tokens {
		if t.Name == tokenName && t.Status == "启用" {
			baseURL = t.BaseURL
			authToken = t.Secret
			model = t.Model
			break
		}
	}
	if authToken == "" {
		return "", false
	}
	if model == "" {
		model = agent.Model
	}

	var exe string
	switch agent.Runtime {
	case "claudecode":
		exe = "claude"
	case "codex":
		exe = "codex"
	case "opencode":
		exe = "opencode"
	case "gemini":
		exe = "gemini"
	case "reasonix":
		exe = "reasonix"
	default:
		return "", false
	}

	if _, err := exec.LookPath(exe); err != nil {
		return "", false // CLI not installed, fall through to API
	}

	var cmd *exec.Cmd
	switch agent.Runtime {
	case "claudecode":
		cmd = exec.Command("claude", "-p", userContent)
	case "codex":
		cmd = exec.Command("codex", "exec", "--model", model, userContent)
	case "opencode":
		cmd = exec.Command("opencode", "run", "--model", model, userContent)
	case "gemini":
		cmd = exec.Command("gemini", "-m", model, "-p", userContent)
	case "reasonix":
		cmd = exec.Command("reasonix", "run", "--model", model, userContent)
	}
	cmd.Env = append(os.Environ(),
		"AGENTBUCKET_AI_TOKEN="+authToken,
		"AGENTBUCKET_AI_BASE_URL="+baseURL,
		"AGENTBUCKET_AI_MODEL="+model,
		"ANTHROPIC_AUTH_TOKEN="+authToken,
		"ANTHROPIC_BASE_URL="+baseURL,
		"ANTHROPIC_MODEL="+model,
		"OPENAI_API_KEY="+authToken,
		"OPENAI_BASE_URL="+baseURL,
		"GEMINI_API_KEY="+authToken,
		"GOOGLE_API_KEY="+authToken,
		"CLAUDE_CODE_DISABLE_NONESSENTIAL_TRAFFIC=1",
	)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	done := make(chan error, 1)
	go func() { done <- cmd.Run() }()

	select {
	case err := <-done:
		if err != nil {
			output := stderr.String()
			if output == "" {
				output = stdout.String()
			}
			if output == "" {
				return "", false
			}
			return output, true
		}
		return stdout.String(), true
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		return "", false
	}
}

func (app *App) streamSidecarChat(w http.ResponseWriter, flusher http.Flusher, sidecarURL string, userContent string, agentID string, sessionID string, agent Agent) bool {
	body, _ := json.Marshal(map[string]any{"message": userContent, "stream": true})
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sidecarURL+"/agent/chat", bytes.NewReader(body))
	if err != nil {
		return false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return false
	}

	var fullContent strings.Builder
	if err := scanSSEData(resp.Body, func(data string) {
		if data == "[DONE]" || strings.HasPrefix(data, "[error]") {
			return
		}
		fullContent.WriteString(strings.ReplaceAll(data, "\\n", "\n"))
		_, _ = w.Write([]byte("data: " + data + "\n\n"))
		flusher.Flush()
	}); err != nil {
		return false
	}

	content := fullContent.String()
	if content == "" {
		return false
	}

	// Persist to DB
	now := time.Now()
	userMsg := ChatMessage{
		ID: fmt.Sprintf("msg-%d-user", now.UnixNano()), SessionID: sessionID, AgentID: agentID,
		Role: "user", Content: userContent, CreatedAt: now,
	}
	assistantMsg := ChatMessage{
		ID: fmt.Sprintf("msg-%d-assistant", now.UnixNano()+1), SessionID: sessionID, AgentID: agentID,
		Role: "assistant", Content: content, CreatedAt: now,
	}
	_ = app.store.update(func(state *State) error {
		ensureChatMaps(state)
		sessions := state.ChatSessions[agentID]
		found := false
		for i := range sessions {
			if sessions[i].ID == sessionID {
				sessions[i].Preview = firstRunes(userContent, 50)
				sessions[i].UpdatedAt = now
				found = true
				break
			}
		}
		if !found {
			s := newChatSession(agentID, firstRunes(userContent, 20))
			s.ID = sessionID
			state.ChatSessions[agentID] = append(state.ChatSessions[agentID], s)
		}
		key := chatKey(agentID, sessionID)
		state.ChatMessages[key] = append(state.ChatMessages[key], userMsg, assistantMsg)
		return nil
	})

	// Send complete event
	w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
	return true
}

func callSidecarChat(sidecarURL string, userContent string) (string, bool) {
	body, _ := json.Marshal(map[string]string{"message": userContent})
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, sidecarURL+"/agent/chat", bytes.NewReader(body))
	if err != nil {
		return "", false
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return "", false
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", false
	}
	var result map[string]string
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return "", false
	}
	content, ok := result["content"]
	return content, ok && content != ""
}

func callAIAPI(app *App, agent Agent, userContent string, tokens []AIToken) string {
	tokenName := agent.APIToken
	if tokenName == "" {
		// Try to find first enabled token
		for _, t := range tokens {
			if t.Status == "启用" {
				tokenName = t.Name
				break
			}
		}
	}

	var baseURL, authToken, model string
	for _, t := range tokens {
		if t.Name == tokenName && t.Status == "启用" {
			baseURL = strings.TrimRight(t.BaseURL, "/")
			authToken = t.Secret
			model = t.Model
			break
		}
	}
	if baseURL == "" || authToken == "" {
		return fmt.Sprintf("未找到可用的 AI token %q，请先在 AI Tokens 页面添加。", tokenName)
	}

	if model == "" {
		model = agent.Model
	}
	if model == "" {
		model = "claude-sonnet-4-20250514"
	}

	// Build rich context prompt
	ctxPrompt := fmt.Sprintf("你是 AgentBucket 平台上的 Agent「%s」(ID: %s)，运行在 %s runtime 上。\n\n", agent.Name, agent.ID, agent.Runtime)
	if len(agent.Skills) > 0 {
		ctxPrompt += fmt.Sprintf("可用技能 Skills: %s\n", strings.Join(agent.Skills, ", "))
	}
	if len(agent.MCPs) > 0 {
		ctxPrompt += fmt.Sprintf("MCP 配置: %s\n", strings.Join(agent.MCPs, ", "))
	}
	ctxPrompt += "\n== AgentBucket 总线当前在线 Agent ==\n"
	busAgents := app.bus.list()
	if len(busAgents) == 0 {
		ctxPrompt += "（总线上暂无其他 Agent）\n"
	} else {
		for _, ba := range busAgents {
			if ba.AgentID == agent.ID {
				continue
			}
			ctxPrompt += fmt.Sprintf("- %s (ID: %s, 状态: %s)\n", ba.Name, ba.AgentID, ba.Status)
		}
		ctxPrompt += "\n你可以让用户帮你传达消息给其他 Agent。发送消息格式：\n"
		ctxPrompt += "POST /api/bus/agents/" + agent.ID + "/message  {\"toAgent\":\"target-id\",\"content\":\"消息内容\"}\n"
	}
	ctxPrompt += "\n查看发给你的消息: GET /api/bus/messages?toAgent=" + agent.ID + "\n"
	ctxPrompt += "\n如果你需要用户做出选择或确认，使用格式：\n[QUESTION:问题描述|选项A|选项B]\n"
	ctxPrompt += "用户会看到按钮并点击回复。仅在需要时使用此格式。\n"
	ctxPrompt += "\n现在回答问题：\n" + userContent

	msgBody := map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"messages": []map[string]string{
			{"role": "user", "content": ctxPrompt},
		},
	}
	payload, _ := json.Marshal(msgBody)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		return fmt.Sprintf("API 请求创建失败：%v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("x-api-key", authToken)
	req.Header.Set("anthropic-version", "2023-06-01")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return fmt.Sprintf("AI API 请求失败：%v", err)
	}
	defer resp.Body.Close()

	raw, _ := io.ReadAll(resp.Body)
	if resp.StatusCode != http.StatusOK {
		return fmt.Sprintf("AI API 返回错误 (HTTP %d)：%s", resp.StatusCode, string(raw[:min(500, len(raw))]))
	}

	var result struct {
		Content []struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(raw, &result); err != nil {
		return fmt.Sprintf("API 响应解析失败：%v\n原始响应：%s", err, string(raw[:min(300, len(raw))]))
	}

	for _, item := range result.Content {
		if item.Type == "text" && item.Text != "" {
			return item.Text
		}
	}

	return "AI 返回了空响应。"
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func (app *App) streamAgentMessage(w http.ResponseWriter, agentID string, userContent string, sessionID string) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		sseError(w, "streaming not supported")
		return
	}

	state := app.store.snapshot()
	var agent Agent
	found := false
	for _, repo := range app.scanRepositories(state.Repositories) {
		if len(repo.Commits) == 0 {
			continue
		}
		for _, candidate := range repo.Commits[0].Agents {
			if candidate.ID == agentID {
				agent = candidate
				found = true
				break
			}
		}
		if found {
			break
		}
	}
	if !found {
		sseError(w, "agent not found")
		return
	}

	// Merge deployment overrides: user-chosen token/model/runtime take precedence
	resolvedToken, resolvedModel, resolvedRuntime := app.resolveAgentConfig(agent, state.AITokens, state.Deployments)
	agent.APIToken = resolvedToken
	agent.Model = resolvedModel
	agent.Runtime = resolvedRuntime

	// Try streaming through sidecar if agent has a running deployment
	for _, d := range state.Deployments {
		if d.AgentID == agentID && d.Status == "running" && d.SidecarURL != "" {
			if app.streamSidecarChat(w, flusher, d.SidecarURL, userContent, agentID, sessionID, agent) {
				return
			}
			// Sidecar failed, fall through to direct AI API
			break
		}
	}

	// Auto-register on bus
	app.bus.register(BusAgent{
		AgentID:  agentID,
		Name:     agent.Name,
		Status:   "online",
		Endpoint: fmt.Sprintf("http://127.0.0.1:%d", hostPortFor(agentID)),
	})

	tokenName := agent.APIToken
	if tokenName == "" {
		for _, t := range state.AITokens {
			if t.Status == "启用" {
				tokenName = t.Name
				break
			}
		}
	}

	var baseURL, authToken, model string
	for _, t := range state.AITokens {
		if t.Name == tokenName && t.Status == "启用" {
			baseURL = strings.TrimRight(t.BaseURL, "/")
			authToken = t.Secret
			model = t.Model
			break
		}
	}
	if baseURL == "" || authToken == "" {
		sseError(w, fmt.Sprintf("未找到可用的 AI token %q", tokenName))
		return
	}
	if model == "" {
		model = agent.Model
	}

	// Build context prompt
	ctxPrompt := fmt.Sprintf("你是 AgentBucket 平台上的 Agent「%s」(ID: %s)，运行在 %s runtime 上。\n\n", agent.Name, agent.ID, agent.Runtime)
	if len(agent.Skills) > 0 {
		ctxPrompt += fmt.Sprintf("可用技能 Skills: %s\n", strings.Join(agent.Skills, ", "))
	}
	if len(agent.MCPs) > 0 {
		ctxPrompt += fmt.Sprintf("MCP 配置: %s\n", strings.Join(agent.MCPs, ", "))
	}
	ctxPrompt += "\n== AgentBucket 总线当前在线 Agent ==\n"
	busAgents := app.bus.list()
	if len(busAgents) == 0 {
		ctxPrompt += "（总线上暂无其他 Agent）\n"
	} else {
		for _, ba := range busAgents {
			if ba.AgentID == agent.ID {
				continue
			}
			ctxPrompt += fmt.Sprintf("- %s (ID: %s, 状态: %s)\n", ba.Name, ba.AgentID, ba.Status)
		}
	}
	ctxPrompt += "\n如果你需要用户做出选择或确认，使用格式：[QUESTION:问题描述|选项A|选项B]\n"
	ctxPrompt += "用户会看到按钮并点击回复。仅在需要时使用此格式。\n\n现在回答问题：\n" + userContent

	msgBody := map[string]any{
		"model":      model,
		"max_tokens": 4096,
		"stream":     true,
		"messages": []map[string]string{
			{"role": "user", "content": ctxPrompt},
		},
	}
	payload, _ := json.Marshal(msgBody)

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, baseURL+"/v1/messages", bytes.NewReader(payload))
	if err != nil {
		sseError(w, err.Error())
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+authToken)
	req.Header.Set("x-api-key", authToken)
	req.Header.Set("anthropic-version", "2023-06-01")
	req.Header.Set("Accept", "text/event-stream")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		sseError(w, fmt.Sprintf("AI API 请求失败：%v", err))
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		sseError(w, fmt.Sprintf("AI API error %d: %s", resp.StatusCode, string(body[:min(300, len(body))])))
		return
	}

	// Read SSE stream from AI API and forward text deltas to client.
	var fullContent strings.Builder
	if err := scanSSEData(resp.Body, func(data string) {
		if text := anthropicTextDelta(data); text != "" {
			fullContent.WriteString(text)
			sseChunk := fmt.Sprintf("data: %s\n\n", strings.ReplaceAll(text, "\n", "\\n"))
			_, _ = w.Write([]byte(sseChunk))
			flusher.Flush()
		}
	}); err != nil {
		sseError(w, fmt.Sprintf("AI API stream read failed: %v", err))
		return
	}

	content := fullContent.String()
	if content == "" {
		sseError(w, "AI 返回了空响应")
		return
	}

	// Persist to DB
	now := time.Now()
	userMsg := ChatMessage{
		ID: fmt.Sprintf("msg-%d-user", now.UnixNano()), SessionID: sessionID, AgentID: agentID,
		Role: "user", Content: userContent, CreatedAt: now,
	}
	assistantMsg := ChatMessage{
		ID: fmt.Sprintf("msg-%d-assistant", now.UnixNano()+1), SessionID: sessionID, AgentID: agentID,
		Role: "assistant", Content: content, CreatedAt: now,
	}
	_ = app.store.update(func(state *State) error {
		ensureChatMaps(state)
		sessions := state.ChatSessions[agentID]
		found := false
		for i := range sessions {
			if sessions[i].ID == sessionID {
				sessions[i].Preview = firstRunes(userContent, 50)
				sessions[i].UpdatedAt = now
				found = true
				break
			}
		}
		if !found {
			if len(sessions) >= 20 {
				return fmt.Errorf("会话数已达上限")
			}
			s := newChatSession(agentID, firstRunes(userContent, 20))
			s.ID = sessionID
			s.Preview = firstRunes(userContent, 50)
			sessions = append([]ChatSession{s}, sessions...)
		}
		state.ChatSessions[agentID] = sessions
		key := chatKey(agentID, sessionID)
		state.ChatMessages[key] = append(state.ChatMessages[key], userMsg, assistantMsg)
		return nil
	})
	w.Write([]byte("data: [DONE]\n\n"))
	flusher.Flush()
}

func sseError(w http.ResponseWriter, msg string) {
	w.Write([]byte(fmt.Sprintf("data: {\"error\":%q}\n\n", msg)))
	if flusher, ok := w.(http.Flusher); ok {
		flusher.Flush()
	}
}

func scanSSEData(r io.Reader, handle func(data string)) error {
	scanner := bufio.NewScanner(r)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "" {
			continue
		}
		handle(data)
	}
	return scanner.Err()
}

func anthropicTextDelta(data string) string {
	if data == "[DONE]" {
		return ""
	}
	var event struct {
		Type  string `json:"type"`
		Delta struct {
			Type string `json:"type"`
			Text string `json:"text"`
		} `json:"delta"`
	}
	if err := json.Unmarshal([]byte(data), &event); err != nil {
		return ""
	}
	if event.Type == "content_block_delta" && event.Delta.Type == "text_delta" {
		return event.Delta.Text
	}
	return ""
}

func firstRunes(s string, n int) string {
	runes := []rune(s)
	if len(runes) > n {
		return string(runes[:n])
	}
	return s
}
