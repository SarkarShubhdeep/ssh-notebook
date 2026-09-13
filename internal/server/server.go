// Package server wires up the Wish SSH server and bridges each session into
// the Bubble Tea notebook TUI.
package server

import (
	"net"

	"github.com/SarkarShubhdeep/ssh-notebook/internal/content"
	"github.com/SarkarShubhdeep/ssh-notebook/internal/tui"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/ssh"
	"github.com/charmbracelet/wish"
	"github.com/charmbracelet/wish/activeterm"
	bm "github.com/charmbracelet/wish/bubbletea"
	lm "github.com/charmbracelet/wish/logging"
)

// Config configures the SSH notebook server.
type Config struct {
	Host        string // bind address, e.g. "0.0.0.0"
	Port        string // e.g. "2222"
	HostKeyPath string // path to the persisted Ed25519 host key
	Password    string // v0.1 placeholder; empty means allow any password
	NotebookDir string // directory of Markdown files to serve
}

// New builds a configured (but not yet listening) Wish SSH server.
func New(cfg Config) (*ssh.Server, error) {
	store := content.New(cfg.NotebookDir)

	return wish.NewServer(
		wish.WithAddress(net.JoinHostPort(cfg.Host, cfg.Port)),
		wish.WithHostKeyPath(cfg.HostKeyPath),
		wish.WithPasswordAuth(func(_ ssh.Context, password string) bool {
			// v0.1: single shared password. Empty config password = open
			// (handy for local dev). Real auth arrives in v0.2.
			if cfg.Password == "" {
				return true
			}
			return password == cfg.Password
		}),
		wish.WithMiddleware(
			bm.Middleware(teaHandler(store)),
			activeterm.Middleware(), // require an interactive PTY
			lm.Middleware(),         // connection logging
		),
	)
}

// teaHandler returns a Bubble Tea program for each SSH session.
func teaHandler(store *content.Store) bm.Handler {
	return func(s ssh.Session) (tea.Model, []tea.ProgramOption) {
		_, _, active := s.Pty()
		if !active {
			wish.Fatalln(s, "no active terminal; this app requires an interactive session")
			return nil, nil
		}

		renderer := bm.MakeRenderer(s)

		// Deep link: `ssh host notes/idea.md` opens that file directly.
		var initial string
		if args := s.Command(); len(args) > 0 {
			initial = args[0]
		}

		m := tui.New(store, renderer, initial)
		return m, []tea.ProgramOption{tea.WithAltScreen()}
	}
}
