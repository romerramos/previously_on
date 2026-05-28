package hooks

import (
	"bytes"
	_ "embed"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

const (
	managedStart = "# >>> previously-on"
	managedEnd   = "# <<< previously-on"
)

//go:embed templates/git_hook.sh
var gitHookTemplate string

var gitHookNames = []string{"post-merge", "post-rewrite"}

type Result struct {
	HookPath string
	HookName string
	Action   string
	Warning  string
}

func InstallGitHooks(repoRoot, executable string) ([]Result, error) {
	hooksDir, err := gitHooksDir(repoRoot)
	if err != nil {
		return nil, err
	}
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		return nil, err
	}

	block := managedBlock(executable)
	var results []Result
	for _, name := range gitHookNames {
		path := filepath.Join(hooksDir, name)
		result, err := installBlock(path, name, block)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func UninstallGitHooks(repoRoot string) ([]Result, error) {
	hooksDir, err := gitHooksDir(repoRoot)
	if err != nil {
		return nil, err
	}

	var results []Result
	for _, name := range gitHookNames {
		path := filepath.Join(hooksDir, name)
		result, err := uninstallBlock(path, name)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}
	return results, nil
}

func installBlock(path, name, block string) (Result, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		content := "#!/bin/sh\n" + block
		if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
			return Result{}, err
		}
		return Result{HookPath: path, HookName: name, Action: "installed"}, nil
	}
	if err != nil {
		return Result{}, err
	}

	content := string(data)
	action := "updated"
	warning := ""
	if strings.Contains(content, managedStart) && strings.Contains(content, managedEnd) {
		content = replaceManagedBlock(content, block)
	} else {
		action = "appended"
		warning = "Existing hook found; appended previously-on block. If the existing hook exits early, previously-on may not run."
		content = strings.TrimRight(content, "\n") + "\n\n" + block
	}
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		return Result{}, err
	}
	if err := os.Chmod(path, 0o755); err != nil {
		return Result{}, err
	}
	return Result{HookPath: path, HookName: name, Action: action, Warning: warning}, nil
}

func uninstallBlock(path, name string) (Result, error) {
	data, err := os.ReadFile(path)
	if os.IsNotExist(err) {
		return Result{HookPath: path, HookName: name, Action: "not installed"}, nil
	}
	if err != nil {
		return Result{}, err
	}
	content := removeManagedBlock(string(data))
	if content == string(data) {
		return Result{HookPath: path, HookName: name, Action: "not installed"}, nil
	}
	if strings.TrimSpace(content) == "" || strings.TrimSpace(content) == "#!/bin/sh" {
		if err := os.Remove(path); err != nil {
			return Result{}, err
		}
		return Result{HookPath: path, HookName: name, Action: "removed file"}, nil
	}
	if err := os.WriteFile(path, []byte(strings.TrimRight(content, "\n")+"\n"), 0o755); err != nil {
		return Result{}, err
	}
	return Result{HookPath: path, HookName: name, Action: "removed block"}, nil
}

func managedBlock(executable string) string {
	replacer := strings.NewReplacer(
		"{{MANAGED_START}}", managedStart,
		"{{EXECUTABLE}}", shellQuote(executable),
		"{{MANAGED_END}}", managedEnd,
	)
	return replacer.Replace(gitHookTemplate)
}

func replaceManagedBlock(content, block string) string {
	start := strings.Index(content, managedStart)
	end := strings.Index(content, managedEnd)
	if start == -1 || end == -1 || end < start {
		return content
	}
	end += len(managedEnd)
	return strings.TrimRight(content[:start], "\n") + "\n" + strings.TrimRight(block, "\n") + "\n" + strings.TrimLeft(content[end:], "\n")
}

func removeManagedBlock(content string) string {
	start := strings.Index(content, managedStart)
	end := strings.Index(content, managedEnd)
	if start == -1 || end == -1 || end < start {
		return content
	}
	end += len(managedEnd)
	return strings.TrimRight(content[:start], "\n") + "\n" + strings.TrimLeft(content[end:], "\n")
}

func shellQuote(value string) string {
	return "'" + strings.ReplaceAll(value, "'", "'\\''") + "'"
}

func gitHooksDir(repoRoot string) (string, error) {
	cmd := exec.Command("git", "rev-parse", "--git-path", "hooks")
	cmd.Dir = repoRoot
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git rev-parse --git-path hooks: %w: %s", err, strings.TrimSpace(stderr.String()))
	}
	path := strings.TrimSpace(stdout.String())
	if !filepath.IsAbs(path) {
		path = filepath.Join(repoRoot, path)
	}
	return path, nil
}
