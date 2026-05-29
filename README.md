# Previously on...

A developer workflow CLI that briefs you on what changed in a Git repository since the last time it was seen.

It is meant to feel like a repo-specific changelog: short, factual, and useful when you come back to a project after time away.

## What It Does

- Detects the current Git repository.
- Stores local last-seen repo state under `~/.previously-on/`.
- Stores user config under `~/.config/previously-on/config.json` or `$XDG_CONFIG_HOME/previously-on/config.json`.
- Compares current Git state with the last seen state.
- Generates a terminal-rendered Markdown briefing.
- Uses AI when configured, with a non-AI factual fallback.
- Can install local Git hooks to show a briefing after successful merge/rebase from pulls.

## Install

Install the latest release:

```bash
curl -fsSL https://raw.githubusercontent.com/romerramos/previously_on/main/install.sh | sh
```

Install to a custom directory:

```bash
curl -fsSL https://raw.githubusercontent.com/romerramos/previously_on/main/install.sh | sh -s -- --dir ~/.local/bin
```

Install a specific version:

```bash
curl -fsSL https://raw.githubusercontent.com/romerramos/previously_on/main/install.sh | sh -s -- --version v0.1.0
```

If the installer uses `~/.local/bin`, make sure that directory is in your `PATH`.

After installing the binary, the installer runs `previously-on connect` so you can configure an AI provider. To skip provider setup during install:

```bash
curl -fsSL https://raw.githubusercontent.com/romerramos/previously_on/main/install.sh | sh -s -- --no-connect
```

You can also skip provider setup with `PREVIOUSLY_ON_SKIP_CONNECT=1`.

Update by running the installer again. It overwrites the existing binary in the target directory:

```bash
curl -fsSL https://raw.githubusercontent.com/romerramos/previously_on/main/install.sh | sh
```

Uninstall the binary:

```bash
rm "$(command -v previously-on)"
```

If you installed to the default user path, you can also remove it directly:

```bash
rm ~/.local/bin/previously-on
```

Optional cleanup:

```bash
previously-on uninstall nvim
previously-on uninstall git-hook
previously-on disconnect --all
previously-on reset --all
```

## Install For Development

Requirements:

- Go `1.25.8` or newer
- Git

Build locally:

```bash
go build -o previously-on .
```

Run from source while developing:

```bash
go run . --help
```

To run this source checkout against another Git repo without installing a binary, build once into a temp or bin path. `go run /path/to/previously_on` from another repo will not work reliably unless that repo is also a Go module.

## Basic Usage

From inside any Git repository:

```bash
previously-on summary
```

First run creates a baseline and summarizes recent history. Future runs only summarize changes after the stored baseline.

Useful commands:

```bash
previously-on status
previously-on summary
previously-on mark-seen
previously-on reset
previously-on config
```

## Report Detail Levels

`summary` defaults to a short report:

```bash
previously-on summary --simple
```

Other modes:

```bash
previously-on summary --medium
previously-on summary --detailed
```

Simple reports focus on:

- TL;DR, when AI is available
- things worth checking, when AI is available
- commits included

Medium and detailed reports include more factual context such as changed files, notable branch activity, recent commits, and diff stats.

## Markdown Output

Interactive terminal output is rendered with Glamour using `tokyo-night` by default.

Raw Markdown is still available:

```bash
previously-on summary --raw
```

Write raw Markdown to a file:

```bash
previously-on summary --output previously-on.md
```

Change render style or width:

```bash
previously-on summary --render-style dark
previously-on summary --render-width 120
```

Piped output is raw Markdown automatically.

## AI Providers

Connect an AI provider with the TUI:

```bash
previously-on connect
```

List providers:

```bash
previously-on connect --list
```

Supported providers:

- OpenAI
- Anthropic
- Gemini
- OpenRouter
- Ollama

API keys are stored in the OS keychain, not in the dotfiles-friendly config file.

Environment variables take precedence over keychain values:

```bash
OPENAI_API_KEY
ANTHROPIC_API_KEY
GEMINI_API_KEY
OPENROUTER_API_KEY
```

