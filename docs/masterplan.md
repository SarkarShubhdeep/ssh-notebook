# SSH Notebook — Master Plan

> A single command — `ssh notebook.<yourdomain>` — that opens your personal notebook (a directory of Markdown files) in any terminal, on any machine, with zero install for the reader. Private by default (password + your SSH keys), with opt-in public pages that need no auth.

Inspired by [terminal.shop](https://www.terminal.shop): SSH is used as an *application delivery protocol*, not a remote shell. The user's terminal is just a renderer; the notebook app runs server-side.

---

## 1. Vision & Goals

**What it is:** A **universal clipboard / notebook**. An SSH server that, instead of giving you a shell, drops you into a Terminal UI (TUI) to browse, read, and **edit** a tree of Markdown files from any machine. Sit down at a friend's computer, `ssh notebook.<yourdomain>`, enter your password, edit a note, save — done. Public notes need no password.

The mental model is a personal scratchpad you can reach from anywhere, not a git workflow. Files live on the server; you just edit and save.

**Why SSH:**
- Zero install for readers — every OS already ships an `ssh` client.
- Identity is free — your SSH key *is* your login.
- Encrypted transport, no TLS certs to manage.
- Resilient over flaky connections; scriptable (`ssh notebook.me notes/x.md`).
- High cool-factor.

**Non-goals (for now):** multi-tenant SaaS, a web frontend, real-time collaboration, rich media beyond terminal-capable image protocols.

---

## 2. Confirmed Decisions

| Area | Decision |
|------|----------|
| Language / stack | **Go + Charm**: [Wish](https://github.com/charmbracelet/wish) (SSH), [Bubble Tea](https://github.com/charmbracelet/bubbletea) (TUI), [Lip Gloss](https://github.com/charmbracelet/lipgloss) (styling), [Glamour](https://github.com/charmbracelet/glamour) (Markdown rendering) |
| Content store | **Plain files on a Fly.io persistent volume** (primary). Edits write straight to disk. |
| Backup / history | **Async, off the save path**: Fly daily volume snapshots + periodic git push (or R2/S3 sync). GitHub is a *backup drive*, not the live store. |
| Auth | **Password for private access + whitelist of my SSH public keys**; a separate **public** entry (subdomain/command) with no auth |
| MVP scope | Viewer **+ in-terminal editing that writes back to disk** |
| Repo | Public GitHub repo `ssh-notebook` (app code); notebook content backup goes to a **separate private repo** |
| Hosting | **Fly.io** (see §6) |

---

## 3. High-Level Architecture

```mermaid
flowchart TB
  subgraph client["Any terminal"]
    U["ssh notebook.me  /  ssh public.notebook.me"]
  end

  subgraph server["SSH Notebook server (Go)"]
    W["Wish SSH server :22"]
    AUTH["Auth middleware\npassword + key whitelist\n+ public bypass"]
    ROUTE["Command/route parser\n(deep links, edit mode)"]
    TUI["Bubble Tea TUI\ntree + viewer + editor"]
    REND["Glamour renderer"]
    STORE["Content store\nfiles on Fly volume"]
    BK["Backup worker\n(async, off save path)"]

    W --> AUTH --> ROUTE --> TUI
    TUI --> REND
    TUI -->|read / write| STORE
    STORE --> BK
  end

  subgraph ext["External (backup only)"]
    GH["Private content repo\n(GitHub)"]
    OBJ["Object storage\n(R2 / S3, optional)"]
  end

  U -->|encrypted PTY| W
  BK -->|periodic push, one-way| GH
  BK -.->|optional sync| OBJ
```

### Session lifecycle
1. `ssh` client connects; server presents a pinned host key.
2. Auth middleware decides: public route → allow anonymous read-only; private route → require password **or** a whitelisted public key fingerprint.
3. Wish allocates a PTY and starts a per-session Bubble Tea program.
4. The route parser inspects the requested command (`ssh notebook.me notes/x.md`, `-t edit ...`) and sets the initial view.
5. TUI reads files directly from the Fly volume; Glamour renders Markdown to styled ANSI.
6. Edits (authenticated only) write straight to disk — the save is complete at that point.
7. A background worker later backs up the volume to GitHub/object storage; this never blocks a save and, since nothing else writes the volume, never conflicts with edits.

---

## 4. Component Breakdown

| Component | Responsibility | Key packages |
|-----------|----------------|--------------|
| SSH server | Accept connections, PTY, host key, middleware stack | `wish`, `gliderlabs/ssh` |
| Auth | Password check, key-fingerprint whitelist, public bypass, per-session identity/permissions | `wish`, `crypto/ssh` |
| Router | Map SSH command args to initial view + mode (read/edit) | stdlib |
| TUI shell | App state, layout, key bindings, responsive sizing | `bubbletea`, `lipgloss`, `bubbles` |
| File tree | Navigate the notebook directory | `bubbles/list` or custom tree |
| Viewer | Render Markdown, scroll, code highlighting | `glamour`, `bubbles/viewport` |
| Editor | Edit buffer, save-back (auth only) | `bubbles/textarea` |
| Search | Filename + full-text search | `bleve` or simple walk + grep |
| Content store | Read/write notebook files on the Fly volume | stdlib `os` / `io/fs` |
| Backup worker | Periodic one-way push to GitHub / object storage; runs off the save path | `go-git` or shell `git`, or `rsync`/S3 SDK |
| Config | Domains, key whitelist, password hash, backup target, public paths | env + small config file |
| Ops | Structured logging, panic recovery, metrics | `wish/logging`, `slog` |

---

## 5. MVP Segmentation

Ship in thin vertical slices; each is independently demoable.

### v0.1 — "Hello, notebook" (read-only, local dir)
- Wish server with password auth + host key persistence.
- Bubble Tea shell: file tree of a local directory (the future volume mount).
- Glamour renders a selected `.md`; viewport scrolling.
- Deep link: `ssh host path/to/file.md` opens that file.
- **Exit criteria:** connect, browse, read a note over SSH locally.

### v0.2 — Auth model
- Password for private access (hashed, from config/secret).
- SSH public-key whitelist → authenticated identity, no password.
- Public bypass: `public.` subdomain **or** a `public` command → read-only, no auth, restricted to a `public/` subtree.
- **Exit criteria:** private needs password/key; public path is open and sandboxed.

### v0.3 — Editing that writes to disk
- Editor view (auth only) with `bubbles/textarea`.
- Save → write file directly to the notebook directory. That's the whole save.
- Guard rails: no editing on public/anonymous routes; safe path handling (no escaping the notebook root).
- **Exit criteria:** edit a note in-terminal, save, reconnect → change persisted.

### v0.4 — Persistence + backup
- Mount a Fly persistent volume as the notebook directory.
- Background backup worker: periodic one-way push to a private GitHub repo (and/or R2/S3). Never blocks a save.
- **Exit criteria:** data survives a machine restart/redeploy; backup lands in GitHub.

### v0.5 — Search & polish
- Filename fuzzy find + full-text search.
- Lip Gloss theming, help bar, responsive breakpoints, resize handling.
- **Exit criteria:** find any note by name/content; UI feels finished.

### v0.6 — Deploy
- Containerize; deploy to chosen host (see §6); DNS + host-key publication.
- **Exit criteria:** `ssh notebook.<yourdomain>` works from a fresh machine.

### Later / stretch
- Inline images (Kitty/iTerm2 protocols) for terminals that support them.
- Multiple notebooks / namespaces.
- Audit log, rate limiting, fail2ban-style protection.
- Per-file public sharing (mark front-matter `public: true`).

---

## 6. Hosting — Fly.io (chosen)

Fly.io was chosen for git-based deploys, persistent volumes, dedicated IPv4 for raw port 22, and low ops. Comparison retained for reference:


| Option | Est. cost | SSH/TCP :22 | Persistent disk | Deploy UX | Ops burden | Best when |
|--------|-----------|-------------|-----------------|-----------|------------|-----------|
| **Fly.io** | ~$5–15/mo | Yes (dedicated IPv4 ~$2/mo) | Volumes | `fly deploy` from git, Dockerfile | Low | You want push-to-deploy + minimal ops **(recommended)** |
| **VPS** (Hetzner ~€4, DO $6) | $5–8/mo | Yes, full control | Native disk | `git pull` + `systemd` service, or Docker | Medium (patching, firewall) | You want full control & lowest cost |
| **Railway / Render** | $5–20/mo | TCP proxy varies — verify raw :22 support | Volumes | Git-based | Low | You prefer a PaaS but confirm raw TCP first |
| **AWS ECS Fargate** | $15–40/mo | Yes (NLB) | EFS/S3 sync | IaC (SST/Terraform) | High | You want terminal.shop-grade scaling (overkill for solo) |
| **Home server + tunnel** | ~$0 | Via Cloudflare Tunnel / Tailscale | Local | Manual | Medium | Free, hobby, you already run a box |

**Recommendation:** Start on **Fly.io** (git deploy, persistent volume, straightforward dedicated IP for port 22). Revisit a VPS if you want to shave cost or need custom networking.

> ⚠️ **Port 22 note:** Whatever host you pick, verify it exposes a *raw TCP* port for SSH (not just HTTP). PaaS platforms that only proxy HTTP won't work; you may need to listen on a non-22 port and map it, or use the platform's TCP service feature.

---

## 7. Key Design Tensions & Decision Points

These need your input before/at the relevant milestone:

1. **✅ Resolved — storage model.** Server volume is the single writer; git/object-storage is one-way backup only. No pull-vs-edit conflict. (Decided: filesystem-primary, GitHub as backup.)

2. **Backup target & cadence (v0.4).** Private GitHub repo, object storage (R2/S3), or both? How often — every N minutes, or debounced a few seconds after each save? **Decision:** pick target + cadence. (Recommended: private GitHub repo, debounced ~30s after last edit.)

3. **Backup credentials.** One-way push needs write access: a **deploy key** (SSH) or a **fine-grained PAT**, stored as a Fly secret. **Decision:** which credential type?

4. **Password storage.** Single shared password (hashed, e.g. bcrypt) vs per-user. MVP = single hashed password in a secret. **Decision:** confirm single-password MVP.

5. **Public surface.** Public via a `public.` **subdomain** (needs extra DNS + cert-free SSH listener) vs a `public` **command/arg** on the same host. Command-arg is simpler. **Decision:** which UX?

6. **Host key trust.** Publish the server's Ed25519 host key (like terminal.shop does on its homepage) so users can pin it. **Decision:** where do you publish it (README? a landing page?).

7. **Domain.** What hostname? (`notebook.<yourdomain>`?) Needed for DNS + host-key setup at v0.6.

---

## 8. Repo Layout (proposed)

```
ssh-notebook/
├── cmd/
│   └── ssh/main.go          # server entrypoint
├── internal/
│   ├── server/              # wish setup, middleware, host key
│   ├── auth/                # password + key whitelist + public bypass
│   ├── tui/                 # bubbletea model: tree, viewer, editor
│   ├── content/             # fs read/write over notebook dir
│   └── backup/              # one-way push to GitHub / object storage
├── docs/
│   └── masterplan.md        # this file
├── config.example.yaml
├── Dockerfile
├── go.mod
└── README.md
```

---

## 9. Milestone Checklist

- [ ] v0.1 Read-only viewer over SSH (local dir)
- [ ] v0.2 Auth model (password + key whitelist + public bypass)
- [ ] v0.3 In-terminal editing that writes to disk
- [ ] v0.4 Fly persistent volume + async backup
- [ ] v0.5 Search + UI polish
- [ ] v0.6 Deploy + DNS + published host key

---

## 10. Prerequisites

- Install Go (not currently on this machine): `brew install go`
- GitHub CLI (present, authenticated as `SarkarShubhdeep`)
- A domain with DNS you control (for v0.6)
- A host account (Fly.io / VPS) for v0.6
