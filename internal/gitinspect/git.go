package gitinspect

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// emptyTreeHash is Git's well-known empty tree object, used to diff a repo
// from no files when generating an initial summary.
const emptyTreeHash = "4b825dc642cb6eb9a060e54bf8d69288fbee4904"

type Repo struct {
	ID             string
	Name           string
	DisplayName    string
	Root           string
	RemoteURL      string
	CurrentBranch  string
	Head           string
	LocalBranches  map[string]string
	RemoteBranches map[string]string
}

type Commit struct {
	Hash     string
	Author   string
	AuthorAt time.Time
	Subject  string
	Refs     string
}

type FileChange struct {
	Path     string
	OldPath  string
	Status   string
	Category string
}

type DiffStat struct {
	FilesChanged int
	Insertions   int
	Deletions    int
}

type Activity struct {
	CommitsOnCurrentBranch []Commit
	CommitsAcrossBranches  []Commit
	ChangedFiles           []FileChange
	DiffStat               DiffStat
	NewLocalBranches       []string
	DeletedLocalBranches   []string
	NewRemoteBranches      []string
	DeletedRemoteBranches  []string
	RiskSignals            []string
	UsedTimestampFallback  bool
	InitialCommitLimit     int
}

func Discover() (Repo, error) {
	root, err := runGit("", "rev-parse", "--show-toplevel")
	if err != nil {
		return Repo{}, errors.New("not inside a Git repository")
	}
	root = strings.TrimSpace(root)

	branch, _ := runGit(root, "branch", "--show-current")
	head, err := runGit(root, "rev-parse", "HEAD")
	if err != nil {
		return Repo{}, err
	}
	remoteURL, _ := runGit(root, "remote", "get-url", "origin")

	repo := Repo{
		Name:           filepath.Base(root),
		Root:           root,
		RemoteURL:      strings.TrimSpace(remoteURL),
		CurrentBranch:  strings.TrimSpace(branch),
		Head:           strings.TrimSpace(head),
		LocalBranches:  map[string]string{},
		RemoteBranches: map[string]string{},
	}
	repo.DisplayName = displayName(repo.Name, repo.RemoteURL)
	repo.ID = repoID(repo.Root, repo.RemoteURL)
	repo.LocalBranches = readRefs(root, "refs/heads")
	repo.RemoteBranches = readRefs(root, "refs/remotes")
	return repo, nil
}

func displayName(name, remoteURL string) string {
	remoteURL = strings.TrimSuffix(strings.TrimSpace(remoteURL), ".git")
	if remoteURL == "" {
		return name
	}
	remoteURL = strings.TrimPrefix(remoteURL, "ssh://")
	remoteURL = strings.TrimPrefix(remoteURL, "https://")
	remoteURL = strings.TrimPrefix(remoteURL, "http://")
	remoteURL = strings.TrimPrefix(remoteURL, "git@")
	remoteURL = strings.Replace(remoteURL, ":", "/", 1)
	parts := strings.Split(remoteURL, "/")
	if len(parts) >= 2 {
		owner := parts[len(parts)-2]
		repo := parts[len(parts)-1]
		if owner != "" && repo != "" {
			return owner + "/" + repo
		}
	}
	return name
}

func Inspect(repo Repo, lastHead string, lastSeenAt time.Time, knownLocal, knownRemote map[string]string) (Activity, error) {
	activity := Activity{}
	rangeSpec := lastHead + "..HEAD"
	if lastHead == "" || !commitExists(repo.Root, lastHead) {
		activity.UsedTimestampFallback = true
		activity.CommitsOnCurrentBranch = gitLogSince(repo.Root, lastSeenAt, false)
		activity.CommitsAcrossBranches = gitLogSince(repo.Root, lastSeenAt, true)
	} else {
		activity.CommitsOnCurrentBranch = gitLogRange(repo.Root, rangeSpec, false)
		activity.CommitsAcrossBranches = gitLogSince(repo.Root, lastSeenAt, true)
		activity.ChangedFiles = changedFiles(repo.Root, rangeSpec)
		activity.DiffStat = diffStat(repo.Root, rangeSpec)
	}

	activity.NewLocalBranches, activity.DeletedLocalBranches = branchDelta(knownLocal, repo.LocalBranches)
	activity.NewRemoteBranches, activity.DeletedRemoteBranches = branchDelta(knownRemote, repo.RemoteBranches)
	for i := range activity.ChangedFiles {
		activity.ChangedFiles[i].Category = categorizePath(activity.ChangedFiles[i].Path)
	}
	activity.RiskSignals = riskSignals(activity)
	return activity, nil
}

