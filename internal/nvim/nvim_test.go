package nvim

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallAtCreatesPlugin(t *testing.T) {
	dataHome := t.TempDir()
	configHome := t.TempDir()
	executable := "/tmp/previously-on"

	result, err := InstallAt(dataHome, configHome, executable, StrategyNative)
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "installed" {
		t.Fatalf("action = %q, want installed", result.Action)
	}

	wantPath := filepath.Join(dataHome, "nvim", pluginRelativePath)
	if result.PluginPath != wantPath {
		t.Fatalf("plugin path = %q, want %q", result.PluginPath, wantPath)
	}
	if result.SpecPath != "" {
		t.Fatalf("spec path = %q, want empty for native strategy", result.SpecPath)
	}
	data, err := os.ReadFile(wantPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		`local executable = "/tmp/previously-on"`,
		`vim.api.nvim_create_buf(false, true)`,
		`vim.bo[summary_buf].filetype = "markdown"`,
		`"summary", "--raw", "--no-mark-seen", "--simple"`,
		`Previously on summary available. Run :PreviouslyOn or <leader>po.`,
		`vim.api.nvim_create_user_command("PreviouslyOn"`,
		`vim.keymap.set("n", "<leader>po"`,
		`vim.api.nvim_create_autocmd("VimEnter"`,
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("plugin content missing %q", want)
		}
	}
	if strings.Contains(content, "{{EXECUTABLE}}") {
		t.Fatal("plugin content still contains executable placeholder")
	}
}

func TestInstallAtUpdatesExistingPlugin(t *testing.T) {
	dataHome := t.TempDir()
	configHome := t.TempDir()

	first, err := InstallAt(dataHome, configHome, "/tmp/previously-on-old", StrategyNative)
	if err != nil {
		t.Fatal(err)
	}
	second, err := InstallAt(dataHome, configHome, "/tmp/previously-on-new", StrategyNative)
	if err != nil {
		t.Fatal(err)
	}
	if second.Action != "updated" {
		t.Fatalf("action = %q, want updated", second.Action)
	}
	if second.PluginPath != first.PluginPath {
		t.Fatalf("plugin path changed from %q to %q", first.PluginPath, second.PluginPath)
	}

	data, err := os.ReadFile(second.PluginPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	if !strings.Contains(content, "/tmp/previously-on-new") {
		t.Fatal("plugin was not updated with new executable path")
	}
	if strings.Contains(content, "/tmp/previously-on-old") {
		t.Fatal("plugin still contains old executable path")
	}
}

func TestUninstallAtRemovesManagedPlugin(t *testing.T) {
	dataHome := t.TempDir()
	configHome := t.TempDir()
	result, err := InstallAt(dataHome, configHome, "/tmp/previously-on", StrategyNative)
	if err != nil {
		t.Fatal(err)
	}

	removed, err := UninstallAt(dataHome, configHome)
	if err != nil {
		t.Fatal(err)
	}
	if removed.Action != "removed" {
		t.Fatalf("action = %q, want removed", removed.Action)
	}
	if _, err := os.Stat(filepath.Dir(filepath.Dir(result.PluginPath))); !os.IsNotExist(err) {
		t.Fatalf("managed plugin directory still exists or stat failed: %v", err)
	}
}

func TestUninstallAtWhenMissing(t *testing.T) {
	result, err := UninstallAt(t.TempDir(), t.TempDir())
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "not installed" {
		t.Fatalf("action = %q, want not installed", result.Action)
	}
}

func TestInstallAtCreatesLazySpecForLazyStrategy(t *testing.T) {
	dataHome := t.TempDir()
	configHome := t.TempDir()

	result, err := InstallAt(dataHome, configHome, "/tmp/previously-on", StrategyLazy)
	if err != nil {
		t.Fatal(err)
	}
	if result.SpecPath == "" {
		t.Fatal("expected lazy.nvim spec path")
	}

	data, err := os.ReadFile(result.SpecPath)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{
		"previously-on managed lazy.nvim spec",
		`name = "previously-on.nvim"`,
		"lazy = false",
	} {
		if !strings.Contains(content, want) {
			t.Fatalf("lazy spec missing %q", want)
		}
	}
	if strings.Contains(content, "{{PLUGIN_DIR}}") {
		t.Fatal("lazy spec still contains plugin dir placeholder")
	}

	if _, err := UninstallAt(dataHome, configHome); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(result.SpecPath); !os.IsNotExist(err) {
		t.Fatalf("lazy spec still exists or stat failed: %v", err)
	}
}

func TestInstallAtRejectsUnsupportedStrategy(t *testing.T) {
	_, err := InstallAt(t.TempDir(), t.TempDir(), "/tmp/previously-on", Strategy("packer"))
	if err == nil {
		t.Fatal("expected unsupported strategy error")
	}
	if !strings.Contains(err.Error(), "choose native or lazy") {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestUninstallAtDoesNotRemoveUnmanagedLazySpec(t *testing.T) {
	dataHome := t.TempDir()
	configHome := t.TempDir()
	specPath := filepath.Join(configHome, "nvim", lazySpecRelativePath)
	if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(specPath, []byte("return {}\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	if _, err := UninstallAt(dataHome, configHome); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(specPath); err != nil {
		t.Fatalf("unmanaged lazy spec was removed: %v", err)
	}
}
