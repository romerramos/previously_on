package ai

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/gitinspect"
)

type Provider interface {
	Generate(context.Context, Payload) (GeneratedSections, error)
}

type GeneratedSections struct {
	TLDR                string   `json:"tldr"`
	MainStoryline       []string `json:"mainStoryline"`
	ThingsWorthChecking []string `json:"thingsWorthChecking"`
}

type Payload struct {
	RepoName      string
	CurrentBranch string
	Commits       []gitinspect.Commit
	ChangedFiles  []gitinspect.FileChange
	RiskSignals   []string
}

func BuildPayload(cfg config.Config, repo gitinspect.Repo, activity gitinspect.Activity) Payload {
	payload := Payload{RepoName: repo.Name, CurrentBranch: repo.CurrentBranch, RiskSignals: activity.RiskSignals}
	if cfg.Privacy.SendCommitMessages {
		payload.Commits = limitCommits(activity.CommitsOnCurrentBranch, cfg.Privacy.MaxCommits)
	}
	if cfg.Privacy.SendFilePaths {
		payload.ChangedFiles = limitFiles(activity.ChangedFiles, cfg.Privacy.MaxFilePaths)
	}
	return payload
}

func SystemPrompt() string {
	return `You write concise developer briefings from Git metadata.
Be factual, specific, useful, and not hypey. Do not invent details.
Return only valid JSON with this shape:
{"tldr":"1-3 conversational sentences about who did what and what areas changed","mainStoryline":["important change"],"thingsWorthChecking":["specific thing to verify"]}
Rules:
- Keep arrays short, usually 3-6 items.
- Write thingsWorthChecking with a code review mindset: read the changes critically and point to files, areas, risks, surprising choices, missing follow-up, or elegant solutions worth noticing.
- Avoid generic QA advice like "test the app" unless the specific commits strongly justify it.
- Prefer concrete file paths, modules, commit subjects, or change areas over vague recommendations.
- If you are unsure, omit the item.`
}

func UserPrompt(payload Payload) string {
	var b strings.Builder
	fmt.Fprintf(&b, "Generate analytical report sections for repo %q on branch %q.\n\n", payload.RepoName, payload.CurrentBranch)
	b.WriteString("Commits:\n")
	for _, commit := range payload.Commits {
		fmt.Fprintf(&b, "- %s: %s\n", commit.Author, commit.Subject)
	}
	b.WriteString("\nChanged files:\n")
	for _, file := range payload.ChangedFiles {
		fmt.Fprintf(&b, "- %s (%s)\n", file.Path, file.Category)
	}
	if len(payload.RiskSignals) > 0 {
		b.WriteString("\nRisk signals:\n")
		for _, signal := range payload.RiskSignals {
			fmt.Fprintf(&b, "- %s\n", signal)
		}
	}
	return b.String()
}

func ParseGeneratedSections(text string) (GeneratedSections, error) {
	text = strings.TrimSpace(text)
	text = strings.TrimPrefix(text, "```json")
	text = strings.TrimPrefix(text, "```")
	text = strings.TrimSuffix(text, "```")
	text = strings.TrimSpace(text)
	var sections GeneratedSections
	if err := json.Unmarshal([]byte(text), &sections); err != nil {
		return GeneratedSections{}, err
	}
	return sections, nil
}

func limitCommits(commits []gitinspect.Commit, max int) []gitinspect.Commit {
	if max <= 0 || len(commits) <= max {
		return commits
	}
	return commits[:max]
}

func limitFiles(files []gitinspect.FileChange, max int) []gitinspect.FileChange {
	if max <= 0 || len(files) <= max {
		return files
	}
	return files[:max]
}