Ollama does not require an API key.

Set or change a model name:

```bash
previously-on models
previously-on models --provider openai --set gpt-4o-mini
```

Model names are not validated remotely. Double-check them in the provider docs or dashboard.

Disconnect credentials:

```bash
previously-on disconnect
previously-on disconnect --provider openai
previously-on disconnect --all
```

## No-AI And Offline Mode

Run without AI:

```bash
previously-on summary --no-ai
```

Avoid network-dependent behavior:

```bash
previously-on summary --offline
```

Environment override:

```bash
PREVIOUSLY_ON_NO_AI=1 previously-on summary
```

Without AI, reports are intentionally factual and avoid analytical sections that require judgment.

## Git Hook Integration

Install local Git hooks in the current repo:

```bash
previously-on install git-hook
```

This installs managed blocks into:

- `post-merge`
- `post-rewrite`

Those hooks run:

```bash
previously-on summary --simple
```

The hook is intended to show a briefing after successful merge/rebase activity, including common `git pull` flows.

Temporarily skip hook output:

```bash
PREVIOUSLY_ON_SKIP_HOOK=1 git pull
```

Uninstall the managed hook blocks:

```bash
previously-on uninstall git-hook
```

The installer does not overwrite existing hooks. It appends a managed block and later removes only that block.

## Neovim Integration

Install the native Neovim package:

```bash
previously-on install nvim --strategy native
```

This writes a managed plugin to Neovim's package path under `~/.local/share/nvim/site/pack/previously-on/start/previously-on.nvim/`.

If you use `lazy.nvim` or LazyVim with `noloadplugins`, install with the lazy strategy instead:

```bash
previously-on install nvim --strategy lazy
```

The lazy strategy writes the same managed plugin package plus a managed local plugin spec under `~/.config/nvim/lua/plugins/previously-on.lua`.

If `--strategy` is omitted, the installer opens a small picker so you can choose the package strategy. In scripts, pass `--strategy native` or `--strategy lazy` explicitly.

When Neovim opens inside a Git repository, the plugin opens a scratch Markdown buffer named `previously-on://summary.md`. The buffer is treated as Markdown and is not written to disk.

Useful Neovim commands:

```vim
:PreviouslyOn
:PreviouslyOnRefresh
```

The plugin runs `previously-on summary --raw --simple`, so opening Neovim updates the repo's last-seen state after generating the summary.

Uninstall the managed plugin:

```bash
previously-on uninstall nvim
```

## Privacy

Avoids sending full source code to AI providers.

AI payloads are built from:

- commit authors and subjects
- changed file paths
- file categories
- risk signals derived from paths and commit messages

API keys are stored in the OS keychain via `go-keyring` unless provided by environment variables.

Config is stored under `~/.config/previously-on/` so it can be managed with dotfiles. It should not contain secrets.

## Local State

Repo state lives under:

```text
~/.previously-on/
```

Config lives under:

```text
~/.config/previously-on/config.json
```

or:

```text
$XDG_CONFIG_HOME/previously-on/config.json
```

## Current Limitations

- First run summarizes recent repo history, then stores a baseline. Later runs summarize only changes since that baseline.
- If the stored baseline commit disappears because of rebases, shallow clones, or rewritten history, the tool falls back to timestamp-based inspection.
- Git hook support covers merge/rebase flows through `post-merge` and `post-rewrite`; Git has no standard `post-fetch` hook.

## Releasing

Releases are built by GitHub Actions when a version tag is pushed.

```bash
git tag v0.1.0
git push origin v0.1.0
```

The release workflow runs tests, builds Linux and macOS binaries for `amd64` and `arm64`, creates a GitHub Release, and uploads archives plus `checksums.txt`.

The public installer downloads those release assets:

```bash
curl -fsSL https://raw.githubusercontent.com/romerramos/previously_on/main/install.sh | sh
```

Do not commit built binaries to the repository. Release binaries belong in GitHub Releases.

## Development

Run tests:

```bash
go test ./...
```

Format code:

```bash
gofmt -w .
```
