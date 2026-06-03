package utils

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadEnv(t *testing.T) {
	dir := t.TempDir()
	envFile := filepath.Join(dir, ".env")

	content := `
# comment
OPENAI_API_KEY=sk-test123
ANTHROPIC_API_KEY="sk-ant-test"
OPENROUTER_API_KEY='sk-or-test'
EMPTY=
`
	os.WriteFile(envFile, []byte(content), 0644)

	if err := LoadEnv(envFile); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		key, want string
	}{
		{"OPENAI_API_KEY", "sk-test123"},
		{"ANTHROPIC_API_KEY", "sk-ant-test"},
		{"OPENROUTER_API_KEY", "sk-or-test"},
	}

	for _, tt := range tests {
		got := os.Getenv(tt.key)
		if got != tt.want {
			t.Fatalf("expected %s=%q, got %q", tt.key, tt.want, got)
		}
	}

	if v := os.Getenv("EMPTY"); v != "" {
		t.Fatalf("expected EMPTY to be unset, got %q", v)
	}
}

func TestLoadEnv_NotExist(t *testing.T) {
	err := LoadEnv("/nonexistent/.env")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}
