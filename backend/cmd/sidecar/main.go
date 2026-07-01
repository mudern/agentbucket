package main

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"strings"
	"sync"
	"time"
)

type Config struct {
	AgentID        string   `json:"agentId"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtimeVersion"`
	Model          string   `json:"model"`
	Skills         []string `json:"skills"`
	MCPs           []string `json:"mcps"`
	AuthTokens     []int    `json:"authTokens"`
}

type RuntimeRunner interface {
	Command(config Config) *exec.Cmd
	ChatCommand(config Config, message string) *exec.Cmd
	Name() string
	Version() string
}

type CodexRunner struct{ version string }

func (r CodexRunner) Name() string { return "codex" }

func (r CodexRunner) Version() string { return r.version }

func (r CodexRunner) Command(config Config) *exec.Cmd {
	cmd := exec.Command("codex", "exec", "--model", config.Model, "AgentBucket sidecar online")
	cmd.Dir = "/app/agent"
	return cmd
}

func (r CodexRunner) ChatCommand(config Config, message string) *exec.Cmd {
	cmd := exec.Command("codex", "exec", "--model", config.Model, message)
	cmd.Dir = "/app/agent"
	return cmd
}

type ClaudeCodeRunner struct{ version string }

func (r ClaudeCodeRunner) Name() string { return "claudecode" }

func (r ClaudeCodeRunner) Version() string { return r.version }

func (r ClaudeCodeRunner) Command(config Config) *exec.Cmd {
	cmd := exec.Command("claude", "-p", "AgentBucket sidecar online")
	cmd.Dir = "/app/agent"
	return cmd
}

func (r ClaudeCodeRunner) ChatCommand(config Config, message string) *exec.Cmd {
	cmd := exec.Command("claude", "-p", message)
	cmd.Dir = "/app/agent"
	return cmd
}

type OpenCodeRunner struct{ version string }

func (r OpenCodeRunner) Name() string { return "opencode" }

func (r OpenCodeRunner) Version() string { return r.version }

func (r OpenCodeRunner) Command(config Config) *exec.Cmd {
	cmd := exec.Command("opencode", "run", "--model", config.Model, "AgentBucket sidecar online")
	cmd.Dir = "/app/agent"
	return cmd
}

func (r OpenCodeRunner) ChatCommand(config Config, message string) *exec.Cmd {
	cmd := exec.Command("opencode", "run", "--model", config.Model, message)
	cmd.Dir = "/app/agent"
	return cmd
}

var (
	config         Config
	agentMu        sync.Mutex
	agentCmd       *exec.Cmd
	agentStartedAt time.Time
	lastError      string
)

func main() {
	raw, err := os.ReadFile("/app/agentbucket.config.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		log.Fatal(err)
	}

	go registerOnBus()

	mux := http.NewServeMux()
	mux.HandleFunc("/health", health)
	mux.HandleFunc("/status", status)
	mux.HandleFunc("/agent/start", startAgent)
	mux.HandleFunc("/agent/stop", stopAgent)
	mux.HandleFunc("/bus/register", registerAgent)
	mux.HandleFunc("/tokens/get", getToken)
	mux.HandleFunc("/agent/chat", handleChat)
	log.Fatal(http.ListenAndServe(":8088", mux))
}

func registerOnBus() {
	baseURL := os.Getenv("AGENTBUCKET_URL")
	if baseURL == "" {
		baseURL = "http://host.docker.internal:8080"
	}
	payload, _ := json.Marshal(map[string]string{
		"name":     config.AgentID,
		"status":   "online",
		"endpoint": "http://localhost:8088",
	})
	for i := 0; i < 10; i++ {
		req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/bus/agents/"+config.AgentID+"/register", bytes.NewReader(payload))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode < 400 {
				break
			}
		}
		time.Sleep(2 * time.Second)
	}
}

func runnerFor(config Config) RuntimeRunner {
	version := config.RuntimeVersion
	if version == "" {
		version = os.Getenv("AGENTBUCKET_RUNTIME_VERSION")
	}
	if version == "" {
		version = "latest"
	}
	switch config.Runtime {
	case "codex":
		return CodexRunner{version: version}
	case "claudecode":
		return ClaudeCodeRunner{version: version}
	case "opencode":
		return OpenCodeRunner{version: version}
	default:
		return CodexRunner{version: version}
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "online": isOnline(), "agent": config.AgentID, "runtime": config.Runtime})
}

