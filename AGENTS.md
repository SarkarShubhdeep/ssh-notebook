# AGENTS.md

Guidance for AI agents working in this repo.

## Project

`ssh-notebook`: an SSH server (Go + Charm stack) that serves a personal Markdown notebook as a Terminal UI. Read [`docs/masterplan.md`](docs/masterplan.md) first — it has the architecture, MVP milestones, and open decision points.

## Stack

- Go + [Wish](https://github.com/charmbracelet/wish) (SSH) + [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI) + [Lip Gloss](https://github.com/charmbracelet/lipgloss) + [Glamour](https://github.com/charmbracelet/glamour) (Markdown).
- Content is a git-backed repo of `.md` files.

## Conventions

- See [`.cursor/rules/charm-ssh-tui.mdc`](.cursor/rules/charm-ssh-tui.mdc) for coding conventions.
- Never commit secrets (password hashes, deploy keys, PATs).
- Never serve edit/write actions to public/anonymous sessions.
- Work in thin vertical slices matching the MVP milestones (v0.1 → v0.6).

## Build / run

```sh
brew install go        # Go is not yet installed on this machine
go run ./cmd/ssh       # once the server is scaffolded
```

## Open decisions

Several design tensions are unresolved (edit-vs-pull policy, content repo visibility, push credentials, public surface UX, domain). See §7 of the master plan and confirm with the user before implementing those areas.
