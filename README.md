# ssh-notebook

Open your personal notebook — a directory of Markdown files — in **any terminal** with a single command:

```sh
ssh notebook.<yourdomain>
```

No install for the reader. SSH is used as an application delivery protocol (à la [terminal.shop](https://www.terminal.shop)): your terminal is just a renderer, the notebook app runs server-side. Private by default (password + your SSH keys), with opt-in public pages that need no auth.

## Status

🚧 Early development. See the [Master Plan](docs/masterplan.md) for architecture, MVP breakdown, hosting comparison, and open decision points.

## Stack

- **[Wish](https://github.com/charmbracelet/wish)** — SSH server
- **[Bubble Tea](https://github.com/charmbracelet/bubbletea)** — TUI framework
- **[Lip Gloss](https://github.com/charmbracelet/lipgloss)** — styling
- **[Glamour](https://github.com/charmbracelet/glamour)** — Markdown rendering
- **Go** + a git-backed content repo

## Planned features (MVP)

- Browse a tree of Markdown notes over SSH
- Rendered Markdown with code highlighting
- Deep links: `ssh notebook.me notes/idea.md`
- Auth: password + whitelisted SSH keys; separate public (no-auth) surface
- In-terminal editing that commits & pushes back to the content repo
- Filename + full-text search

## Development

```sh
# Prerequisite: install Go
brew install go

# Run the server locally (once scaffolded)
go run ./cmd/ssh
```

## Host key

The server's Ed25519 host key will be published here so clients can pin it (TODO once deployed).

## License

TBD
