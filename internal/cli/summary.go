package cli

import (
	"fmt"
	"io"
	"os"
	"strings"
	"sync/atomic"
	"time"

	"github.com/romerramos/previously_on/internal/ai"
	"github.com/romerramos/previously_on/internal/config"
	"github.com/romerramos/previously_on/internal/gitinspect"
	"github.com/romerramos/previously_on/internal/report"
	"github.com/romerramos/previously_on/internal/secrets"
	"github.com/romerramos/previously_on/internal/state"
	"github.com/romerramos/previously_on/internal/termrender"
	"github.com/spf13/cobra"
)

type summaryOutputOptions struct {
	Raw         bool
	RenderStyle string
	RenderWidth int
}

type summaryDetailFlags struct {
	Simple   bool
	Medium   bool
	Detailed bool
}

func summaryCmd() *cobra.Command {
	var noAI bool
	var offline bool
	var output string
	var noMarkSeen bool
	var showAIPayload bool
	var initialCommits int
	var baselineOnly bool
	var outputOptions summaryOutputOptions
	var detailFlags summaryDetailFlags

	cmd := &cobra.Command{
		Use:   "summary",
		Short: "Generate a Markdown briefing for the current repo",
		RunE: func(cmd *cobra.Command, args []string) error {
			detail, err := resolveSummaryDetail(detailFlags)
			if err != nil {
				return err
			}

			ctx, err := loadContext()
			if err != nil {
				return err
			}

			previous, err := ctx.Store.Load(ctx.Repo.ID)
			if err != nil {
				return err
			}

			if previous == nil {
				snapshot, err := state.FromRepo(ctx.Repo)
				if err != nil {
					return err
				}

				if baselineOnly {
					if !noMarkSeen {
						if err := ctx.Store.Save(snapshot); err != nil {
							return err
						}
					}
					markdown := report.Baseline(ctx.Repo)
					return writeSummary(cmd, markdown, output, outputOptions)
				}

				activity, err := gitinspect.InspectInitial(ctx.Repo, initialCommits)
				if err != nil {
					return err
				}
				payload := ai.BuildPayload(ctx.Config, ctx.Repo, activity)
				if showAIPayload {
					fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\nThis is an initial repository briefing, not a since-last-seen summary.\n%s\n", ai.SystemPrompt(), ai.UserPrompt(payload))
				}

				generated := generateSections(cmd, ctx.Config, ctx.Repo.DisplayName, payload, noAI, offline)

				reportModel := report.BuildInitialWithIncludedCommits(ctx.Repo, activity, generated, payload.Commits)
				markdown := report.RenderWithOptions(reportModel, report.RenderOptions{Detail: detail})
				if err := writeSummary(cmd, markdown, output, outputOptions); err != nil {
					return err
				}
				if noMarkSeen {
					return nil
				}
				snapshot.LastGeneratedSummaryAt = time.Now().UTC()
				if output == "" {
					path, err := ctx.Store.SaveSummary(ctx.Repo.ID, snapshot.LastGeneratedSummaryAt, markdown)
					if err != nil {
						return err
					}
					snapshot.LastSummaryPath = path
				} else {
					snapshot.LastSummaryPath = output
				}
				return ctx.Store.Save(snapshot)
			}

			activity, err := gitinspect.Inspect(ctx.Repo, previous.LastSeenHead, previous.LastSeenAt, previous.KnownLocalBranches, previous.KnownRemoteBranches)
			if err != nil {
				return err
			}

			payload := ai.BuildPayload(ctx.Config, ctx.Repo, activity)
			if showAIPayload {
				fmt.Fprintf(cmd.ErrOrStderr(), "%s\n\n%s\n", ai.SystemPrompt(), ai.UserPrompt(payload))
			}

			generated := generateSections(cmd, ctx.Config, ctx.Repo.DisplayName, payload, noAI, offline)

			reportModel := report.BuildWithIncludedCommits(ctx.Repo, *previous, activity, generated, payload.Commits)
			markdown := report.RenderWithOptions(reportModel, report.RenderOptions{Detail: detail})
			if err := writeSummary(cmd, markdown, output, outputOptions); err != nil {
				return err
			}

			if noMarkSeen {
				return nil
			}

			next, err := state.FromRepo(ctx.Repo)
			if err != nil {
				return err
			}
			next.CreatedAt = previous.CreatedAt
			next.LastGeneratedSummaryAt = time.Now().UTC()
			if output == "" {
				path, err := ctx.Store.SaveSummary(ctx.Repo.ID, next.LastGeneratedSummaryAt, markdown)
				if err != nil {
					return err
				}
				next.LastSummaryPath = path
			} else {
				next.LastSummaryPath = output
			}
			return ctx.Store.Save(next)
		},
	}

	cmd.Flags().BoolVar(&noAI, "no-ai", false, "generate the deterministic local summary without AI")
	cmd.Flags().BoolVar(&offline, "offline", false, "avoid network-dependent features")
	cmd.Flags().StringVarP(&output, "output", "o", "", "write Markdown report to a file")
	cmd.Flags().BoolVar(&noMarkSeen, "no-mark-seen", false, "do not update last-seen state after generating the report")
	cmd.Flags().BoolVar(&showAIPayload, "show-ai-payload", false, "print the prompt payload that would be sent to the AI provider")
	cmd.Flags().IntVar(&initialCommits, "initial-commits", 30, "number of recent commits to summarize on first run")
	cmd.Flags().BoolVar(&baselineOnly, "baseline-only", false, "on first run, only store the baseline and skip the initial history summary")
	cmd.Flags().BoolVar(&outputOptions.Raw, "raw", false, "print raw Markdown instead of rendered terminal output")
	cmd.Flags().StringVar(&outputOptions.RenderStyle, "render-style", "tokyo-night", "Glamour style for rendered terminal output")
	cmd.Flags().IntVar(&outputOptions.RenderWidth, "render-width", 100, "word wrap width for rendered terminal output")
	cmd.Flags().BoolVar(&detailFlags.Simple, "simple", false, "show a short report focused on AI conclusions")
	cmd.Flags().BoolVar(&detailFlags.Medium, "medium", false, "show a balanced report")
	cmd.Flags().BoolVar(&detailFlags.Detailed, "detailed", false, "show a detailed report with extra files, commits, and diff stats")
	return cmd
}

