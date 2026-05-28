package state

import (
	"testing"

	"github.com/romerramos/previously_on/internal/gitinspect"
)

func TestStoreSaveLoadAndReset(t *testing.T) {
	store, err := NewStore(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	repo := gitinspect.Repo{ID: "repo-id", Name: "repo", Root: "/tmp/repo", Head: "abc", CurrentBranch: "main"}
	snapshot, err := FromRepo(repo)
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Save(snapshot); err != nil {
		t.Fatal(err)
	}

	loaded, err := store.Load("repo-id")
	if err != nil {
		t.Fatal(err)
	}
	if loaded == nil || loaded.LastSeenHead != "abc" {
		t.Fatalf("unexpected loaded state: %#v", loaded)
	}

	if err := store.Reset("repo-id"); err != nil {
		t.Fatal(err)
	}
	loaded, err = store.Load("repo-id")
	if err != nil {
		t.Fatal(err)
	}
	if loaded != nil {
		t.Fatalf("expected reset state to be nil, got %#v", loaded)
	}
}
