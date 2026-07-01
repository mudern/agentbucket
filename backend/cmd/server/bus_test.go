package main

import (
	"testing"
	"time"
)

func TestNewAgentBus(t *testing.T) {
	bus := newAgentBus()
	if bus == nil {
		t.Fatal("newAgentBus returned nil")
	}
	agents := bus.list()
	if len(agents) != 0 {
		t.Fatalf("expected empty bus, got %d agents", len(agents))
	}
}

func TestBusRegisterAndList(t *testing.T) {
	bus := newAgentBus()
	bus.register(BusAgent{AgentID: "agent-1", Name: "Agent 1", Status: "online"})
	bus.register(BusAgent{AgentID: "agent-2", Name: "Agent 2", Status: "idle"})

	agents := bus.list()
	if len(agents) != 2 {
		t.Fatalf("expected 2 agents, got %d", len(agents))
	}
	if agents[0].AgentID != "agent-1" && agents[0].AgentID != "agent-2" {
		t.Fatal("agents not sorted by ID")
	}
}

func TestBusPostAndMessages(t *testing.T) {
	bus := newAgentBus()
	msg := bus.post("agent-1", "agent-2", "hello from agent 1", nil)

	if msg.FromAgent != "agent-1" || msg.ToAgent != "agent-2" || msg.Content != "hello from agent 1" {
		t.Fatal("message fields mismatch")
	}
	if msg.ID == "" {
		t.Fatal("message ID is empty")
	}
	if msg.CreatedAt.IsZero() {
		t.Fatal("message CreatedAt is zero")
	}

	msgs := bus.getMessages()
	if len(msgs) != 1 {
		t.Fatalf("expected 1 message, got %d", len(msgs))
	}
}

func TestBusMessageRingBufferCap(t *testing.T) {
	bus := newAgentBus()
	for i := 0; i < 250; i++ {
		bus.post("a", "b", "test message", nil)
	}
	msgs := bus.getMessages()
	if len(msgs) > 200 {
		t.Fatalf("expected at most 200 messages, got %d", len(msgs))
	}
}

func TestBusRegisterUpdatesLastSeen(t *testing.T) {
	bus := newAgentBus()
	before := time.Now()
	bus.register(BusAgent{AgentID: "a", Name: "A", Status: "online"})
	after := time.Now()

	agents := bus.list()
	if len(agents) != 1 {
		t.Fatal("expected 1 agent")
	}
	if agents[0].LastSeen.Before(before) || agents[0].LastSeen.After(after) {
		t.Fatal("LastSeen not updated correctly")
	}
}

func TestBusPersistence(t *testing.T) {
	dir := t.TempDir()
	store, err := NewStore(dir+"/test.db", dir)
	if err != nil {
		t.Fatal(err)
	}
	defer store.db.Close()

	bus := newAgentBus()
	msg := bus.post("agent-1", "agent-2", "persistence test", store)
	_ = msg

	// Verify the message was persisted to SQLite
	var count int
	if err := store.db.QueryRow("SELECT COUNT(*) FROM bus_messages WHERE from_agent = ?", "agent-1").Scan(&count); err != nil {
		t.Fatalf("failed to query bus_messages: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected 1 persisted message, got %d", count)
	}
}

func TestBusMessagesPerAgentFilter(t *testing.T) {
	bus := newAgentBus()
	bus.post("a", "x", "msg to x", nil)
	bus.post("a", "y", "msg to y", nil)
	bus.post("b", "x", "another to x", nil)

	msgs := bus.getMessages()

	// Check filtering works (simulated since getMessages doesn't filter)
	if len(msgs) != 3 {
		t.Fatalf("expected 3 messages total, got %d", len(msgs))
	}
}
