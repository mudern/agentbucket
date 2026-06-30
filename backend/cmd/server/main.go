package main

import (
	"archive/tar"
	"bytes"
	"context"
	"crypto/sha1"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"sync"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type App struct {
	rootDir string
	dataDir string
	store   *Store
	bus     *AgentBus
}

type AgentBus struct {
	mu       sync.RWMutex
	agents   map[string]BusAgent
	messages []BusMessage
}

type BusAgent struct {
	AgentID  string    `json:"agentId"`
	Name     string    `json:"name"`
	Status   string    `json:"status"`
	Endpoint string    `json:"endpoint"`
	LastSeen time.Time `json:"lastSeen"`
}

type BusMessage struct {
	ID        string    `json:"id"`
	FromAgent string    `json:"fromAgent"`
	ToAgent   string    `json:"toAgent"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type Store struct {
	mu    sync.Mutex
	db    *sql.DB
	state State
}

type State struct {
	CurrentUser  CurrentUser              `json:"currentUser"`
	Repositories []Repository             `json:"repositories"`
	AITokens     []AIToken                `json:"aiTokens"`
	AuthTokens   []AuthToken              `json:"authTokens"`
	Users        []User                   `json:"users"`
	Approvals    []Approval               `json:"approvals"`
	Deployments  []Deployment             `json:"deployments"`
	ChatSessions map[string][]ChatSession `json:"chatSessions"`
	ChatMessages map[string][]ChatMessage `json:"chatMessages"`
}

type CurrentUser struct {
	ID   string `json:"id"`
	Name string `json:"name"`
	Role string `json:"role"`
}

type Repository struct {
	ID         string   `json:"id"`
	Provider   string   `json:"provider"`
	URL        string   `json:"url"`
	Branch     string   `json:"branch"`
	AgentsPath string   `json:"agentsPath"`
	LocalPath  string   `json:"localPath"`
	Status     string   `json:"status"`
	Commits    []Commit `json:"commits"`
}

type Commit struct {
	Hash        string  `json:"hash"`
	Message     string  `json:"message"`
	CommittedAt string  `json:"committedAt"`
	Agents      []Agent `json:"agents"`
}

type Agent struct {
	ID             string   `json:"id"`
	Name           string   `json:"name"`
	Path           string   `json:"path"`
	Description    string   `json:"description"`
	Model          string   `json:"model"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtimeVersion"`
	APIToken       string   `json:"apiToken"`
	Skills         []string `json:"skills"`
	MCPs           []string `json:"mcps"`
	ExtraInstall   []string `json:"extraInstall,omitempty"`
	Status         string   `json:"status,omitempty"`
	Tags           []string `json:"tags,omitempty"`
	UpdatedAt      string   `json:"updatedAt,omitempty"`
}

type AIToken struct {
	ID       int    `json:"id"`
	Name     string `json:"name"`
	Provider string `json:"provider"`
	Scope    string `json:"scope"`
	Usage    string `json:"usage"`
	Status   string `json:"status"`
	BaseURL  string `json:"baseUrl,omitempty"`
	Model    string `json:"model,omitempty"`
	Secret   string `json:"-"`
}

type AuthToken struct {
	ID           int    `json:"id"`
	Name         string `json:"name"`
	AccessTarget string `json:"accessTarget"`
	Script       string `json:"script"`
	FunctionName string `json:"functionName"`
	Argument     string `json:"argument"`
	Status       string `json:"status"`
	UpdatedAt    string `json:"updatedAt"`
}

type User struct {
	ID     int    `json:"id"`
	Name   string `json:"name"`
	Email  string `json:"email"`
	Role   string `json:"role"`
	Active bool   `json:"active"`
}

type Approval struct {
	ID        int    `json:"id"`
	Type      string `json:"type"`
	Applicant string `json:"applicant"`
	Summary   string `json:"summary"`
	Priority  string `json:"priority"`
	Status    string `json:"status"`
	Reviewer  string `json:"reviewer,omitempty"`
}

type MCPServer struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Scope string `json:"scope"`
}

type DeployOptions struct {
	Repositories []Repository `json:"repositories"`
	Models       []string     `json:"models"`
	Runtimes     []string     `json:"runtimes"`
	RuntimeTags  []string     `json:"runtimeTags"`
	MCPServers   []MCPServer  `json:"mcpServers"`
	AITokens     []AIToken    `json:"aiTokens"`
	AuthTokens   []AuthToken  `json:"authTokens"`
}

type Deployment struct {
	ID             string    `json:"id"`
	RepositoryID   string    `json:"repositoryId"`
	CommitHash     string    `json:"commitHash"`
	AgentID        string    `json:"agentId"`
	APITokenID     int       `json:"apiTokenId"`
	Model          string    `json:"model"`
	Runtime        string    `json:"runtime"`
	RuntimeVersion string    `json:"runtimeVersion"`
	Skills         []string  `json:"skills"`
	MCPs           []string  `json:"mcps"`
	AuthTokens     []int     `json:"authTokens"`
	ImageTag       string    `json:"imageTag"`
	ContainerName  string    `json:"containerName"`
	Status         string    `json:"status"`
	Message        string    `json:"message"`
	BuildContext   string    `json:"buildContext"`
	SidecarURL     string    `json:"sidecarUrl"`
	HostPort       int       `json:"hostPort"`
	CreatedAt      time.Time `json:"createdAt"`
}

