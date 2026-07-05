package main

import (
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"time"

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
	// Check if users already exist in SQL table before seeding
	existingUsers, _ := store.loadUsers()
	if len(raw) == 0 && len(existingUsers) == 0 {
		store.state = seedState(rootDir)
		store.state.Users = ensureUserPasswordHashes(store.state.Users)
		printFirstStartCredentials(store.state.Users)
		store.importProviderTokens()
		return store.saveLocked()
	}
	if len(raw) == 0 {
		// DB has users but no state JSON — rebuild state from SQL tables
		store.state = seedState(rootDir)
		store.state.Users = existingUsers
		return store.saveLocked()
	}
	if err := json.Unmarshal(raw, &store.state); err != nil {
		return nil, err
	}
	// Always reload users from SQL table on startup (more reliable than JSON state)
	users, err := store.loadUsers()
	if err == nil && len(users) > 0 {
		store.state.Users = users
	}
	if err := store.loadChat(); err != nil {
		return nil, err
	}
	if _, err := store.saveLocked(); err != nil {
		return nil, err
	}
	store.importProviderTokens()
	return store, nil
}

func (s *Store) importProviderTokens() {
	dir := os.Getenv("AGENTBUCKET_PROVIDERS_DIR")
	if dir == "" {
		home, err := os.UserHomeDir()
		if err != nil { return }
		dir = filepath.Join(home, ".config", "ccs", "providers")
	}
	entries, err := os.ReadDir(dir)
	if err != nil { return }
	existing := map[string]bool{}
	for _, t := range s.state.AITokens { existing[t.Name] = true }
	nextID := 1
	for _, t := range s.state.AITokens { if t.ID >= nextID { nextID = t.ID + 1 } }
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".env") { continue }
		name := strings.TrimSuffix(entry.Name(), ".env")
		if existing[name] { continue }
		vals := readEnvFile(filepath.Join(dir, entry.Name()))
		if len(vals) == 0 { continue }
		s.state.AITokens = append(s.state.AITokens, AIToken{
			ID: nextID, Name: name, Provider: strings.ToUpper(name),
			Status: "启用", Scope: "imported", Usage: "local",
			BaseURL: vals["ANTHROPIC_BASE_URL"],
			Model:   vals["ANTHROPIC_MODEL"],
			Secret:  vals["ANTHROPIC_AUTH_TOKEN"],
		})
		nextID++
	}
}

func readEnvFile(path string) map[string]string {
	raw, err := os.ReadFile(path)
	if err != nil { return nil }
	vals := map[string]string{}
	for _, line := range strings.Split(string(raw), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") { continue }
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 { continue }
		vals[strings.TrimSpace(parts[0])] = strings.Trim(strings.TrimSpace(parts[1]), `"'`)
	}
	return vals
}

