// Package tui implements the Bubble Tea terminal UI for the notebook:
// a file list plus a Glamour-rendered Markdown viewer. v0.1 is read-only.
package tui

import (
	"fmt"

	"github.com/SarkarShubhdeep/ssh-notebook/internal/content"
	"github.com/charmbracelet/bubbles/list"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
)

type mode int

const (
	modeList mode = iota
	modeView
)

// item is a single Markdown file in the list.
type item struct{ path string }

func (i item) Title() string       { return i.path }
func (i item) Description() string { return "" }
func (i item) FilterValue() string { return i.path }

// Model is the root Bubble Tea model for one SSH session.
type Model struct {
	store    *content.Store
	renderer *lipgloss.Renderer
	list     list.Model
	viewport viewport.Model
	mode     mode
	width    int
	height   int
	current  string
	initial  string
	ready    bool
	err      error
}

// New builds a Model. If initial is non-empty, that file is opened on start
// (used for deep links like `ssh host notes/idea.md`).
func New(store *content.Store, renderer *lipgloss.Renderer, initial string) Model {
	delegate := list.NewDefaultDelegate()
	delegate.ShowDescription = false

	l := list.New(listItems(store), delegate, 0, 0)
	l.Title = "Notebook"
	l.SetShowStatusBar(false)

	return Model{
		store:    store,
		renderer: renderer,
		list:     l,
		mode:     modeList,
		initial:  initial,
	}
}

func listItems(store *content.Store) []list.Item {
	files, err := store.List()
	if err != nil {
		return nil
	}
	items := make([]list.Item, len(files))
	for i, f := range files {
		items[i] = item{path: f}
	}
	return items
}

// Init implements tea.Model.
func (m Model) Init() tea.Cmd { return nil }

// Update implements tea.Model.
func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.list.SetSize(msg.Width, msg.Height)
		if !m.ready {
			m.viewport = viewport.New(msg.Width, max(msg.Height-2, 1))
			m.ready = true
			if m.initial != "" {
				return m.openFile(m.initial), nil
			}
		} else {
			m.viewport.Width = msg.Width
			m.viewport.Height = max(msg.Height-2, 1)
			if m.mode == modeView && m.current != "" {
				return m.openFile(m.current), nil
			}
		}
		return m, nil

	case tea.KeyMsg:
		switch m.mode {
		case modeList:
			// Let the list own typing while filtering.
			if m.list.FilterState() == list.Filtering {
				break
			}
			switch msg.String() {
			case "q", "ctrl+c":
				return m, tea.Quit
			case "enter":
				if it, ok := m.list.SelectedItem().(item); ok {
					return m.openFile(it.path), nil
				}
			}
		case modeView:
			switch msg.String() {
			case "ctrl+c":
				return m, tea.Quit
			case "q", "esc", "backspace":
				m.mode = modeList
				m.err = nil
				return m, nil
			}
		}
	}

	var cmd tea.Cmd
	switch m.mode {
	case modeList:
		m.list, cmd = m.list.Update(msg)
	case modeView:
		m.viewport, cmd = m.viewport.Update(msg)
	}
	return m, cmd
}

// openFile loads and renders a Markdown file, switching to view mode.
func (m Model) openFile(path string) Model {
	raw, err := m.store.Read(path)
	if err != nil {
		m.err = err
		m.mode = modeView
		m.current = path
		m.viewport.SetContent(m.renderer.NewStyle().Render(fmt.Sprintf("could not open %q: %v", path, err)))
		return m
	}

	width := m.width
	if width <= 0 {
		width = 80
	}
	out, err := renderMarkdown(raw, width)
	if err != nil {
		out = raw // fall back to unrendered text
	}
	m.viewport.SetContent(out)
	m.viewport.GotoTop()
	m.current = path
	m.mode = modeView
	m.err = nil
	return m
}

func renderMarkdown(raw string, width int) (string, error) {
	r, err := glamour.NewTermRenderer(
		glamour.WithStandardStyle("dark"),
		glamour.WithWordWrap(max(width-2, 20)),
	)
	if err != nil {
		return "", err
	}
	return r.Render(raw)
}

// View implements tea.Model.
func (m Model) View() string {
	if !m.ready {
		return "\n  Initializing…\n"
	}
	if m.mode == modeView {
		header := m.renderer.NewStyle().Bold(true).Render(m.current)
		footer := m.renderer.NewStyle().Faint(true).Render("↑/↓ scroll · q/esc back")
		return fmt.Sprintf("%s\n%s\n%s", header, m.viewport.View(), footer)
	}
	return m.list.View()
}