type DeployRequest struct {
	RepositoryID   string   `json:"repositoryId"`
	CommitHash     string   `json:"commitHash"`
	AgentID        string   `json:"agentId"`
	APITokenID     int      `json:"apiTokenId"`
	Model          string   `json:"model"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtimeVersion"`
	AuthTokens     []int    `json:"authTokens"`
	Skills         []string `json:"skills"`
	MCPs           []string `json:"mcps"`
	ExtraInstall   []string `json:"extraInstall"`
}

type TokenResolveRequest struct {
	TokenID int    `json:"tokenId"`
	Param   string `json:"param"`
}

type ChatSession struct {
	ID        string    `json:"id"`
	AgentID   string    `json:"agentId"`
	Title     string    `json:"title"`
	Preview   string    `json:"preview"`
	CreatedAt time.Time `json:"createdAt"`
	UpdatedAt time.Time `json:"updatedAt"`
}

type ChatMessage struct {
	ID        string    `json:"id"`
	SessionID string    `json:"sessionId"`
	AgentID   string    `json:"agentId"`
	Role      string    `json:"role"`
	Content   string    `json:"content"`
	CreatedAt time.Time `json:"createdAt"`
}

type SendMessageRequest struct {
	SessionID string `json:"sessionId"`
	Content   string `json:"content"`
}

func main() {
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
	// Agent bus
	mux.HandleFunc("GET /api/bus/agents", app.busAgents)
	mux.HandleFunc("POST /api/bus/agents/{agentId}/register", app.busRegister)
	mux.HandleFunc("POST /api/bus/agents/{agentId}/message", app.busSendMessage)
	mux.HandleFunc("GET /api/bus/messages", app.busMessages)

	addr := env("AGENTBUCKET_ADDR", "127.0.0.1:8080")
	log.Printf("AgentBucket backend listening on http://%s", addr)
	log.Fatal(http.ListenAndServe(addr, withCORS(mux)))
}

func NewStore(path string, rootDir string) (*Store, error) {
	db, err := sql.Open("sqlite3", path)
	if err != nil {
		return nil, err
	}
	store := &Store{db: db}
	if err := store.initSchema(); err != nil {
		return nil, err
	}
	raw, err := store.loadStateJSON()
	if err != nil {
		return nil, err
	}
	if len(raw) == 0 {
		store.state = seedState(rootDir)
		store.importCCSAITokens()
		return store.saveLocked()
	}
	if err := json.Unmarshal(raw, &store.state); err != nil {
		return nil, err
	}
	if len(store.state.Users) == 0 {
		users, err := store.loadUsers()
		if err != nil {
			return nil, err
		}
		store.state.Users = users
	}
	if err := store.loadChat(); err != nil {
		return nil, err
	}
	store.importCCSAITokens()
	if _, err := store.saveLocked(); err != nil {
		return nil, err
	}
	return store, nil
}

