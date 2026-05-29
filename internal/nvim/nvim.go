package nvim

import (
	_ "embed"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
)

const (
	pluginRelativePath   = "site/pack/previously-on/start/previously-on.nvim/plugin/previously_on.lua"
	lazySpecRelativePath = "lua/plugins/previously-on.lua"
)

//go:embed templates/previously_on.lua
var pluginTemplate string

//go:embed templates/lazy_spec.lua
var lazySpecTemplate string

type Strategy string

const (
	StrategyNative Strategy = "native"
	StrategyLazy   Strategy = "lazy"
)

type Result struct {
	PluginPath string
	SpecPath   string
	Action     string
}

func Install(executable string) (Result, error) {
	return InstallAt(dataHome(), configHome(), executable, StrategyNative)
}

func InstallAt(dataHomePath, configHomePath, executable string, strategy Strategy) (Result, error) {
	if err := validateStrategy(strategy); err != nil {
		return Result{}, err
	}
	pluginPath := filepath.Join(dataHomePath, "nvim", pluginRelativePath)
	pluginDir := filepath.Dir(filepath.Dir(pluginPath))
	if err := os.MkdirAll(filepath.Dir(pluginPath), 0o755); err != nil {
		return Result{}, err
	}

	action := "installed"
	if _, err := os.Stat(pluginPath); err == nil {
		action = "updated"
	} else if err != nil && !os.IsNotExist(err) {
		return Result{}, err
	}

	if err := os.WriteFile(pluginPath, []byte(pluginSource(executable)), 0o644); err != nil {
		return Result{}, err
	}

	result := Result{PluginPath: pluginPath, Action: action}
	specPath := filepath.Join(configHomePath, "nvim", lazySpecRelativePath)
	if strategy == StrategyNative {
		if err := removeLazySpec(specPath); err != nil {
			return Result{}, err
		}
		return result, nil
	}
	if strategy == StrategyLazy {
		if err := os.MkdirAll(filepath.Dir(specPath), 0o755); err != nil {
			return Result{}, err
		}
		if err := os.WriteFile(specPath, []byte(lazySpecSource(pluginDir)), 0o644); err != nil {
			return Result{}, err
		}
		result.SpecPath = specPath
	}
	return result, nil
}

func Uninstall() (Result, error) {
	return UninstallAt(dataHome(), configHome())
}

func UninstallAt(dataHomePath, configHomePath string) (Result, error) {
	pluginDir := filepath.Join(dataHomePath, "nvim", "site/pack/previously-on/start/previously-on.nvim")
	pluginPath := filepath.Join(pluginDir, "plugin/previously_on.lua")
	specPath := filepath.Join(configHomePath, "nvim", lazySpecRelativePath)
	if err := removeLazySpec(specPath); err != nil {
		return Result{}, err
	}
	if _, err := os.Stat(pluginDir); os.IsNotExist(err) {
		return Result{PluginPath: pluginPath, SpecPath: specPath, Action: "not installed"}, nil
	} else if err != nil {
		return Result{}, err
	}
	if err := os.RemoveAll(pluginDir); err != nil {
		return Result{}, err
	}
	return Result{PluginPath: pluginPath, SpecPath: specPath, Action: "removed"}, nil
}

func dataHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_DATA_HOME")); value != "" {
		return value
	}
	if runtime.GOOS == "darwin" {
		if home, err := os.UserHomeDir(); err == nil {
			return filepath.Join(home, ".local", "share")
		}
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".local", "share")
	}
	return "."
}

func configHome() string {
	if value := strings.TrimSpace(os.Getenv("XDG_CONFIG_HOME")); value != "" {
		return value
	}
	if home, err := os.UserHomeDir(); err == nil {
		return filepath.Join(home, ".config")
	}
	return "."
}

func validateStrategy(strategy Strategy) error {
	switch strategy {
	case StrategyNative, StrategyLazy:
		return nil
	default:
		return fmt.Errorf("unsupported Neovim install strategy %q (choose native or lazy)", strategy)
	}
}

func removeLazySpec(specPath string) error {
	data, err := os.ReadFile(specPath)
	if os.IsNotExist(err) {
		return nil
	}
	if err != nil {
		return err
	}
	if !strings.Contains(string(data), "previously-on managed lazy.nvim spec") {
		return nil
	}
	return os.Remove(specPath)
}

func luaString(value string) string {
	return fmt.Sprintf("%q", value)
}

func pluginSource(executable string) string {
	return strings.ReplaceAll(pluginTemplate, "{{EXECUTABLE}}", luaString(executable))
}

func lazySpecSource(pluginDir string) string {
	return strings.ReplaceAll(lazySpecTemplate, "{{PLUGIN_DIR}}", luaString(pluginDir))
}