func status(w http.ResponseWriter, r *http.Request) {
	runner := runnerFor(config)
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":             true,
		"online":         isOnline(),
		"agent":          config.AgentID,
		"runtime":        runner.Name(),
		"runtimeVersion": runner.Version(),
		"model":          config.Model,
		"skills":         config.Skills,
		"mcps":           config.MCPs,
		"authTokens":     config.AuthTokens,
		"startedAt":      agentStartedAt,
		"lastError":      lastError,
	})
}

func startAgent(w http.ResponseWriter, r *http.Request) {
	agentMu.Lock()
	defer agentMu.Unlock()
	if agentCmd != nil && agentCmd.Process != nil {
		_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "already_running"})
		return
	}
	runner := runnerFor(config)
	agentCmd = runner.Command(config)
	agentCmd.Env = append(os.Environ(),
		"AGENTBUCKET_AGENT_ID="+config.AgentID,
		"AGENTBUCKET_MODEL="+config.Model,
		"AGENTBUCKET_RUNTIME="+runner.Name(),
		"AGENTBUCKET_RUNTIME_VERSION="+runner.Version(),
	)
	if err := agentCmd.Start(); err != nil {
		lastError = err.Error()
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	agentStartedAt = time.Now()
	lastError = ""
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true, "pid": agentCmd.Process.Pid})
}

func stopAgent(w http.ResponseWriter, r *http.Request) {
	agentMu.Lock()
	defer agentMu.Unlock()
	if agentCmd != nil && agentCmd.Process != nil {
		_ = agentCmd.Process.Kill()
		agentCmd = nil
	}
	_ = json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func registerAgent(w http.ResponseWriter, r *http.Request) {
	_ = json.NewEncoder(w).Encode(map[string]any{
		"ok":      true,
		"agent":   config.AgentID,
		"runtime": config.Runtime,
		"online":  isOnline(),
		"skills":  config.Skills,
		"mcps":    config.MCPs,
	})
}

func getToken(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	body, _ := json.Marshal(payload)
	baseURL := os.Getenv("AGENTBUCKET_URL")
	if baseURL == "" {
		baseURL = "http://host.docker.internal:8080"
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/tokens/resolve", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", config.AgentID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadGateway)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}
	var req struct {
		Message string `json:"message"`
		Stream  bool   `json:"stream"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	runner := runnerFor(config)
	cmd := runner.ChatCommand(config, req.Message)
	cmd.Env = append(os.Environ(),
		"AGENTBUCKET_AGENT_ID="+config.AgentID,
		"AGENTBUCKET_MODEL="+config.Model,
	)

	if req.Stream {
		handleStreamChat(w, cmd)
	} else {
		handleOneShotChat(w, cmd)
	}
}

func handleStreamChat(w http.ResponseWriter, cmd *exec.Cmd) {
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	flusher, ok := w.(http.Flusher)
	if !ok {
		_ = json.NewEncoder(w).Encode(map[string]string{"content": "SSE not supported"})
		return
	}

	stdout, _ := cmd.StdoutPipe()
	var stderr bytes.Buffer
	cmd.Stderr = &stderr

	if err := cmd.Start(); err != nil {
		fmt.Fprintf(w, "data: [error] %s\n\n", err.Error())
		flusher.Flush()
		return
	}

	scanner := bufio.NewScanner(stdout)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" {
			continue
		}
		fmt.Fprintf(w, "data: %s\n\n", strings.ReplaceAll(line, "\n", "\\n"))
		flusher.Flush()
	}

	if err := cmd.Wait(); err != nil {
		errOutput := stderr.String()
		if errOutput == "" {
			errOutput = err.Error()
		}
		fmt.Fprintf(w, "data: [error] %s\n\n", strings.ReplaceAll(errOutput, "\n", "\\n"))
		flusher.Flush()
	}
}

func handleOneShotChat(w http.ResponseWriter, cmd *exec.Cmd) {
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	defer cancel()

	done := make(chan error, 1)
	go func() {
		done <- cmd.Run()
	}()

	select {
	case err := <-done:
		if err != nil {
			output := stderr.String()
			if output == "" {
				output = stdout.String()
			}
			if output == "" {
				output = err.Error()
			}
			_ = json.NewEncoder(w).Encode(map[string]string{"content": output})
			return
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"content": stdout.String()})
	case <-ctx.Done():
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		_ = json.NewEncoder(w).Encode(map[string]string{"content": "命令执行超时。"})
	}
}

func isOnline() bool {
	agentMu.Lock()
	defer agentMu.Unlock()
	return agentCmd != nil && agentCmd.Process != nil
}