func InspectInitial(repo Repo, commitLimit int) (Activity, error) {
	if commitLimit <= 0 {
		commitLimit = 30
	}
	activity := Activity{InitialCommitLimit: commitLimit}
	activity.CommitsOnCurrentBranch = gitLogLimit(repo.Root, commitLimit, false)
	activity.CommitsAcrossBranches = gitLogLimit(repo.Root, commitLimit, true)

	base := initialBase(repo.Root, commitLimit)
	rangeSpec := base + "..HEAD"
	activity.ChangedFiles = changedFiles(repo.Root, rangeSpec)
	activity.DiffStat = diffStat(repo.Root, rangeSpec)

	for i := range activity.ChangedFiles {
		activity.ChangedFiles[i].Category = categorizePath(activity.ChangedFiles[i].Path)
	}
	activity.RiskSignals = riskSignals(activity)
	return activity, nil
}

func repoID(root, remote string) string {
	sum := sha256.Sum256([]byte(root + "|" + remote))
	return hex.EncodeToString(sum[:])
}

func readRefs(root, ref string) map[string]string {
	out, err := runGit(root, "for-each-ref", ref, "--format=%(refname:short) %(objectname)")
	if err != nil {
		return map[string]string{}
	}
	refs := map[string]string{}
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		fields := strings.Fields(line)
		if len(fields) == 2 {
			refs[fields[0]] = fields[1]
		}
	}
	return refs
}

func gitLogRange(root, rangeSpec string, all bool) []Commit {
	args := []string{"log", rangeSpec, "--date=iso-strict", "--pretty=format:%H%x1f%an%x1f%aI%x1f%D%x1f%s%x1e"}
	if all {
		args = append([]string{"log", "--all"}, args[2:]...)
	}
	return parseCommits(mustGit(root, args...))
}

func gitLogSince(root string, since time.Time, all bool) []Commit {
	args := []string{"log", "--since", since.Format(time.RFC3339), "--date=iso-strict", "--pretty=format:%H%x1f%an%x1f%aI%x1f%D%x1f%s%x1e"}
	if all {
		args = append([]string{"log", "--all"}, args[1:]...)
	}
	return parseCommits(mustGit(root, args...))
}

func gitLogLimit(root string, limit int, all bool) []Commit {
	args := []string{"log", fmt.Sprintf("--max-count=%d", limit), "--date=iso-strict", "--pretty=format:%H%x1f%an%x1f%aI%x1f%D%x1f%s%x1e"}
	if all {
		args = append([]string{"log", "--all"}, args[1:]...)
	}
	return parseCommits(mustGit(root, args...))
}

func parseCommits(out string) []Commit {
	var commits []Commit
	for _, record := range strings.Split(out, "\x1e") {
		parts := strings.Split(strings.TrimSpace(record), "\x1f")
		if len(parts) != 5 {
			continue
		}
		authorAt, _ := time.Parse(time.RFC3339, parts[2])
		commits = append(commits, Commit{Hash: parts[0], Author: parts[1], AuthorAt: authorAt, Refs: parts[3], Subject: parts[4]})
	}
	return commits
}

func changedFiles(root, rangeSpec string) []FileChange {
	out := mustGit(root, "diff", "--name-status", "--find-renames", rangeSpec)
	var changes []FileChange
	for _, line := range strings.Split(strings.TrimSpace(out), "\n") {
		if strings.TrimSpace(line) == "" {
			continue
		}
		parts := strings.Split(line, "\t")
		change := FileChange{Status: parts[0]}
		if strings.HasPrefix(change.Status, "R") && len(parts) >= 3 {
			change.OldPath = parts[1]
			change.Path = parts[2]
		} else if len(parts) >= 2 {
			change.Path = parts[1]
		}
		changes = append(changes, change)
	}
	return changes
}

