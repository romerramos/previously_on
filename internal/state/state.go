package state

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"time"

	"github.com/romerramos/previously_on/internal/gitinspect"
)

type RepoState struct {
	SchemaVersion          int               `json:"schemaVersion"`
	RepoID                 string            `json:"repoId"`
	RepoName               string            `json:"repoName"`
	RepoRoot               string            `json:"repoRoot"`
	RemoteURL              string            `json:"remoteUrl"`
	CreatedAt              time.Time         `json:"createdAt"`
	LastSeenAt             time.Time         `json:"lastSeenAt"`
	LastSeenBranch         string            `json:"lastSeenBranch"`
	LastSeenHead           string            `json:"lastSeenHead"`
	KnownLocalBranches     map[string]string `json:"knownLocalBranches"`
	KnownRemoteBranches    map[string]string `json:"knownRemoteBranches"`
	LastGeneratedSummaryAt time.Time         `json:"lastGeneratedSummaryAt,omitempty"`
	LastSummaryPath        string            `json:"lastSummaryPath,omitempty"`
}

type Store struct {
	root string
}

func NewStore(root string) (*Store, error) {
	if err := os.MkdirAll(filepath.Join(root, "repos"), 0o755); err != nil {
		return nil, err
	}
	return &Store{root: root}, nil
}

func FromRepo(repo gitinspect.Repo) (RepoState, error) {
	now := time.Now().UTC()
	return RepoState{
		SchemaVersion:       1,
		RepoID:              repo.ID,
		RepoName:            repo.Name,
		RepoRoot:            repo.Root,
		RemoteURL:           repo.RemoteURL,
		CreatedAt:           now,
		LastSeenAt:          now,
		LastSeenBranch:      repo.CurrentBranch,
		LastSeenHead:        repo.Head,
		KnownLocalBranches:  repo.LocalBranches,
		KnownRemoteBranches: repo.RemoteBranches,
	}, nil
}

func (s *Store) Load(repoID string) (*RepoState, error) {
	data, err := os.ReadFile(s.statePath(repoID))
	if errors.Is(err, os.ErrNotExist) {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	var state RepoState
	if err := json.Unmarshal(data, &state); err != nil {
		return nil, err
	}
	return &state, nil
}

func (s *Store) Save(repoState RepoState) error {
	if repoState.CreatedAt.IsZero() {
		repoState.CreatedAt = time.Now().UTC()
	}
	data, err := json.MarshalIndent(repoState, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(s.statePath(repoState.RepoID), append(data, '\n'), 0o644)
}

func (s *Store) SaveSummary(repoID string, generatedAt time.Time, markdown string) (string, error) {
	dir := filepath.Join(s.root, "repos", repoID+".summaries")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return "", err
	}
	path := filepath.Join(dir, generatedAt.Format("2006-01-02T15-04-05Z")+".md")
	return path, os.WriteFile(path, []byte(markdown), 0o644)
}

func (s *Store) Reset(repoID string) error {
	if err := os.Remove(s.statePath(repoID)); errors.Is(err, os.ErrNotExist) {
		return nil
	} else if err != nil {
		return err
	}
	return os.RemoveAll(filepath.Join(s.root, "repos", repoID+".summaries"))
}

func (s *Store) ResetAll() error {
	return os.RemoveAll(s.root)
}

func (s *Store) statePath(repoID string) string {
	return filepath.Join(s.root, "repos", repoID+".json")
}