func (s *Store) initSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS app_state (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL UNIQUE,
			role TEXT NOT NULL,
			active INTEGER NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_sessions (
			id TEXT PRIMARY KEY,
			agent_id TEXT NOT NULL,
			title TEXT NOT NULL,
			preview TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		)`,
		`CREATE TABLE IF NOT EXISTS chat_messages (
			id TEXT PRIMARY KEY,
			session_id TEXT NOT NULL,
			agent_id TEXT NOT NULL,
			role TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_sessions_agent_updated ON chat_sessions(agent_id, updated_at DESC)`,
		`CREATE INDEX IF NOT EXISTS idx_chat_messages_agent_session_created ON chat_messages(agent_id, session_id, created_at ASC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	return nil
}

func (s *Store) loadStateJSON() ([]byte, error) {
	var value string
	err := s.db.QueryRow(`SELECT value FROM app_state WHERE key = 'state'`).Scan(&value)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return []byte(value), nil
}

func (s *Store) loadUsers() ([]User, error) {
	rows, err := s.db.Query(`SELECT id, name, email, role, active FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		var active int
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &active); err != nil {
			return nil, err
		}
		user.Active = active == 1
		users = append(users, user)
	}
	return users, rows.Err()
}

func (s *Store) loadChat() error {
	sessions := map[string][]ChatSession{}
	sessionRows, err := s.db.Query(`SELECT id, agent_id, title, preview, created_at, updated_at FROM chat_sessions ORDER BY updated_at DESC`)
	if err != nil {
		return err
	}
	defer sessionRows.Close()
	for sessionRows.Next() {
		var session ChatSession
		var createdAt, updatedAt string
		if err := sessionRows.Scan(&session.ID, &session.AgentID, &session.Title, &session.Preview, &createdAt, &updatedAt); err != nil {
			return err
		}
		session.CreatedAt = parseStoredTime(createdAt)
		session.UpdatedAt = parseStoredTime(updatedAt)
		sessions[session.AgentID] = append(sessions[session.AgentID], session)
	}
	if err := sessionRows.Err(); err != nil {
		return err
	}

	messages := map[string][]ChatMessage{}
	messageRows, err := s.db.Query(`SELECT id, session_id, agent_id, role, content, created_at FROM chat_messages ORDER BY created_at ASC`)
	if err != nil {
		return err
	}
	defer messageRows.Close()
	for messageRows.Next() {
		var message ChatMessage
		var createdAt string
		if err := messageRows.Scan(&message.ID, &message.SessionID, &message.AgentID, &message.Role, &message.Content, &createdAt); err != nil {
			return err
		}
		message.CreatedAt = parseStoredTime(createdAt)
		messages[chatKey(message.AgentID, message.SessionID)] = append(messages[chatKey(message.AgentID, message.SessionID)], message)
	}
	if err := messageRows.Err(); err != nil {
		return err
	}
	s.state.ChatSessions = sessions
	s.state.ChatMessages = messages
	return nil
}

func (s *Store) importCCSAITokens() {
	home, err := os.UserHomeDir()
	if err != nil {
		return
	}
	providersDir := filepath.Join(home, ".config", "ccs", "providers")
	entries, err := os.ReadDir(providersDir)
	if err != nil {
		return
	}
	existing := map[string]int{}
	maxID := 0
	for _, token := range s.state.AITokens {
		existing[token.Name] = token.ID
		if token.ID > maxID {
			maxID = token.ID
		}
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".env") {
			continue
		}
		provider := strings.TrimSuffix(entry.Name(), ".env")
		values, err := readEnvFile(filepath.Join(providersDir, entry.Name()))
		if err != nil {
			continue
		}
		name := provider
		token := AIToken{
			Name:     name,
			Provider: strings.ToUpper(provider),
			Scope:    "imported provider env",
			Usage:    "local env",
			Status:   "启用",
			BaseURL:  values["ANTHROPIC_BASE_URL"],
			Model:    values["ANTHROPIC_MODEL"],
			Secret:   values["ANTHROPIC_AUTH_TOKEN"],
		}
		if id, ok := existing[name]; ok {
			for i := range s.state.AITokens {
				if s.state.AITokens[i].ID == id {
					token.ID = id
					s.state.AITokens[i] = token
					break
				}
			}
			continue
		}
		maxID++
		token.ID = maxID
		s.state.AITokens = append(s.state.AITokens, token)
	}
}

func readEnvFile(path string) (map[string]string, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	values := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		values[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	}
	return values, nil
}

func seedState(rootDir string) State {
	return State{
		CurrentUser: CurrentUser{ID: "u-1001", Name: "管理员", Role: "super_admin"},
		Repositories: []Repository{
			{
				ID:         "agentbucket-example",
				Provider:   "Local",
				URL:        "file://backend/examples/agent-repo",
				Branch:     "main",
				AgentsPath: "agents",
				LocalPath:  filepath.Join(rootDir, "examples", "agent-repo"),
				Status:     "启用",
			},
		},
		AITokens:    []AIToken{},
		AuthTokens: []AuthToken{
			{ID: 101, Name: "Test Public API", AccessTarget: "测试外部公开 API，所有已部署 Agent 可访问", Script: "tokens/test_public.py", FunctionName: "get_token", Argument: "scope", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 102, Name: "Test Admin API", AccessTarget: "测试管理员 API，仅允许显式授权 Agent", Script: "tokens/test_admin.py", FunctionName: "get_token", Argument: "scope", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 103, Name: "Test Disabled API", AccessTarget: "测试停用 Token 拒绝逻辑", Script: "tokens/test_disabled.py", FunctionName: "get_token", Argument: "scope", Status: "停用", UpdatedAt: "刚刚"},
			{ID: 104, Name: "GitHub Token", AccessTarget: "访问 GitHub 仓库、Issues、PR", Script: "tokens/github_token.py", FunctionName: "get_token", Argument: "repo", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 105, Name: "Internal DB", AccessTarget: "内部数据库只读访问", Script: "tokens/db_read.py", FunctionName: "get_token", Argument: "database", Status: "启用", UpdatedAt: "刚刚"},
		},
		Users: []User{
			{ID: 1, Name: "Luna", Email: "luna@agentbucket.dev", Role: "super_admin", Active: true},
			{ID: 2, Name: "Alex", Email: "alex@agentbucket.dev", Role: "admin", Active: true},
			{ID: 3, Name: "Ivy", Email: "ivy@agentbucket.dev", Role: "user", Active: true},
			{ID: 4, Name: "Noah", Email: "noah@agentbucket.dev", Role: "user", Active: false},
		},
		Approvals:    []Approval{},
		ChatSessions: map[string][]ChatSession{},
		ChatMessages: map[string][]ChatMessage{},
	}
}

func (s *Store) saveLocked() (*Store, error) {
	raw, err := json.MarshalIndent(s.state, "", "  ")
	if err != nil {
		return nil, err
	}
	tx, err := s.db.Begin()
	if err != nil {
		return nil, err
	}
	if _, err := tx.Exec(`INSERT INTO app_state (key, value) VALUES ('state', ?) ON CONFLICT(key) DO UPDATE SET value = excluded.value`, string(raw)); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM users`); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM chat_sessions`); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	if _, err := tx.Exec(`DELETE FROM chat_messages`); err != nil {
		_ = tx.Rollback()
		return nil, err
	}
	for _, user := range s.state.Users {
		active := 0
		if user.Active {
			active = 1
		}
		if _, err := tx.Exec(`INSERT INTO users (id, name, email, role, active) VALUES (?, ?, ?, ?, ?)`, user.ID, user.Name, user.Email, user.Role, active); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
	}
	for _, sessions := range s.state.ChatSessions {
		for _, session := range sessions {
			if _, err := tx.Exec(
				`INSERT INTO chat_sessions (id, agent_id, title, preview, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				session.ID,
				session.AgentID,
				session.Title,
				session.Preview,
				formatStoredTime(session.CreatedAt),
				formatStoredTime(session.UpdatedAt),
			); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
		}
	}
	for _, messages := range s.state.ChatMessages {
		for _, message := range messages {
			if _, err := tx.Exec(
				`INSERT INTO chat_messages (id, session_id, agent_id, role, content, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
				message.ID,
				message.SessionID,
				message.AgentID,
				message.Role,
				message.Content,
				formatStoredTime(message.CreatedAt),
			); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) snapshot() State {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.state
}

func (s *Store) update(fn func(*State) error) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if err := fn(&s.state); err != nil {
		return err
	}
	_, err := s.saveLocked()
	return err
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
	writeJSON(w, http.StatusOK, app.store.snapshot().Approvals)
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
		writeJSON(w, http.StatusOK, app.store.snapshot().AuthTokens)
	case http.MethodPost:
		var token AuthToken
		if err := json.NewDecoder(r.Body).Decode(&token); err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if token.Name == "" || token.AccessTarget == "" {
			writeError(w, http.StatusBadRequest, fmt.Errorf("name and accessTarget are required"))
			return
		}
		if token.FunctionName == "" {
			token.FunctionName = "get_token"
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
	var agents []Agent
	for _, repo := range app.scanRepositories(state.Repositories) {
		if len(repo.Commits) == 0 {
			continue
		}
		for _, agent := range repo.Commits[0].Agents {
			agent.Status = "离线"
			for _, d := range state.Deployments {
				if d.AgentID == agent.ID && d.Status == "running" {
					agent.Status = "已部署"
					break
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
		var req SendMessageRequest
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
		Runtimes:     []string{"codex", "claudecode"},
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
		if err := app.store.update(func(state *State) error {
			state.Repositories = append(state.Repositories, repo)
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
		deployment, err := app.createDeployment(req)
		if err != nil {
			writeError(w, http.StatusBadRequest, err)
			return
		}
		if err := app.store.update(func(state *State) error {
			state.Deployments = append([]Deployment{deployment}, state.Deployments...)
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
	deployments := app.store.snapshot().Deployments
	for i := range deployments {
		if deployments[i].ID == id {
			writeJSON(w, http.StatusOK, deployments[i])
			return
		}
	}
	writeError(w, http.StatusNotFound, fmt.Errorf("deployment %q not found", id))
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
	run := exec.Command(
		"docker", "run", "-d", "--rm",
		"--name", target.ContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:8088", target.HostPort),
		"--add-host", "host.docker.internal:host-gateway",
		"-e", fmt.Sprintf("AGENTBUCKET_URL=http://host.docker.internal:%d", mustPort()),
		target.ImageTag,
	)
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
	writeJSON(w, http.StatusOK, map[string]any{
		"tokenId":      token.ID,
		"name":         token.Name,
		"accessTarget": token.AccessTarget,
		"param":        req.Param,
		"token":        fmt.Sprintf("test-token-%d-%s-%s", token.ID, slug(agentID), shortHash(req.Param)),
	})
}

func (app *App) scanRepositories(repos []Repository) []Repository {
	scanned := make([]Repository, 0, len(repos))
	for _, repo := range repos {
		next := repo
		commit := Commit{
			Hash:        shortHash(repo.URL + repo.LocalPath + repo.AgentsPath),
			Message:     "scanned local agent manifests",
			CommittedAt: "刚刚",
			Agents:      scanAgents(repo),
		}
		next.Commits = []Commit{commit}
		scanned = append(scanned, next)
	}
	return scanned
}

func scanAgents(repo Repository) []Agent {
	root := repoPath(repo)
	agentsDir := filepath.Join(root, filepath.FromSlash(repo.AgentsPath))
	entries, err := os.ReadDir(agentsDir)
	if err != nil {
		return nil
	}
	var agents []Agent
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		manifest := filepath.Join(agentsDir, entry.Name(), "agent.toml")
		agent, err := parseAgentManifest(manifest)
		if err != nil {
			continue
		}
		rel, _ := filepath.Rel(root, manifest)
		agent.Path = filepath.ToSlash(rel)
		if agent.ID == "" {
			agent.ID = entry.Name()
		}
		if agent.Name == "" {
			agent.Name = agent.ID
		}
		if agent.Runtime == "" {
			agent.Runtime = "codex"
		}
		if agent.RuntimeVersion == "" {
			agent.RuntimeVersion = "latest"
		}
		agents = append(agents, agent)
	}
	sort.Slice(agents, func(i, j int) bool { return agents[i].ID < agents[j].ID })
	return agents
}

func scanMCPServers(repos []Repository) []MCPServer {
	seen := map[string]MCPServer{}
	for _, repo := range repos {
		mcpDir := filepath.Join(repoPath(repo), "mcp")
		entries, err := os.ReadDir(mcpDir)
		if err != nil {
			continue
		}
		for _, entry := range entries {
			if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
				continue
			}
			var server MCPServer
			raw, err := os.ReadFile(filepath.Join(mcpDir, entry.Name()))
			if err != nil || json.Unmarshal(raw, &server) != nil || server.ID == "" {
				continue
			}
			seen[server.ID] = server
		}
	}
	for _, fallback := range []MCPServer{
		{ID: "github-mcp", Name: "GitHub MCP", Scope: "代码仓库"},
		{ID: "notion-mcp", Name: "Notion MCP", Scope: "知识库"},
		{ID: "filesystem-mcp", Name: "Filesystem MCP", Scope: "文档读取"},
		{ID: "jira-mcp", Name: "Jira MCP", Scope: "项目管理"},
		{ID: "grafana-mcp", Name: "Grafana MCP", Scope: "监控查询"},
	} {
		if _, ok := seen[fallback.ID]; !ok {
			seen[fallback.ID] = fallback
		}
	}
	var servers []MCPServer
	for _, server := range seen {
		servers = append(servers, server)
	}
	sort.Slice(servers, func(i, j int) bool { return servers[i].ID < servers[j].ID })
	return servers
}

func parseAgentManifest(path string) (Agent, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return Agent{}, err
	}
	values := parseSimpleTOML(string(raw))
	return Agent{
		ID:             values["id"].scalar,
		Name:           values["name"].scalar,
		Description:    values["description"].scalar,
		Model:          values["model"].scalar,
		Runtime:        values["runtime"].scalar,
		RuntimeVersion: values["runtime_version"].scalar,
		APIToken:       values["api_token"].scalar,
		Skills:         values["skills"].list,
		MCPs:           values["mcps"].list,
		ExtraInstall:   values["extra_install"].list,
	}, nil
}

type tomlValue struct {
	scalar string
	list   []string
}

func parseSimpleTOML(raw string) map[string]tomlValue {
	result := map[string]tomlValue{}
	for _, line := range strings.Split(raw, "\n") {
		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") {
			continue
		}
		parts := strings.SplitN(trimmed, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if strings.HasPrefix(value, "[") && strings.HasSuffix(value, "]") {
			result[key] = tomlValue{list: parseTOMLStringList(value)}
			continue
		}
		result[key] = tomlValue{scalar: strings.Trim(value, `"'`)}
	}
	return result
}

func parseTOMLStringList(value string) []string {
	value = strings.TrimSpace(strings.TrimSuffix(strings.TrimPrefix(value, "["), "]"))
	if value == "" {
		return nil
	}
	var items []string
	for _, item := range strings.Split(value, ",") {
		item = strings.Trim(strings.TrimSpace(item), `"'`)
		if item != "" {
			items = append(items, item)
		}
	}
	return items
}

func (app *App) createDeployment(req DeployRequest) (Deployment, error) {
	state := app.store.snapshot()
	repos := app.scanRepositories(state.Repositories)
	repo, commit, agent, err := findDeploymentTarget(repos, req)
	if err != nil {
		return Deployment{}, err
	}
	if req.Runtime == "" {
		req.Runtime = agent.Runtime
	}
	if req.RuntimeVersion == "" {
		req.RuntimeVersion = agent.RuntimeVersion
	}
	if req.Model == "" {
		req.Model = agent.Model
	}
	if len(req.Skills) == 0 {
		req.Skills = agent.Skills
	}
	if len(req.MCPs) == 0 {
		req.MCPs = agent.MCPs
	}
	if len(req.ExtraInstall) == 0 {
		req.ExtraInstall = agent.ExtraInstall
	}
	if req.Runtime != "codex" && req.Runtime != "claudecode" {
		return Deployment{}, fmt.Errorf("unsupported runtime %q", req.Runtime)
	}

	id := fmt.Sprintf("dep-%s-%d", slug(agent.ID), time.Now().Unix())
	contextDir := filepath.Join(app.dataDir, "deployments", id, "context")
	if err := os.MkdirAll(contextDir, 0o755); err != nil {
		return Deployment{}, err
	}
	if err := app.writeBuildContext(contextDir, repo, commit, agent, req); err != nil {
		return Deployment{}, err
	}

	deployment := Deployment{
		ID:             id,
		RepositoryID:   repo.ID,
		CommitHash:     commit.Hash,
		AgentID:        agent.ID,
		APITokenID:     req.APITokenID,
		Model:          req.Model,
		Runtime:        req.Runtime,
		RuntimeVersion: req.RuntimeVersion,
		Skills:         req.Skills,
		MCPs:           req.MCPs,
		AuthTokens:     req.AuthTokens,
		ImageTag:       "agentbucket/" + slug(agent.ID) + ":" + commit.Hash,
		ContainerName:  "agentbucket-" + slug(agent.ID),
		Status:         "packaged",
		Message:        "Docker build context generated",
		BuildContext:   contextDir,
		HostPort:       hostPortFor(agent.ID),
		CreatedAt:      time.Now(),
	}
	deployment.SidecarURL = fmt.Sprintf("http://127.0.0.1:%d", deployment.HostPort)

	if _, err := exec.LookPath("docker"); err != nil {
		deployment.Message = "Docker CLI not found; generated build context only"
		return deployment, nil
	}
	ctx, cancel := context.WithTimeout(context.Background(), buildTimeout())
	defer cancel()
	build := exec.CommandContext(ctx, "docker", "build", "-t", deployment.ImageTag, contextDir)
	out, err := build.CombinedOutput()
	if err != nil {
		deployment.Status = "build_failed"
		if ctx.Err() == context.DeadlineExceeded {
			deployment.Message = "docker build timed out: " + string(out)
		} else {
			deployment.Message = string(out)
		}
		return deployment, nil
	}
	_ = exec.Command("docker", "rm", "-f", deployment.ContainerName).Run()
	run := exec.Command(
		"docker", "run", "-d", "--rm",
		"--name", deployment.ContainerName,
		"-p", fmt.Sprintf("127.0.0.1:%d:8088", deployment.HostPort),
		"--add-host", "host.docker.internal:host-gateway",
		"-e", "AGENTBUCKET_URL=http://host.docker.internal:8080",
		deployment.ImageTag,
	)
	out, err = run.CombinedOutput()
	if err != nil {
		deployment.Status = "run_failed"
		deployment.Message = string(out)
		return deployment, nil
	}
	deployment.Status = "running"
	deployment.Message = strings.TrimSpace(string(out))
	return deployment, nil
}

func findDeploymentTarget(repos []Repository, req DeployRequest) (Repository, Commit, Agent, error) {
	for _, repo := range repos {
		if repo.ID != req.RepositoryID {
			continue
		}
		for _, commit := range repo.Commits {
			if req.CommitHash != "" && commit.Hash != req.CommitHash {
				continue
			}
			for _, agent := range commit.Agents {
				if agent.ID == req.AgentID {
					return repo, commit, agent, nil
				}
			}
		}
	}
	return Repository{}, Commit{}, Agent{}, fmt.Errorf("deployment target not found")
}

func (app *App) writeBuildContext(contextDir string, repo Repository, commit Commit, agent Agent, req DeployRequest) error {
	repoRoot := repoPath(repo)
	agentSource := filepath.Join(repoRoot, filepath.FromSlash(agent.Path))
	agentDir := filepath.Dir(agentSource)
	if err := copyDir(agentDir, filepath.Join(contextDir, "agent")); err != nil {
		return err
	}
	if err := copySelectedSkills(filepath.Join(repoRoot, "skills"), filepath.Join(contextDir, "skills"), req.Skills); err != nil {
		return err
	}
	if err := copyOptionalDir(filepath.Join(repoRoot, "mcp"), filepath.Join(contextDir, "mcp")); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Join(contextDir, "sidecar"), 0o755); err != nil {
		return err
	}
	config := map[string]any{
		"repositoryId":   repo.ID,
		"commitHash":     commit.Hash,
		"agentId":        agent.ID,
		"model":          req.Model,
		"runtime":        req.Runtime,
		"runtimeVersion": req.RuntimeVersion,
		"apiTokenId":     req.APITokenID,
		"skills":         req.Skills,
		"mcps":           req.MCPs,
		"authTokens":     req.AuthTokens,
	}
	raw, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDir, "agentbucket.config.json"), raw, 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDir, "sidecar", "main.go"), []byte(sidecarSource), 0o644); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(contextDir, "Dockerfile"), []byte(dockerfileFor(req.Runtime, req.RuntimeVersion, req.ExtraInstall)), 0o644); err != nil {
		return err
	}
	return writeTarball(contextDir)
}

