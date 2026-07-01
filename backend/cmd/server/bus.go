package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"sort"
	"strings"
	"time"
)

func mustPort() int {
	addr := env("AGENTBUCKET_ADDR", "127.0.0.1:8080")
	parts := strings.Split(addr, ":")
	if len(parts) == 2 {
		var port int
		fmt.Sscanf(parts[1], "%d", &port)
		return port
	}
	return 8080
}

func newAgentBus() *AgentBus {
	return &AgentBus{agents: map[string]BusAgent{}}
}

func (bus *AgentBus) register(agent BusAgent) {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	agent.LastSeen = time.Now()
	bus.agents[agent.AgentID] = agent
}

func (bus *AgentBus) list() []BusAgent {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	agents := make([]BusAgent, 0, len(bus.agents))
	for _, a := range bus.agents {
		agents = append(agents, a)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].AgentID < agents[j].AgentID })
	return agents
}

func (bus *AgentBus) post(fromAgent, toAgent, content string) BusMessage {
	bus.mu.Lock()
	defer bus.mu.Unlock()
	msg := BusMessage{
		ID:        fmt.Sprintf("bus-%d", time.Now().UnixNano()),
		FromAgent: fromAgent,
		ToAgent:   toAgent,
		Content:   content,
		CreatedAt: time.Now(),
	}
	bus.messages = append(bus.messages, msg)
	if len(bus.messages) > 200 {
		bus.messages = bus.messages[len(bus.messages)-200:]
	}
	return msg
}

func (bus *AgentBus) getMessages() []BusMessage {
	bus.mu.RLock()
	defer bus.mu.RUnlock()
	msgs := make([]BusMessage, len(bus.messages))
	copy(msgs, bus.messages)
	return msgs
}

func (app *App) busAgents(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, app.bus.list())
}

func (app *App) busRegister(w http.ResponseWriter, r *http.Request) {
	agentID := r.PathValue("agentId")
	var agent BusAgent
	if err := json.NewDecoder(r.Body).Decode(&agent); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	agent.AgentID = agentID
	app.bus.register(agent)
	writeJSON(w, http.StatusOK, map[string]any{"ok": true, "agent": agent})
}

func (app *App) busSendMessage(w http.ResponseWriter, r *http.Request) {
	fromAgent := r.PathValue("agentId")
	var req struct {
		ToAgent string `json:"toAgent"`
		Content string `json:"content"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}
	if req.ToAgent == "" || req.Content == "" {
		writeError(w, http.StatusBadRequest, fmt.Errorf("toAgent and content are required"))
		return
	}
	msg := app.bus.post(fromAgent, req.ToAgent, req.Content)
	writeJSON(w, http.StatusOK, msg)
}

func (app *App) busMessages(w http.ResponseWriter, r *http.Request) {
	msgs := app.bus.getMessages()
	toAgent := r.URL.Query().Get("toAgent")
	if toAgent != "" {
		filtered := make([]BusMessage, 0)
		for _, msg := range msgs {
			if msg.ToAgent == toAgent {
				filtered = append(filtered, msg)
			}
		}
		msgs = filtered
	}
	writeJSON(w, http.StatusOK, msgs)
}
