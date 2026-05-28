package report

import (
	"fmt"
	"sort"
	"strings"
	"time"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/gitinspect"
	"github.com/romerramos/previously_on/internal/state"
)

type Report struct {
	RepoName        string
	GeneratedAt     time.Time
	Initial         bool
	AIEnhanced      bool
	TLDR            string
	Overview        string
	MainStoryline   []string
	BranchActivity  []string
	ImportantFiles  []string
	ThingsToCheck   []string
	RecentCommits   []string
	IncludedCommits []string
	DiffStat        gitinspect.DiffStat
}

type DetailLevel string

const (
	DetailSimple   DetailLevel = "simple"
	DetailMedium   DetailLevel = "medium"
	DetailDetailed DetailLevel = "detailed"
)

type RenderOptions struct {
	Detail DetailLevel
}

func Baseline(repo gitinspect.Repo) string {
	return fmt.Sprintf("# Previously on `%s`\n\nThis is the first time Previously on... has seen this repo. Baseline stored at `%s` on `%s`.\n", repo.Name, shortHash(repo.Head), displayBranch(repo.CurrentBranch))
}

func Build(repo gitinspect.Repo, previous state.RepoState, activity gitinspect.Activity, generated ai.GeneratedSections) Report {
	return build(repo, previous, activity, generated, false)
}

func BuildWithIncludedCommits(repo gitinspect.Repo, previous state.RepoState, activity gitinspect.Activity, generated ai.GeneratedSections, included []gitinspect.Commit) Report {
	report := build(repo, previous, activity, generated, false)
	report.IncludedCommits = commitLines(included)
	return report
}

func BuildInitial(repo gitinspect.Repo, activity gitinspect.Activity, generated ai.GeneratedSections) Report {
	previous := state.RepoState{LastSeenBranch: repo.CurrentBranch, LastSeenHead: repo.Head, LastSeenAt: time.Now().UTC()}
	return build(repo, previous, activity, generated, true)
}

func BuildInitialWithIncludedCommits(repo gitinspect.Repo, activity gitinspect.Activity, generated ai.GeneratedSections, included []gitinspect.Commit) Report {
	report := BuildInitial(repo, activity, generated)
	report.IncludedCommits = commitLines(included)
	return report
}

func build(repo gitinspect.Repo, previous state.RepoState, activity gitinspect.Activity, generated ai.GeneratedSections, initial bool) Report {
	report := Report{RepoName: repo.Name, GeneratedAt: time.Now().UTC(), Initial: initial}
	report.AIEnhanced = generated.TLDR != "" || len(generated.MainStoryline) > 0 || len(generated.ThingsWorthChecking) > 0
	report.TLDR = generated.TLDR
	if initial {
		report.Overview = initialOverview(repo, activity)
	} else {
		report.Overview = overview(repo, previous, activity)
	}
	report.MainStoryline = generated.MainStoryline
	if initial {
		report.BranchActivity = initialBranchActivity(repo)
	} else {
		report.BranchActivity = branchActivity(repo, previous, activity)
	}
	report.ImportantFiles = importantFiles(activity)
	report.ThingsToCheck = generated.ThingsWorthChecking
	report.RecentCommits = recentCommits(activity)
	report.IncludedCommits = includedCommits(activity)
	report.DiffStat = activity.DiffStat
	return report
}

func Render(r Report) string {
	return RenderWithOptions(r, RenderOptions{Detail: DetailSimple})
}

func RenderWithOptions(r Report, opts RenderOptions) string {
	detail := opts.Detail
	if detail == "" {
		detail = DetailSimple
	}

	var b strings.Builder
	fmt.Fprintf(&b, "# Previously on `%s`\n\n", r.RepoName)
	if r.AIEnhanced && r.TLDR != "" {
		fmt.Fprintf(&b, "## TL;DR\n\n%s\n\n", r.TLDR)
	}
	if detail != DetailSimple || !r.AIEnhanced {
		if r.Initial {
			fmt.Fprintf(&b, "## First look\n\n%s\n\n", r.Overview)
		} else {
			fmt.Fprintf(&b, "## Since you were last here\n\n%s\n\n", r.Overview)
		}
	}
	if r.AIEnhanced && detail != DetailSimple {
		renderList(&b, "Main storyline", r.MainStoryline)
	}
	if detail != DetailSimple && hasNotableBranchActivity(r.BranchActivity) {
		renderList(&b, "Branch activity", r.BranchActivity)
	}
	if detail != DetailSimple {
		renderListLimit(&b, "Files that moved the plot", r.ImportantFiles, fileLimit(detail))
	}
	if r.AIEnhanced {
		renderList(&b, "Things worth checking", r.ThingsToCheck)
	} else if detail != DetailSimple {
		renderListLimit(&b, "Recent commits", r.RecentCommits, commitLimit(detail))
	}
	if detail == DetailSimple {
		renderListLimit(&b, "Commits included", r.IncludedCommits, 5)
	}
	if detail == DetailDetailed {
		renderDiffStat(&b, r.DiffStat)
		if r.AIEnhanced {
			renderListLimit(&b, "Recent commits", r.RecentCommits, commitLimit(detail))
		}
	}
	return b.String()
}

func renderList(b *strings.Builder, title string, items []string) {
	fmt.Fprintf(b, "## %s\n\n", title)
	if len(items) == 0 {
		b.WriteString("- Nothing notable detected.\n\n")
		return
	}
	for _, item := range items {
		fmt.Fprintf(b, "- %s\n", item)
	}
	b.WriteString("\n")
}

func renderListLimit(b *strings.Builder, title string, items []string, limit int) {
	if limit > 0 && len(items) > limit {
		items = items[:limit]
	}
	renderList(b, title, items)
}