func dockerfileFor(runtime string, version string, extraInstall []string) string {
	if version == "" {
		version = "latest"
	}
	runtimeLine := fmt.Sprintf("ENV AGENTBUCKET_RUNTIME=%s\nENV AGENTBUCKET_RUNTIME_VERSION=%s\n", runtime, version)
	installLine := runtimeInstallLine(runtime, version)
	extraLine := ""
	for _, cmd := range extraInstall {
		extraLine += "RUN " + cmd + "\n"
	}
	return `FROM golang:1.22-alpine AS sidecar-build
WORKDIR /src
COPY sidecar/main.go .
RUN go build -o /out/agentbucket-sidecar main.go

FROM node:20-alpine
RUN apk add --no-cache ca-certificates bash curl git
` + extraLine + installLine + `
WORKDIR /app
` + runtimeLine + `COPY --from=sidecar-build /out/agentbucket-sidecar /usr/local/bin/agentbucket-sidecar
COPY agentbucket.config.json /app/agentbucket.config.json
COPY agent /app/agent
COPY skills /app/skills
COPY mcp /app/mcp
EXPOSE 8088
ENTRYPOINT ["/usr/local/bin/agentbucket-sidecar"]
`
}

func runtimeInstallLine(runtime string, version string) string {
	switch runtime {
	case "codex":
		return fmt.Sprintf("RUN npm install -g @openai/codex@%s", version)
	case "claudecode":
		return fmt.Sprintf("RUN npm install -g @anthropic-ai/claude-code@%s", version)
	default:
		return "RUN true"
	}
}

