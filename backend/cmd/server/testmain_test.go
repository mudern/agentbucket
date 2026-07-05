package main

import (
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	providersDir, err := os.MkdirTemp("", "agentbucket-empty-providers-*")
	if err != nil {
		panic(err)
	}
	_ = os.Setenv("AGENTBUCKET_PROVIDERS_DIR", providersDir)
	code := m.Run()
	_ = os.RemoveAll(providersDir)
	os.Exit(code)
}