func resolveSummaryDetail(flags summaryDetailFlags) (report.DetailLevel, error) {
	selected := 0
	if flags.Simple {
		selected++
	}
	if flags.Medium {
		selected++
	}
	if flags.Detailed {
		selected++
	}
	if selected > 1 {
		return "", fmt.Errorf("choose only one of --simple, --medium, or --detailed")
	}
	if flags.Medium {
		return report.DetailMedium, nil
	}
	if flags.Detailed {
		return report.DetailDetailed, nil
	}
	return report.DetailSimple, nil
}

func generateSections(cmd *cobra.Command, cfg config.Config, repoLabel string, payload ai.Payload, noAI, offline bool) ai.GeneratedSections {
	if noAI || offline || !cfg.AI.Enabled {
		return ai.GeneratedSections{}
	}
	info, ok := ai.FindProvider(cfg.AI.Provider)
	if !ok {
		fmt.Fprintf(cmd.ErrOrStderr(), "Unknown AI provider %q; using local fallback summary.\n", cfg.AI.Provider)
		return ai.GeneratedSections{}
	}
	key := ""
	if info.RequiresKey {
		resolved, _, err := secrets.Resolve(secrets.KeyringStore{}, info.ID, info.EnvVar)
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "No %s API key found. Run `previously-on connect` or set %s. Using local fallback summary.\n", info.Name, info.EnvVar)
			return ai.GeneratedSections{}
		}
		key = resolved
	}
	model := cfg.AI.Model
	if cfg.AI.Models != nil && cfg.AI.Models[info.ID] != "" {
		model = cfg.AI.Models[info.ID]
	}
	stop := startAISpinner(cmd.ErrOrStderr(), repoLabel)
	sections, err := (ai.GenkitProvider{Info: info, Key: key, Model: model}).Generate(cmd.Context(), payload)
	stop()
	if err != nil {
		fmt.Fprintf(cmd.ErrOrStderr(), "AI summary failed: %v. Using local fallback summary.\n", err)
		return ai.GeneratedSections{}
	}
	return sections
}

func startAISpinner(w io.Writer, repoLabel string) func() {
	if !isTerminal(w) {
		return func() {}
	}
	if strings.TrimSpace(repoLabel) == "" {
		repoLabel = "this repo"
	}
	var done atomic.Bool
	go func() {
		frames := []string{"⠋", "⠙", "⠹", "⠸", "⠼", "⠴", "⠦", "⠧", "⠇", "⠏"}
		for i := 0; !done.Load(); i++ {
			fmt.Fprintf(w, "\r%s Previously on %s...", frames[i%len(frames)], repoLabel)
			time.Sleep(100 * time.Millisecond)
		}
	}()
	return func() {
		done.Store(true)
		time.Sleep(120 * time.Millisecond)
		fmt.Fprint(w, "\r\033[2K")
	}
}

func writeSummary(cmd *cobra.Command, markdown, output string, opts summaryOutputOptions) error {
	if output == "" {
		if opts.Raw || !isTerminal(cmd.OutOrStdout()) {
			fmt.Fprint(cmd.OutOrStdout(), markdown)
			return nil
		}
		rendered, err := termrender.Markdown(markdown, termrender.MarkdownOptions{Style: opts.RenderStyle, Width: opts.RenderWidth})
		if err != nil {
			fmt.Fprintf(cmd.ErrOrStderr(), "Markdown rendering failed: %v. Printing raw Markdown.\n", err)
			fmt.Fprint(cmd.OutOrStdout(), markdown)
			return nil
		}
		fmt.Fprint(cmd.OutOrStdout(), rendered)
		return nil
	}
	if err := os.WriteFile(output, []byte(markdown), 0o644); err != nil {
		return err
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Wrote %s\n", output)
	return nil
}

func isTerminal(file any) bool {
	f, ok := file.(*os.File)
	if !ok {
		return false
	}
	info, err := f.Stat()
	if err != nil {
		return false
	}
	return info.Mode()&os.ModeCharDevice != 0 && f.Name() != os.DevNull
}