const sidecarSource = `package main

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"log"
	"net/http"
	"os"
	"os/exec"
	"sync"
	"time"
)

type Config struct {
	AgentID string ` + "`json:\"agentId\"`" + `
	Runtime string ` + "`json:\"runtime\"`" + `
	RuntimeVersion string ` + "`json:\"runtimeVersion\"`" + `
	Model string ` + "`json:\"model\"`" + `
	Skills []string ` + "`json:\"skills\"`" + `
	MCPs []string ` + "`json:\"mcps\"`" + `
	AuthTokens []int ` + "`json:\"authTokens\"`" + `
}

type RuntimeRunner interface {
	Command(config Config) *exec.Cmd
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

type ClaudeCodeRunner struct{ version string }
func (r ClaudeCodeRunner) Name() string { return "claudecode" }
func (r ClaudeCodeRunner) Version() string { return r.version }
func (r ClaudeCodeRunner) Command(config Config) *exec.Cmd {
	cmd := exec.Command("claude", "-p", "AgentBucket sidecar online")
	cmd.Dir = "/app/agent"
	return cmd
}

var (
	config Config
	agentMu sync.Mutex
	agentCmd *exec.Cmd
	agentStartedAt time.Time
	lastError string
)

func main() {
	raw, err := os.ReadFile("/app/agentbucket.config.json")
	if err != nil {
		log.Fatal(err)
	}
	if err := json.Unmarshal(raw, &config); err != nil {
		log.Fatal(err)
	}
	// Auto-register on AgentBucket bus
	go func() {
		baseURL := os.Getenv("AGENTBUCKET_URL")
		if baseURL == "" {
			baseURL = "http://host.docker.internal:8080"
		}
		payload, _ := json.Marshal(map[string]string{
			"name": config.AgentID,
			"status": "online",
			"endpoint": "http://localhost:8088",
		})
		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest(http.MethodPost, baseURL+"/api/bus/agents/"+config.AgentID+"/register", bytes.NewReader(payload))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err == nil {
				resp.Body.Close()
				if resp.StatusCode < 400 {
					break
				}
			}
			time.Sleep(2 * time.Second)
		}
	}()
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
	default:
		return CodexRunner{version: version}
	}
}

func health(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "online": isOnline(), "agent": config.AgentID, "runtime": config.Runtime})
}

func status(w http.ResponseWriter, r *http.Request) {
	runner := runnerFor(config)
	json.NewEncoder(w).Encode(map[string]any{
		"ok": true,
		"online": isOnline(),
		"agent": config.AgentID,
		"runtime": runner.Name(),
		"runtimeVersion": runner.Version(),
		"model": config.Model,
		"skills": config.Skills,
		"mcps": config.MCPs,
		"authTokens": config.AuthTokens,
		"startedAt": agentStartedAt,
		"lastError": lastError,
	})
}

func startAgent(w http.ResponseWriter, r *http.Request) {
	agentMu.Lock()
	defer agentMu.Unlock()
	if agentCmd != nil && agentCmd.Process != nil {
		json.NewEncoder(w).Encode(map[string]any{"ok": true, "status": "already_running"})
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
		http.Error(w, err.Error(), 500)
		return
	}
	agentStartedAt = time.Now()
	lastError = ""
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "pid": agentCmd.Process.Pid})
}

func stopAgent(w http.ResponseWriter, r *http.Request) {
	agentMu.Lock()
	defer agentMu.Unlock()
	if agentCmd != nil && agentCmd.Process != nil {
		_ = agentCmd.Process.Kill()
		agentCmd = nil
	}
	json.NewEncoder(w).Encode(map[string]any{"ok": true})
}

func registerAgent(w http.ResponseWriter, r *http.Request) {
	json.NewEncoder(w).Encode(map[string]any{"ok": true, "agent": config.AgentID, "runtime": config.Runtime, "online": isOnline(), "skills": config.Skills, "mcps": config.MCPs})
}

func getToken(w http.ResponseWriter, r *http.Request) {
	var payload map[string]any
	if err := json.NewDecoder(r.Body).Decode(&payload); err != nil {
		http.Error(w, err.Error(), 400)
		return
	}
	body, _ := json.Marshal(payload)
	baseURL := os.Getenv("AGENTBUCKET_URL")
	if baseURL == "" {
		baseURL = "http://host.docker.internal:8080"
	}
	req, err := http.NewRequest(http.MethodPost, baseURL+"/api/tokens/resolve", bytes.NewReader(body))
	if err != nil {
		http.Error(w, err.Error(), 500)
		return
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Agent-ID", config.AgentID)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		http.Error(w, err.Error(), 502)
		return
	}
	defer resp.Body.Close()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(resp.StatusCode)
	_, _ = io.Copy(w, resp.Body)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", 405)
		return
	}
	var req struct {
		Message string ` + "`json:\"message\"`" + `
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil || req.Message == "" {
		http.Error(w, "bad request", 400)
		return
	}

	runner := runnerFor(config)
	cmd := runner.Command(config)
	// Replace the static prompt with user message
	cmd.Args = append(cmd.Args[:len(cmd.Args)-1], req.Message)
	cmd.Env = append(os.Environ(),
		"AGENTBUCKET_AGENT_ID="+config.AgentID,
		"AGENTBUCKET_MODEL="+config.Model,
	)
	cmd.Dir = "/app/agent"

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
			json.NewEncoder(w).Encode(map[string]string{"content": output})
			return
		}
		json.NewEncoder(w).Encode(map[string]string{"content": stdout.String()})
	case <-ctx.Done():
		if cmd.Process != nil {
			cmd.Process.Kill()
		}
		json.NewEncoder(w).Encode(map[string]string{"content": "命令执行超时。"})
	}
}

func isOnline() bool {
	agentMu.Lock()
	defer agentMu.Unlock()
	return agentCmd != nil && agentCmd.Process != nil
}
`

