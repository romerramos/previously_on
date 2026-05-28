package report

import (
	"strings"
	"testing"
	"time"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/gitinspect"
	"github.com/romerramos/previously_on/internal/state"
)

func TestRenderIncludesConversationalTLDR(t *testing.T) {
	repo := gitinspect.Repo{Name: "billing-api", CurrentBranch: "main", Head: "abcdef123456789"}
	previous := state.RepoState{LastSeenBranch: "main", LastSeenHead: "11111112222222", LastSeenAt: time.Now().Add(-24 * time.Hour)}
	activity := gitinspect.Activity{
		CommitsOnCurrentBranch: []gitinspect.Commit{{Author: "Jack", Subject: "Refactor billing process"}},
		ChangedFiles:           []gitinspect.FileChange{{Path: "services/billing.go", Status: "M", Category: "source"}},
		DiffStat:               gitinspect.DiffStat{FilesChanged: 1},
	}

	markdown := Render(Build(repo, previous, activity, ai.GeneratedSections{TLDR: "Jack refactored billing, affecting services and components."}))

	if !strings.Contains(markdown, "## TL;DR") {
		t.Fatalf("expected TL;DR section in markdown:\n%s", markdown)
	}
	if !strings.Contains(markdown, "Jack refactored billing") {
		t.Fatalf("expected custom TL;DR in markdown:\n%s", markdown)
	}
}

func TestFallbackReportOmitsAnalyticalSections(t *testing.T) {
	report := Build(gitinspect.Repo{Name: "app", CurrentBranch: "main"}, state.RepoState{}, gitinspect.Activity{
		CommitsOnCurrentBranch: []gitinspect.Commit{{Author: "Jack", Subject: "Update dependencies"}},
		ChangedFiles: []gitinspect.FileChange{
			{Path: "Gemfile.lock", Category: "dependency"},
			{Path: "db/migrate/20260528000000_add_users.rb", Category: "migration"},
		},
	}, ai.GeneratedSections{})

	markdown := Render(report)
	for _, unwanted := range []string{"## TL;DR", "## Main storyline", "## Things worth checking", "## Suggested next commands"} {
		if strings.Contains(markdown, unwanted) {
			t.Fatalf("did not expect %q in fallback markdown:\n%s", unwanted, markdown)
		}
	}
	if !strings.Contains(markdown, "## Commits included") {
		t.Fatalf("expected included commits in fallback markdown:\n%s", markdown)
	}
}

func TestBuildInitialUsesFirstLookWording(t *testing.T) {
	repo := gitinspect.Repo{Name: "twitreads", CurrentBranch: "main", Head: "abcdef123456789", LocalBranches: map[string]string{"main": "abcdef123456789"}}
	activity := gitinspect.Activity{
		InitialCommitLimit:     30,
		CommitsOnCurrentBranch: []gitinspect.Commit{{Author: "Kate", Subject: "Update docs and pipeline"}},
		ChangedFiles:           []gitinspect.FileChange{{Path: ".gitlab-ci.yml", Status: "M", Category: "config"}},
		DiffStat:               gitinspect.DiffStat{FilesChanged: 1},
	}

	markdown := RenderWithOptions(BuildInitial(repo, activity, ai.GeneratedSections{TLDR: "Kate changed the docs and GitLab pipeline setup."}), RenderOptions{Detail: DetailMedium})

	for _, want := range []string{"## TL;DR", "## First look", "Kate changed the docs", "Future reports will only include changes after this point"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected %q in markdown:\n%s", want, markdown)
		}
	}
	if strings.Contains(markdown, "## Since you were last here") {
		t.Fatalf("initial report should not use since-last-seen wording:\n%s", markdown)
	}
}

func TestSimpleAIReportFocusesOnConclusions(t *testing.T) {
	report := Build(gitinspect.Repo{Name: "app", CurrentBranch: "main", Head: "abc123456"}, state.RepoState{LastSeenHead: "def123456"}, gitinspect.Activity{
		CommitsOnCurrentBranch: []gitinspect.Commit{{Hash: "abc123456", Author: "Jack", Subject: "Update dependencies"}},
		ChangedFiles:           []gitinspect.FileChange{{Path: "package.json", Status: "M", Category: "dependency"}},
	}, ai.GeneratedSections{
		TLDR:                "Jack updated dependencies.",
		MainStoryline:       []string{"Dependencies changed."},
		ThingsWorthChecking: []string{"Review dependency updates."},
	})

	markdown := RenderWithOptions(report, RenderOptions{Detail: DetailSimple})
	for _, want := range []string{"## TL;DR", "## Things worth checking", "## Commits included", "`abc1234` Jack: Update dependencies"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected %q in simple markdown:\n%s", want, markdown)
		}
	}
	for _, unwanted := range []string{"## Main storyline", "## Files that moved the plot", "## Branch activity"} {
		if strings.Contains(markdown, unwanted) {
			t.Fatalf("did not expect %q in simple markdown:\n%s", unwanted, markdown)
		}
	}
}

func TestDetailedReportIncludesDiffStatsAndRecentCommits(t *testing.T) {
	report := Build(gitinspect.Repo{Name: "app", CurrentBranch: "main", Head: "abc123456"}, state.RepoState{LastSeenHead: "def123456"}, gitinspect.Activity{
		CommitsOnCurrentBranch: []gitinspect.Commit{{Author: "Jack", Subject: "Update dependencies"}},
		DiffStat:               gitinspect.DiffStat{FilesChanged: 2, Insertions: 10, Deletions: 3},
	}, ai.GeneratedSections{TLDR: "Jack updated dependencies."})

	markdown := RenderWithOptions(report, RenderOptions{Detail: DetailDetailed})
	for _, want := range []string{"## Diff stats", "Files changed: 2", "## Recent commits"} {
		if !strings.Contains(markdown, want) {
			t.Fatalf("expected %q in detailed markdown:\n%s", want, markdown)
		}
	}
}