func renderDiffStat(b *strings.Builder, stat gitinspect.DiffStat) {
	fmt.Fprintf(b, "## Diff stats\n\n")
	fmt.Fprintf(b, "- Files changed: %d\n", stat.FilesChanged)
	fmt.Fprintf(b, "- Insertions: %d\n", stat.Insertions)
	fmt.Fprintf(b, "- Deletions: %d\n\n", stat.Deletions)
}

func hasNotableBranchActivity(items []string) bool {
	return len(items) > 1
}

func fileLimit(detail DetailLevel) int {
	if detail == DetailDetailed {
		return 25
	}
	return 12
}

func commitLimit(detail DetailLevel) int {
	if detail == DetailDetailed {
		return 20
	}
	if detail == DetailSimple {
		return 0
	}
	return 8
}

func overview(repo gitinspect.Repo, previous state.RepoState, activity gitinspect.Activity) string {
	commitCount := len(activity.CommitsOnCurrentBranch)
	if activity.UsedTimestampFallback {
		return fmt.Sprintf("You last saw `%s` on `%s`. The previous commit is no longer available locally, so this summary uses commits since `%s` as a fallback.", repo.Name, displayBranch(previous.LastSeenBranch), previous.LastSeenAt.Format("2006-01-02"))
	}
	return fmt.Sprintf("You last saw `%s` on `%s` at `%s`. Since then, `%s` has %d new commit%s with %d files changed.", repo.Name, displayBranch(previous.LastSeenBranch), shortHash(previous.LastSeenHead), displayBranch(repo.CurrentBranch), commitCount, plural(commitCount), activity.DiffStat.FilesChanged)
}

func initialOverview(repo gitinspect.Repo, activity gitinspect.Activity) string {
	commitCount := len(activity.CommitsOnCurrentBranch)
	limit := activity.InitialCommitLimit
	if limit <= 0 {
		limit = commitCount
	}
	return fmt.Sprintf("This is the first time Previously on... has seen `%s`. I reviewed the latest %d commit%s on `%s` and stored `%s` as the baseline. Future reports will only include changes after this point.", repo.Name, commitCount, plural(commitCount), displayBranch(repo.CurrentBranch), shortHash(repo.Head))
}

func recentCommits(activity gitinspect.Activity) []string {
	var items []string
	for i, commit := range activity.CommitsOnCurrentBranch {
		if i >= 20 {
			break
		}
		items = append(items, fmt.Sprintf("%s: %s", commit.Author, commit.Subject))
	}
	return items
}

func includedCommits(activity gitinspect.Activity) []string {
	return commitLines(activity.CommitsOnCurrentBranch)
}

func commitLines(commits []gitinspect.Commit) []string {
	var items []string
	for i, commit := range commits {
		if i >= 20 {
			break
		}
		items = append(items, fmt.Sprintf("`%s` %s: %s", shortHash(commit.Hash), commit.Author, commit.Subject))
	}
	return items
}

func branchActivity(repo gitinspect.Repo, previous state.RepoState, activity gitinspect.Activity) []string {
	items := []string{fmt.Sprintf("`%s` moved from `%s` to `%s`.", displayBranch(repo.CurrentBranch), shortHash(previous.LastSeenHead), shortHash(repo.Head))}
	for _, branch := range activity.NewLocalBranches {
		items = append(items, fmt.Sprintf("New local branch detected: `%s`.", branch))
	}
	for _, branch := range activity.NewRemoteBranches {
		items = append(items, fmt.Sprintf("New remote branch detected: `%s`.", branch))
	}
	for _, branch := range activity.DeletedLocalBranches {
		items = append(items, fmt.Sprintf("Previously known local branch is missing: `%s`.", branch))
	}
	for _, branch := range activity.DeletedRemoteBranches {
		items = append(items, fmt.Sprintf("Previously known remote branch is missing: `%s`.", branch))
	}
	return items
}

func initialBranchActivity(repo gitinspect.Repo) []string {
	items := []string{fmt.Sprintf("Current branch is `%s` at `%s`.", displayBranch(repo.CurrentBranch), shortHash(repo.Head))}
	if len(repo.LocalBranches) > 0 {
		items = append(items, fmt.Sprintf("Detected %d local branch%s.", len(repo.LocalBranches), plural(len(repo.LocalBranches))))
	}
	if len(repo.RemoteBranches) > 0 {
		items = append(items, fmt.Sprintf("Detected %d remote branch%s.", len(repo.RemoteBranches), plural(len(repo.RemoteBranches))))
	}
	return items
}

func importantFiles(activity gitinspect.Activity) []string {
	changes := activity.ChangedFiles
	sort.SliceStable(changes, func(i, j int) bool {
		return categoryRank(changes[i].Category) < categoryRank(changes[j].Category)
	})
	var items []string
	for i, change := range changes {
		if i >= 25 {
			break
		}
		label := change.Path
		if change.OldPath != "" {
			label = change.OldPath + " -> " + change.Path
		}
		items = append(items, fmt.Sprintf("`%s` (%s, %s)", label, change.Status, change.Category))
	}
	return items
}

func categoryRank(category string) int {
	switch category {
	case "migration":
		return 0
	case "dependency":
		return 1
	case "config":
		return 2
	case "source":
		return 3
	case "test":
		return 4
	default:
		return 5
	}
}

func shortHash(hash string) string {
	if len(hash) > 7 {
		return hash[:7]
	}
	return hash
}

func displayBranch(branch string) string {
	if branch == "" {
		return "detached HEAD"
	}
	return branch
}

func plural(count int) string {
	if count == 1 {
		return ""
	}
	return "s"
}
