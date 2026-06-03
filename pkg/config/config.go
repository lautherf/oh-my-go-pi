package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

type Config struct {
	Model         string `json:"model"`
	OllamaURL     string `json:"ollama_url"`
	Provider      string `json:"provider"`
	OpenAIKey     string `json:"openai_key"`
	AnthropicKey  string `json:"anthropic_key"`
	OpenRouterKey string `json:"openrouter_key"`
	LogLevel      string `json:"log_level"`
	LogFile       string `json:"log_file"`
}

func Defaults() *Config {
	return &Config{
		Model:     "llama3.2",
		OllamaURL: "http://localhost:11434",
		Provider:  "ollama",
		LogLevel:  "warn",
	}
}

func Load(paths ...string) (*Config, error) {
	cfg := Defaults()

	for _, p := range paths {
		if p == "" {
			continue
		}
		data, err := os.ReadFile(p)
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, fmt.Errorf("config read %s: %w", p, err)
		}
		if err := json.Unmarshal(data, cfg); err != nil {
			return nil, fmt.Errorf("config parse %s: %w", p, err)
		}
	}

	return cfg, nil
}

func Find() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}

	paths := []string{
		filepath.Join(".", ".omp", "config.json"),
		filepath.Join(".", "omp.json"),
		filepath.Join(home, ".omp", "config.json"),
	}

	for _, p := range paths {
		fi, err := os.Stat(p)
		if err == nil && !fi.IsDir() {
			return p, nil
		}
	}

	return filepath.Join(home, ".omp", "config.json"), nil
}

func ApplyEnv(cfg *Config) {
	if v := os.Getenv("OMP_MODEL"); v != "" {
		cfg.Model = v
	}
	if v := os.Getenv("OMP_OLLAMA_URL"); v != "" {
		cfg.OllamaURL = v
	}
	if v := os.Getenv("OMP_PROVIDER"); v != "" {
		cfg.Provider = v
	}
	if v := os.Getenv("OMP_OPENAI_KEY"); v != "" {
		cfg.OpenAIKey = v
	}
	if v := os.Getenv("OMP_ANTHROPIC_KEY"); v != "" {
		cfg.AnthropicKey = v
	}
	if v := os.Getenv("OMP_OPENROUTER_KEY"); v != "" {
		cfg.OpenRouterKey = v
	}
	if v := os.Getenv("OMP_LOG_LEVEL"); v != "" {
		cfg.LogLevel = v
	}
	if v := os.Getenv("OMP_LOG_FILE"); v != "" {
		cfg.LogFile = v
	}
}

func (c *Config) String() string {
	var b strings.Builder
	b.WriteString(fmt.Sprintf("model=%s\n", c.Model))
	b.WriteString(fmt.Sprintf("provider=%s\n", c.Provider))
	b.WriteString(fmt.Sprintf("ollama_url=%s\n", c.OllamaURL))
	b.WriteString(fmt.Sprintf("openai_key=%s\n", mask(c.OpenAIKey)))
	b.WriteString(fmt.Sprintf("anthropic_key=%s\n", mask(c.AnthropicKey)))
	b.WriteString(fmt.Sprintf("openrouter_key=%s\n", mask(c.OpenRouterKey)))
	b.WriteString(fmt.Sprintf("log_level=%s\n", c.LogLevel))
	b.WriteString(fmt.Sprintf("log_file=%s\n", c.LogFile))
	return b.String()
}

func mask(s string) string {
	if len(s) <= 8 {
		return "***"
	}
	return s[:4] + "..." + s[len(s)-4:]
}
