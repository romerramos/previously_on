package config

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
)

type Config struct {
	SchemaVersion int           `json:"schemaVersion"`
	ConfigDir     string        `json:"configDir"`
	StateDir      string        `json:"stateDir"`
	AI            AIConfig      `json:"ai"`
	Privacy       PrivacyConfig `json:"privacy"`
	Output        OutputConfig  `json:"output"`
}

type AIConfig struct {
	Enabled  bool              `json:"enabled"`
	Provider string            `json:"provider"`
	Model    string            `json:"model"`
	Models   map[string]string `json:"models,omitempty"`
}

type PrivacyConfig struct {
	SendCommitMessages bool `json:"sendCommitMessages"`
	SendCommitBodies   bool `json:"sendCommitBodies"`
	SendFilePaths      bool `json:"sendFilePaths"`
	SendDiffStats      bool `json:"sendDiffStats"`
	SendSmallDiffs     bool `json:"sendSmallDiffs"`
	MaxCommits         int  `json:"maxCommits"`
	MaxFilePaths       int  `json:"maxFilePaths"`
	MaxDiffBytes       int  `json:"maxDiffBytes"`
}

type OutputConfig struct {
	DefaultWriteFile bool `json:"defaultWriteFile"`
}

func Load() (Config, error) {
	cfg, err := defaultConfig()
	if err != nil {
		return Config{}, err
	}

	data, err := os.ReadFile(filepath.Join(cfg.ConfigDir, "config.json"))
	if errors.Is(err, os.ErrNotExist) {
		return cfg, nil
	}
	if err != nil {
		return Config{}, err
	}
	if err := json.Unmarshal(data, &cfg); err != nil {
		return Config{}, err
	}
	if os.Getenv("PREVIOUSLY_ON_NO_AI") == "1" {
		cfg.AI.Enabled = false
	}
	return cfg, nil
}

func Save(cfg Config) error {
	if err := os.MkdirAll(cfg.ConfigDir, 0o700); err != nil {
		return err
	}
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(cfg.ConfigDir, "config.json"), append(data, '\n'), 0o600)
}

func defaultConfig() (Config, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return Config{}, err
	}
	configHome := os.Getenv("XDG_CONFIG_HOME")
	if configHome == "" {
		configHome = filepath.Join(home, ".config")
	}
	return Config{
		SchemaVersion: 1,
		ConfigDir:     filepath.Join(configHome, "previously-on"),
		StateDir:      filepath.Join(home, ".previously-on"),
		AI: AIConfig{
			Enabled:  true,
			Provider: "openai",
			Model:    "gpt-4.1-mini",
			Models: map[string]string{
				"openai":     "gpt-4.1-mini",
				"anthropic":  "claude-3-5-haiku-latest",
				"gemini":     "gemini-2.5-flash",
				"openrouter": "openai/gpt-4o-mini",
				"ollama":     "llama3.2",
			},
		},
		Privacy: PrivacyConfig{
			SendCommitMessages: true,
			SendCommitBodies:   false,
			SendFilePaths:      true,
			SendDiffStats:      true,
			SendSmallDiffs:     false,
			MaxCommits:         100,
			MaxFilePaths:       300,
			MaxDiffBytes:       12000,
		},
	}, nil
}
