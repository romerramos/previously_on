package hooks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestInstallBlockAppendsAndReplacesManagedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "post-merge")
	if err := os.WriteFile(path, []byte("#!/bin/sh\necho existing\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := installBlock(path, "post-merge", managedBlock("/tmp/previously-on"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "appended" || result.Warning == "" {
		t.Fatalf("expected append warning, got %#v", result)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), managedStart) != 1 {
		t.Fatalf("expected one managed block:\n%s", string(data))
	}

	result, err = installBlock(path, "post-merge", managedBlock("/tmp/previously-on-new"))
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "updated" {
		t.Fatalf("expected update, got %#v", result)
	}
	data, err = os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(data), managedStart) != 1 || !strings.Contains(string(data), "/tmp/previously-on-new") {
		t.Fatalf("expected replaced managed block:\n%s", string(data))
	}
}

func TestManagedBlockRendersTemplate(t *testing.T) {
	block := managedBlock("/tmp/previously-on")
	for _, unwanted := range []string{"{{MANAGED_START}}", "{{EXECUTABLE}}", "{{MANAGED_END}}"} {
		if strings.Contains(block, unwanted) {
			t.Fatalf("managed block still contains placeholder %q:\n%s", unwanted, block)
		}
	}
	if !strings.Contains(block, managedStart) || !strings.Contains(block, managedEnd) || !strings.Contains(block, "'/tmp/previously-on' summary --simple") {
		t.Fatalf("unexpected managed block:\n%s", block)
	}
}

func TestUninstallBlockRemovesOnlyManagedBlock(t *testing.T) {
	path := filepath.Join(t.TempDir(), "post-merge")
	content := "#!/bin/sh\necho existing\n\n" + managedBlock("/tmp/previously-on")
	if err := os.WriteFile(path, []byte(content), 0o755); err != nil {
		t.Fatal(err)
	}

	result, err := uninstallBlock(path, "post-merge")
	if err != nil {
		t.Fatal(err)
	}
	if result.Action != "removed block" {
		t.Fatalf("expected removed block, got %#v", result)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), managedStart) || !strings.Contains(string(data), "echo existing") {
		t.Fatalf("unexpected hook content:\n%s", string(data))
	}
}