func copyOptionalDir(src, dst string) error {
	if _, err := os.Stat(src); errors.Is(err, os.ErrNotExist) {
		return os.MkdirAll(dst, 0o755)
	}
	return copyDir(src, dst)
}

func copySelectedSkills(srcRoot, dstRoot string, skillIDs []string) error {
	if err := os.MkdirAll(dstRoot, 0o755); err != nil {
		return err
	}
	for _, skillID := range skillIDs {
		src := filepath.Join(srcRoot, filepath.Clean(skillID))
		if !strings.HasPrefix(src, filepath.Clean(srcRoot)+string(os.PathSeparator)) {
			return fmt.Errorf("invalid skill id %q", skillID)
		}
		if _, err := os.Stat(filepath.Join(src, "SKILL.md")); err != nil {
			return fmt.Errorf("skill %q must be a standard skill directory with SKILL.md", skillID)
		}
		if err := copyDir(src, filepath.Join(dstRoot, skillID)); err != nil {
			return err
		}
	}
	return nil
}

func copyDir(src, dst string) error {
	return filepath.WalkDir(src, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(src, path)
		if err != nil {
			return err
		}
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		in, err := os.Open(path)
		if err != nil {
			return err
		}
		defer in.Close()
		out, err := os.Create(target)
		if err != nil {
			return err
		}
		defer out.Close()
		_, err = io.Copy(out, in)
		return err
	})
}

