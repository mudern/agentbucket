package main

import (
	"strings"
	"testing"
)

func TestScanSSEData(t *testing.T) {
	input := strings.Join([]string{
		"event: content_block_delta",
		"data: first",
		"",
		": keepalive",
		"data: second with spaces   ",
		"data:",
		"data: third",
		"",
	}, "\n")

	var got []string
	if err := scanSSEData(strings.NewReader(input), func(data string) {
		got = append(got, data)
	}); err != nil {
		t.Fatalf("scanSSEData returned error: %v", err)
	}

	want := []string{"first", "second with spaces", "third"}
	if len(got) != len(want) {
		t.Fatalf("expected %d items, got %d: %#v", len(want), len(got), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("item %d: expected %q, got %q", i, want[i], got[i])
		}
	}
}

func TestAnthropicTextDelta(t *testing.T) {
	data := `{"type":"content_block_delta","delta":{"type":"text_delta","text":"hello"}}`
	if got := anthropicTextDelta(data); got != "hello" {
		t.Fatalf("expected text delta, got %q", got)
	}
	if got := anthropicTextDelta(`{"type":"message_start"}`); got != "" {
		t.Fatalf("expected non-delta event to be ignored, got %q", got)
	}
	if got := anthropicTextDelta("[DONE]"); got != "" {
		t.Fatalf("expected DONE to be ignored, got %q", got)
	}
}
