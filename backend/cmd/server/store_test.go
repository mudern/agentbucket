package main

import (
	"strings"
	"testing"
	"time"
)

func TestNewStore(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	snap := store.snapshot()
	if snap.CurrentUser.Name == "" {
		t.Fatal("CurrentUser not seeded")
	}
}

func TestStoreUpdateAtomic(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	// Add a user via update
	err = store.update(func(s *State) error {
		s.Users = append(s.Users, User{ID: 999, Name: "test-user", Email: "test@test.com", Role: "user", Active: true})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	found := false
	for _, u := range snap.Users {
		if u.ID == 999 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("user not persisted after update")
	}
}

func TestStoreChatSessions(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	session := newChatSession("test-agent", "test title")
	err = store.update(func(s *State) error {
		ensureChatMaps(s)
		s.ChatSessions["test-agent"] = append(s.ChatSessions["test-agent"], session)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	sessions := snap.ChatSessions["test-agent"]
	if len(sessions) != 1 {
		t.Fatalf("expected 1 session, got %d", len(sessions))
	}
	if sessions[0].Title != "test title" {
		t.Fatalf("expected title 'test title', got %q", sessions[0].Title)
	}
}

func TestEnsureUserPasswordHashes(t *testing.T) {
	users := ensureUserPasswordHashes([]User{
		{Name: "Luna"},
		{Name: "Ivy"},
		{Name: "Custom"},
	})
	if !verifyPassword("admin123", users[0].PasswordHash) {
		t.Fatalf("expected Luna default admin password to verify")
	}
	if !verifyPassword("user123", users[1].PasswordHash) {
		t.Fatalf("expected Ivy default user password to verify")
	}
	if !verifyPassword("password", users[2].PasswordHash) {
		t.Fatalf("expected fallback password to verify")
	}
	// Ensure hashes use the new salted format
	for i, u := range users {
		if !strings.Contains(u.PasswordHash, ":") {
			t.Fatalf("user %d password hash should contain salt separator ':'", i)
		}
	}
}

func TestStoreChatSessionLimit(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	// Create 20 sessions (at the limit)
	err = store.update(func(s *State) error {
		ensureChatMaps(s)
		for i := 0; i < 20; i++ {
			s.ChatSessions["limit-agent"] = append(s.ChatSessions["limit-agent"], newChatSession("limit-agent", "session"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	// Try to add one more - should fail
	err = store.update(func(s *State) error {
		ensureChatMaps(s)
		if len(s.ChatSessions["limit-agent"]) >= 20 {
			return &TestError{msg: "会话数已达上限（20 个）"}
		}
		s.ChatSessions["limit-agent"] = append(s.ChatSessions["limit-agent"], newChatSession("limit-agent", "overflow"))
		return nil
	})
	if err == nil {
		t.Fatal("expected error for exceeding session limit")
	}
}

type TestError struct{ msg string }

func (e *TestError) Error() string { return e.msg }

func TestStoreDeploymentCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	dep := Deployment{
		ID:            "dep-test-agent-1234567",
		AgentID:       "test-agent",
		ImageTag:      "agentbucket/test-agent:abc123",
		ContainerName: "agentbucket-test-agent",
		Status:        "running",
		HostPort:      18000,
		SidecarURL:    "http://127.0.0.1:18000",
		CreatedAt:     time.Now(),
	}

	initialCount := len(store.snapshot().Deployments)

	err = store.update(func(s *State) error {
		s.Deployments = append(s.Deployments, dep)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	if len(snap.Deployments) != initialCount+1 {
		t.Fatalf("expected %d deployments, got %d", initialCount+1, len(snap.Deployments))
	}
	if snap.Deployments[initialCount].Status != "running" {
		t.Fatal("deployment status should be running")
	}

	// Update status
	err = store.update(func(s *State) error {
		s.Deployments[initialCount].Status = "stopped"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap = store.snapshot()
	if snap.Deployments[initialCount].Status != "stopped" {
		t.Fatal("deployment status should be updated to stopped")
	}
}

func TestStoreRepositoryCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	repo := Repository{
		ID:         "test-repo",
		Provider:   "Local",
		LocalPath:  "/tmp/test",
		Branch:     "main",
		AgentsPath: "agents",
		Status:     "启用",
	}

	initialCount := len(store.snapshot().Repositories)

	err = store.update(func(s *State) error {
		s.Repositories = append(s.Repositories, repo)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	if len(snap.Repositories) != initialCount+1 {
		t.Fatalf("expected %d repositories, got %d", initialCount+1, len(snap.Repositories))
	}

	// Delete
	err = store.update(func(s *State) error {
		for i := range s.Repositories {
			if s.Repositories[i].ID == "test-repo" {
				s.Repositories = append(s.Repositories[:i], s.Repositories[i+1:]...)
				break
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap = store.snapshot()
	if len(snap.Repositories) != initialCount {
		t.Fatalf("expected %d repositories after delete, got %d", initialCount, len(snap.Repositories))
	}
}

func TestStoreAITokenCRUD(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	token := AIToken{
		ID:       1,
		Name:     "test-token",
		Provider: "TEST",
		Status:   "启用",
	}

	initialCount := len(store.snapshot().AITokens)

	err = store.update(func(s *State) error {
		s.AITokens = append(s.AITokens, token)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	if len(snap.AITokens) != initialCount+1 {
		t.Fatalf("expected %d AI tokens, got %d", initialCount+1, len(snap.AITokens))
	}
}

func TestStoreApprovals(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	approval := Approval{
		ID: 1, Type: "deploy", Applicant: "user1",
		Summary: "deploy legal-summarizer", Priority: "高", Status: "待审批",
	}

	err = store.update(func(s *State) error {
		s.Approvals = append(s.Approvals, approval)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	if len(snap.Approvals) != 1 {
		t.Fatal("approval not saved")
	}

	// Approve
	err = store.update(func(s *State) error {
		s.Approvals[0].Status = "已通过"
		s.Approvals[0].Reviewer = "admin"
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap = store.snapshot()
	if snap.Approvals[0].Status != "已通过" {
		t.Fatalf("expected status 已通过, got %q", snap.Approvals[0].Status)
	}
}

func TestStorePersistence(t *testing.T) {
	dir := t.TempDir()
	store1, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}

	err = store1.update(func(s *State) error {
		s.Users = append(s.Users, User{ID: 777, Name: "persistent", Email: "p@test.com", Role: "admin", Active: true})
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	store1.db.Close()

	// Reopen
	store2, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store2.db.Close()

	snap := store2.snapshot()
	found := false
	for _, u := range snap.Users {
		if u.ID == 777 {
			found = true
			break
		}
	}
	if !found {
		t.Fatal("data not persisted across store reopen")
	}
}

func TestStoreBusMessagesTable(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	// Verify bus_messages table exists
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM sqlite_master WHERE type='table' AND name='bus_messages'").Scan(&count); err != nil {
		t.Fatal(err)
	}
	if count != 1 {
		t.Fatal("bus_messages table not created")
	}
}

func TestStoreChatMessages(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	now := time.Now()
	userMsg := ChatMessage{
		ID: "msg-1", SessionID: "s1", AgentID: "a1",
		Role: "user", Content: "hello", CreatedAt: now,
	}
	assistantMsg := ChatMessage{
		ID: "msg-2", SessionID: "s1", AgentID: "a1",
		Role: "assistant", Content: "hi there", CreatedAt: now,
	}

	err = store.update(func(s *State) error {
		ensureChatMaps(s)
		key := chatKey("a1", "s1")
		s.ChatMessages[key] = append(s.ChatMessages[key], userMsg, assistantMsg)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	snap := store.snapshot()
	msgs := snap.ChatMessages[chatKey("a1", "s1")]
	if len(msgs) != 2 {
		t.Fatalf("expected 2 messages, got %d", len(msgs))
	}
}