func diffStat(root, rangeSpec string) DiffStat {
	out := mustGit(root, "diff", "--shortstat", rangeSpec)
	return parseShortStat(out)
}

func parseShortStat(out string) DiffStat {
	stat := DiffStat{}
	for _, part := range strings.Split(out, ",") {
		fields := strings.Fields(strings.TrimSpace(part))
		if len(fields) < 2 {
			continue
		}
		var value int
		fmt.Sscanf(fields[0], "%d", &value)
		text := strings.Join(fields[1:], " ")
		switch {
		case strings.Contains(text, "file"):
			stat.FilesChanged = value
		case strings.Contains(text, "insertion"):
			stat.Insertions = value
		case strings.Contains(text, "deletion"):
			stat.Deletions = value
		}
	}
	return stat
}

func categorizePath(path string) string {
	lower := strings.ToLower(path)
	base := filepath.Base(lower)
	deps := map[string]bool{"package.json": true, "package-lock.json": true, "pnpm-lock.yaml": true, "yarn.lock": true, "gemfile": true, "gemfile.lock": true, "go.mod": true, "go.sum": true, "requirements.txt": true, "pyproject.toml": true, "poetry.lock": true, "cargo.toml": true, "cargo.lock": true}
	if deps[base] {
		return "dependency"
	}
	if strings.Contains(lower, "db/migrate/") || strings.Contains(lower, "migrations/") {
		return "migration"
	}
	if strings.HasPrefix(lower, ".github/workflows/") || strings.Contains(lower, "docker") || strings.Contains(lower, "deploy") || strings.Contains(lower, "terraform") || strings.Contains(lower, "k8s/") || strings.Contains(lower, "config/") || strings.HasPrefix(base, ".env") {
		return "config"
	}
	if strings.Contains(lower, "test") || strings.Contains(lower, "spec") {
		return "test"
	}
	return "source"
}

func riskSignals(activity Activity) []string {
	seen := map[string]bool{}
	add := func(signal string) {
		if !seen[signal] {
			seen[signal] = true
		}
	}
	for _, file := range activity.ChangedFiles {
		switch file.Category {
		case "dependency":
			add("Dependency files changed")
		case "migration":
			add("Database migrations changed")
		case "config":
			add("Config, CI, or deployment files changed")
		}
		if strings.HasPrefix(file.Status, "D") {
			add("Files were deleted")
		}
	}
	for _, commit := range activity.CommitsOnCurrentBranch {
		lower := strings.ToLower(commit.Subject)
		for _, word := range []string{"breaking", "remove", "drop", "deprecate", "security", "hotfix", "revert", "rollback"} {
			if strings.Contains(lower, word) {
				add("Risk-looking commit messages found")
			}
		}
	}
	if activity.DiffStat.FilesChanged >= 25 || activity.DiffStat.Insertions+activity.DiffStat.Deletions >= 1500 {
		add("Large diff since last seen")
	}
	var signals []string
	for signal := range seen {
		signals = append(signals, signal)
	}
	return signals
}

func branchDelta(previous, current map[string]string) ([]string, []string) {
	var added, deleted []string
	for name := range current {
		if _, ok := previous[name]; !ok {
			added = append(added, name)
		}
	}
	for name := range previous {
		if _, ok := current[name]; !ok {
			deleted = append(deleted, name)
		}
	}
	return added, deleted
}

func initialBase(root string, commitLimit int) string {
	out := mustGit(root, "rev-list", fmt.Sprintf("--max-count=%d", commitLimit+1), "HEAD")
	commits := strings.Fields(out)
	if len(commits) <= commitLimit {
		return emptyTreeHash
	}
	return commits[len(commits)-1]
}

func commitExists(root, commit string) bool {
	_, err := runGit(root, "cat-file", "-e", commit+"^{commit}")
	return err == nil
}

func mustGit(root string, args ...string) string {
	out, _ := runGit(root, args...)
	return out
}

func runGit(root string, args ...string) (string, error) {
	cmd := exec.Command("git", args...)
	if root != "" {
		cmd.Dir = root
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.String()))
	}
	return stdout.String(), nil
}