func writeTarball(contextDir string) error {
	out, err := os.Create(filepath.Join(filepath.Dir(contextDir), "context.tar"))
	if err != nil {
		return err
	}
	defer out.Close()
	tw := tar.NewWriter(out)
	defer tw.Close()
	return filepath.WalkDir(contextDir, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		rel, err := filepath.Rel(contextDir, path)
		if err != nil {
			return err
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		header, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return err
		}
		header.Name = filepath.ToSlash(rel)
		if err := tw.WriteHeader(header); err != nil {
			return err
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		_, err = tw.Write(raw)
		return err
	})
}

func repoPath(repo Repository) string {
	if repo.LocalPath != "" {
		return repo.LocalPath
	}
	if strings.HasPrefix(repo.URL, "file://") {
		return strings.TrimPrefix(repo.URL, "file://")
	}
	return repo.URL
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
	}
	cmd.Env = append(os.Environ(),
		"ANTHROPIC_AUTH_TOKEN="+authToken,
		"ANTHROPIC_BASE_URL="+baseURL,
		"ANTHROPIC_MODEL="+model,
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

func parseStoredTime(value string) time.Time {
	parsed, err := time.Parse(time.RFC3339Nano, value)
	if err != nil {
		return time.Time{}
	}
	return parsed
}

func formatStoredTime(value time.Time) string {
	if value.IsZero() {
		value = time.Now()
	}
	return value.Format(time.RFC3339Nano)
}

func shortHash(value string) string {
	sum := sha1.Sum([]byte(value))
	return hex.EncodeToString(sum[:])[:7]
}

func hostPortFor(value string) int {
	sum := sha1.Sum([]byte(value))
	return 18000 + int(sum[0])%1000
}

func slug(value string) string {
	value = strings.ToLower(value)
	var buf bytes.Buffer
	lastDash := false
	for _, ch := range value {
		ok := (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9')
		if ok {
			buf.WriteRune(ch)
			lastDash = false
			continue
		}
		if !lastDash {
			buf.WriteByte('-')
			lastDash = true
		}
	}
	return strings.Trim(buf.String(), "-")
}
