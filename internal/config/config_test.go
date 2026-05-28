package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestLoadUsesDotConfigDirectory(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", "")

	cfgDir := filepath.Join(home, ".config", "previously-on")
	if err := os.MkdirAll(cfgDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(cfgDir, "config.json"), []byte(`{"ai":{"enabled":false,"provider":"openai","model":"test-model"}}`), 0o644); err != nil {
		t.Fatal(err)
	}

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	if cfg.ConfigDir != cfgDir {
		t.Fatalf("expected config dir %q, got %q", cfgDir, cfg.ConfigDir)
	}
	if cfg.StateDir != filepath.Join(home, ".previously-on") {
		t.Fatalf("expected state dir to remain under ~/.previously-on, got %q", cfg.StateDir)
	}
	if cfg.AI.Enabled {
		t.Fatal("expected config file to disable AI")
	}
}

func TestLoadHonorsXDGConfigHome(t *testing.T) {
	home := t.TempDir()
	configHome := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("XDG_CONFIG_HOME", configHome)

	cfg, err := Load()
	if err != nil {
		t.Fatal(err)
	}

	want := filepath.Join(configHome, "previously-on")
	if cfg.ConfigDir != want {
		t.Fatalf("expected config dir %q, got %q", want, cfg.ConfigDir)
	}
}
