package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	_ "github.com/mattn/go-sqlite3"
)

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
		store.state.Users = ensureUserPasswordHashes(store.state.Users)
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
	store.state.Users = ensureUserPasswordHashes(store.state.Users)
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
			active INTEGER NOT NULL,
			password_hash TEXT NOT NULL DEFAULT ''
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
		`CREATE TABLE IF NOT EXISTS bus_messages (
			id TEXT PRIMARY KEY,
			from_agent TEXT NOT NULL,
			to_agent TEXT NOT NULL,
			content TEXT NOT NULL,
			created_at TEXT NOT NULL
		)`,
		`CREATE INDEX IF NOT EXISTS idx_bus_messages_to_agent ON bus_messages(to_agent, created_at DESC)`,
	}
	for _, statement := range statements {
		if _, err := s.db.Exec(statement); err != nil {
			return err
		}
	}
	if _, err := s.db.Exec(`ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`); err != nil && !strings.Contains(err.Error(), "duplicate column name") {
		return err
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
	rows, err := s.db.Query(`SELECT id, name, email, role, active, password_hash FROM users ORDER BY id`)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var users []User
	for rows.Next() {
		var user User
		var active int
		if err := rows.Scan(&user.ID, &user.Name, &user.Email, &user.Role, &active, &user.PasswordHash); err != nil {
			return nil, err
		}
		user.Active = active == 1
		users = append(users, user)
	}
	return users, rows.Err()
}

func ensureUserPasswordHashes(users []User) []User {
	for i := range users {
		if users[i].PasswordHash != "" {
			continue
		}
		switch users[i].Name {
		case "Luna", "Alex":
			users[i].PasswordHash = hashPassword("admin123")
		case "Ivy", "Noah":
			users[i].PasswordHash = hashPassword("user123")
		default:
			users[i].PasswordHash = hashPassword("password")
		}
	}
	return users
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
	providersDir := os.Getenv("AGENTBUCKET_PROVIDERS_DIR")
	if providersDir == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			return
		}
		providersDir = filepath.Join(home, ".config", "ccs", "providers")
	}
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
			{
				ID:         "github-test-agents",
				Provider:   "GitHub",
				URL:        "https://github.com/mudern/agentbucket-test-agents",
				Branch:     "main",
				AgentsPath: "agents",
				LocalPath:  "/tmp/agentbucket-test-agents",
				Status:     "启用",
			},
		},
		AITokens: []AIToken{},
		AuthTokens: []AuthToken{
			{ID: 101, Name: "Test Public API", AccessTarget: "测试外部公开 API，所有已部署 Agent 可访问", Script: "tokens/test_public.py", FunctionName: "get_token", Argument: "scope", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 102, Name: "Test Admin API", AccessTarget: "测试管理员 API，仅允许显式授权 Agent", Script: "tokens/test_admin.py", FunctionName: "get_token", Argument: "scope", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 103, Name: "Test Disabled API", AccessTarget: "测试停用 Token 拒绝逻辑", Script: "tokens/test_disabled.py", FunctionName: "get_token", Argument: "scope", Status: "停用", UpdatedAt: "刚刚"},
			{ID: 104, Name: "GitHub Token", AccessTarget: "访问 GitHub 仓库、Issues、PR", Script: "tokens/github_token.py", FunctionName: "get_token", Argument: "repo", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 105, Name: "Internal DB", AccessTarget: "内部数据库只读访问", Script: "tokens/db_read.py", FunctionName: "get_token", Argument: "database", Status: "启用", UpdatedAt: "刚刚"},
		},
		Users: []User{
			{ID: 1, Name: "Luna", Email: "luna@agentbucket.dev", Role: "super_admin", Active: true, PasswordHash: hashPassword("admin123")},
			{ID: 2, Name: "Alex", Email: "alex@agentbucket.dev", Role: "admin", Active: true, PasswordHash: hashPassword("admin123")},
			{ID: 3, Name: "Ivy", Email: "ivy@agentbucket.dev", Role: "user", Active: true, PasswordHash: hashPassword("user123")},
			{ID: 4, Name: "Noah", Email: "noah@agentbucket.dev", Role: "user", Active: false, PasswordHash: hashPassword("user123")},
		},
		Approvals:    []Approval{},
		ChatSessions: map[string][]ChatSession{},
		ChatMessages: map[string][]ChatMessage{},
		Deployments:  []Deployment{},
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
		if _, err := tx.Exec(`INSERT INTO users (id, name, email, role, active, password_hash) VALUES (?, ?, ?, ?, ?, ?)`, user.ID, user.Name, user.Email, user.Role, active, user.PasswordHash); err != nil {
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

func hashPassword(password string) string {
	hash := sha256.Sum256([]byte(password))
	return fmt.Sprintf("%x", hash)
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
