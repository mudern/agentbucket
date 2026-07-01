package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestCopySelectedSkillsRequiresSkillMD(t *testing.T) {
	root := t.TempDir()
	src := filepath.Join(root, "skills")
	dst := filepath.Join(root, "out")
	mustWriteFile(t, filepath.Join(src, "valid", "SKILL.md"), "---\nname: valid\n---\n")
	if err := copySelectedSkills(src, dst, []string{"valid"}); err != nil {
		t.Fatalf("valid skill copy failed: %v", err)
	}
	if _, err := os.Stat(filepath.Join(dst, "valid", "SKILL.md")); err != nil {
		t.Fatalf("copied skill missing: %v", err)
	}
	if err := copySelectedSkills(src, dst, []string{"missing"}); err == nil {
		t.Fatal("expected missing SKILL.md error")
	}
}

func mustWriteFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}