func (s *Store) initSchema() error {
	statements := []string{
		`CREATE TABLE IF NOT EXISTS app_state (key TEXT PRIMARY KEY, value TEXT NOT NULL)`,
		`CREATE TABLE IF NOT EXISTS users (
			id INTEGER PRIMARY KEY,
			name TEXT NOT NULL,
			email TEXT NOT NULL DEFAULT '',
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
	// Migration: add password_hash column if missing (safe to run)
	var colCount int
	if err := s.db.QueryRow(`SELECT COUNT(*) FROM pragma_table_info('users') WHERE name = 'password_hash'`).Scan(&colCount); err == nil && colCount == 0 {
		if _, err := s.db.Exec(`ALTER TABLE users ADD COLUMN password_hash TEXT NOT NULL DEFAULT ''`); err != nil {
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

func printFirstStartCredentials(users []User) {
	for _, u := range users {
		if u.Role == "super_admin" {
			log.Println("")
			log.Println("First start complete. Admin credentials saved to DB.")
			log.Println("Username: admin")
			break
		}
	}
}

func ensureUserPasswordHashes(users []User) []User {
	upgraded := 0
	for i := range users {
		if users[i].PasswordHash == "" {
			users[i].PasswordHash = hashPassword("password")
		} else if !strings.Contains(users[i].PasswordHash, ":") {
			users[i].PasswordHash = hashPassword("password")
			upgraded++
		}
	}
	if upgraded > 0 {
		log.Printf("upgraded %d legacy password hashes to salted format", upgraded)
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

func seedState(rootDir string) State {
	adminPass := randomPassword(16)
	userPass := randomPassword(12)
	log.Printf("")
	log.Printf("╔══════════════════════════════════════════╗")
	log.Printf("║        AgentBucket first start            ║")
	log.Printf("╠══════════════════════════════════════════╣")
	log.Printf("║  admin:    %-30s║", adminPass)
	log.Printf("║  user:     %-30s║", userPass)
	log.Printf("╚══════════════════════════════════════════╝")
	log.Printf("")

	return State{
		CurrentUser: CurrentUser{ID: "u-1001", Name: "admin", Role: "super_admin"},
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
		AITokens: []AIToken{},
		AuthTokens: []AuthToken{
			{ID: 101, Name: "GitHub Token", Description: "访问 GitHub 仓库、Issues、PR", Secret: "ghp_demo_placeholder", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 102, Name: "Notion API Key", Description: "Notion 知识库读写权限", Secret: "ntn_demo_placeholder", Status: "启用", UpdatedAt: "刚刚"},
			{ID: 103, Name: "Internal DB", Description: "内部数据库只读访问凭据", Secret: "db_demo_placeholder", Status: "启用", UpdatedAt: "刚刚"},
		},
		Users: []User{
			{ID: 1, Name: "admin", Role: "super_admin", Active: true, PasswordHash: hashPassword(adminPass)},
			{ID: 2, Name: "user", Role: "user", Active: true, PasswordHash: hashPassword(userPass)},
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
	// Use INSERT OR REPLACE instead of DELETE + INSERT for users
	keepUserIDs := map[int]bool{}
	for _, user := range s.state.Users {
		active := 0
		if user.Active {
			active = 1
		}
		if _, err := tx.Exec(
			`INSERT OR REPLACE INTO users (id, name, email, role, active, password_hash) VALUES (?, ?, ?, ?, ?, ?)`,
			user.ID, user.Name, user.Email, user.Role, active, user.PasswordHash,
		); err != nil {
			_ = tx.Rollback()
			return nil, err
		}
		keepUserIDs[user.ID] = true
		log.Printf("[SAVE] inserted user id=%d name=%s", user.ID, user.Name)
	}
	// Use INSERT OR REPLACE for chat sessions
	keepSessionIDs := map[string]bool{}
	for _, sessions := range s.state.ChatSessions {
		for _, session := range sessions {
			if _, err := tx.Exec(
				`INSERT OR REPLACE INTO chat_sessions (id, agent_id, title, preview, created_at, updated_at) VALUES (?, ?, ?, ?, ?, ?)`,
				session.ID, session.AgentID, session.Title, session.Preview,
				formatStoredTime(session.CreatedAt), formatStoredTime(session.UpdatedAt),
			); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
			keepSessionIDs[session.ID] = true
		}
	}
	// Use INSERT OR REPLACE for chat messages
	keepMessageIDs := map[string]bool{}
	for _, messages := range s.state.ChatMessages {
		for _, message := range messages {
			if _, err := tx.Exec(
				`INSERT OR REPLACE INTO chat_messages (id, session_id, agent_id, role, content, created_at) VALUES (?, ?, ?, ?, ?, ?)`,
				message.ID, message.SessionID, message.AgentID, message.Role, message.Content,
				formatStoredTime(message.CreatedAt),
			); err != nil {
				_ = tx.Rollback()
				return nil, err
			}
			keepMessageIDs[message.ID] = true
		}
	}
	// Clean up stale records not in current state
	if len(keepUserIDs) > 0 {
		rows, err := tx.Query(`SELECT id FROM users`)
		if err == nil {
			var staleIDs []string
			for rows.Next() {
				var id int
				if err := rows.Scan(&id); err == nil && !keepUserIDs[id] && id > 2 {
					staleIDs = append(staleIDs, fmt.Sprintf("%d", id))
				}
			}
			rows.Close()
			if len(staleIDs) > 0 {
				tx.Exec(fmt.Sprintf(`DELETE FROM users WHERE id IN (%s)`, strings.Join(staleIDs, ",")))
			}
		}
	}
	if len(keepSessionIDs) > 0 {
		rows, err := tx.Query(`SELECT id FROM chat_sessions`)
		if err == nil {
			var staleIDs []string
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil && !keepSessionIDs[id] {
					staleIDs = append(staleIDs, fmt.Sprintf("'%s'", strings.ReplaceAll(id, "'", "''")))
				}
			}
			rows.Close()
			if len(staleIDs) > 0 {
				tx.Exec(fmt.Sprintf(`DELETE FROM chat_sessions WHERE id IN (%s)`, strings.Join(staleIDs, ",")))
			}
		}
	}
	if len(keepMessageIDs) > 0 {
		rows, err := tx.Query(`SELECT id FROM chat_messages`)
		if err == nil {
			var staleIDs []string
			for rows.Next() {
				var id string
				if err := rows.Scan(&id); err == nil && !keepMessageIDs[id] {
					staleIDs = append(staleIDs, fmt.Sprintf("'%s'", strings.ReplaceAll(id, "'", "''")))
				}
			}
			rows.Close()
			if len(staleIDs) > 0 {
				tx.Exec(fmt.Sprintf(`DELETE FROM chat_messages WHERE id IN (%s)`, strings.Join(staleIDs, ",")))
			}
		}
	}
	if err := tx.Commit(); err != nil {
		return nil, err
	}
	return s, nil
}

func (s *Store) snapshot() State {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.state
}

func randomPassword(length int) string {
	const chars = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	b := make([]byte, length)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("ab%d", time.Now().UnixNano()%1000000)
	}
	for i := range b {
		b[i] = chars[int(b[i])%len(chars)]
	}
	return string(b)
}

func hashPassword(password string) string {
	salt := make([]byte, 16)
	if _, err := rand.Read(salt); err != nil {
		// Fallback to a deterministic salt if crypto/rand fails
		h := sha256.Sum256([]byte("agentbucket-fallback-salt-" + password))
		copy(salt, h[:16])
	}
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(salt) + ":" + hex.EncodeToString(hash[:])
}

func verifyPassword(password, stored string) bool {
	parts := strings.SplitN(stored, ":", 2)
	if len(parts) != 2 {
		// Legacy unsalted hash
		return hashPasswordLegacy(password) == stored
	}
	salt, err := hex.DecodeString(parts[0])
	if err != nil {
		return false
	}
	expectedHash, err := hex.DecodeString(parts[1])
	if err != nil {
		return false
	}
	hash := sha256.Sum256(append(salt, []byte(password)...))
	return hex.EncodeToString(hash[:]) == hex.EncodeToString(expectedHash)
}

func hashPasswordLegacy(password string) string {
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
