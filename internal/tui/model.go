package tui

import (
	"context"
	"fmt"
	"strings"

	"github.com/alzubair-alojali/hova/internal/inventory"
	sshclient "github.com/alzubair-alojali/hova/internal/ssh"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

type Model struct {
	store                 inventory.Store
	servers               []inventory.Server
	search                textinput.Model
	width, height, cursor int
	mode, message         string
	inputs                []textinput.Model
	focus                 int
	original              inventory.Server
	editing               bool
}

type sessionEnded struct{ err error }

func New(store inventory.Store, servers []inventory.Server) Model {
	search := textinput.New()
	search.Prompt = "/  "
	search.Placeholder = "Search servers, addresses, tags…"
	search.CharLimit = 128
	search.Width = 50
	search.PromptStyle = accent
	search.TextStyle = normal
	search.PlaceholderStyle = muted
	return Model{store: store, servers: servers, search: search, width: 100, height: 32, mode: "browse"}
}

func Run(store inventory.Store) error { return run(store, nil, false) }
func RunForm(store inventory.Store, s inventory.Server, editing bool) error {
	return run(store, &s, editing)
}
func run(store inventory.Store, s *inventory.Server, editing bool) error {
	if _, present := lookupNoColor(); present {
		lipgloss.SetColorProfile(termenv.Ascii)
	}
	servers, err := store.Load()
	if err != nil {
		return err
	}
	m := New(store, servers)
	if s != nil {
		m.startForm(*s, editing)
	}
	_, err = tea.NewProgram(&m, tea.WithAltScreen()).Run()
	return err
}

func (m *Model) Init() tea.Cmd { return textinput.Blink }
func (m *Model) filtered() []inventory.Server {
	return inventory.Filter(m.servers, m.search.Value(), "")
}

func (m *Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
		m.search.Width = max(8, msg.Width-12)
		return m, nil
	case sessionEnded:
		if msg.err != nil {
			m.message = "SSH: " + msg.err.Error()
		} else {
			m.message = "Session closed. Welcome back."
		}
		return m, nil
	case tea.KeyMsg:
		if msg.Type == tea.KeyCtrlC {
			return m, tea.Quit
		}
		if m.height < 18 || m.width < 24 {
			if msg.String() == "q" {
				return m, tea.Quit
			}
			return m, nil
		}
		if m.mode == "form" {
			return m, m.updateForm(msg)
		}
		if m.mode == "delete" {
			return m, m.updateDelete(msg)
		}
		if m.search.Focused() {
			return m, m.updateSearch(msg)
		}
		return m, m.updateBrowse(msg)
	}
	if m.mode == "form" && len(m.inputs) > 0 {
		var cmd tea.Cmd
		m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
		return m, cmd
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	return m, cmd
}

func (m *Model) updateSearch(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.search.Blur()
		return nil
	case "enter":
		m.search.Blur()
		return m.connect()
	case "down", "ctrl+n":
		m.move(1)
		return nil
	case "up", "ctrl+p":
		m.move(-1)
		return nil
	}
	var cmd tea.Cmd
	m.search, cmd = m.search.Update(msg)
	m.cursor = 0
	return cmd
}

func (m *Model) updateBrowse(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "q":
		return tea.Quit
	case "up", "k":
		m.move(-1)
	case "down", "j":
		m.move(1)
	case "home", "g":
		m.cursor = 0
	case "end", "G":
		m.cursor = max(0, len(m.filtered())-1)
	case "/":
		return m.search.Focus()
	case "esc":
		m.search.Reset()
		m.cursor = 0
	case "enter":
		return m.connect()
	case "a":
		m.startForm(inventory.Server{Port: 22}, false)
		return textinput.Blink
	case "e":
		if s, ok := m.selected(); ok {
			m.startForm(s, true)
			return textinput.Blink
		}
	case "d":
		if s, ok := m.selected(); ok {
			m.original = s
			m.mode = "delete"
			m.message = ""
		}
	case "r":
		m.reload()
	}
	return nil
}

func (m *Model) move(delta int) { m.cursor = max(0, min(m.cursor+delta, len(m.filtered())-1)) }
func (m *Model) selected() (inventory.Server, bool) {
	items := m.filtered()
	if len(items) == 0 {
		return inventory.Server{}, false
	}
	m.cursor = min(m.cursor, len(items)-1)
	return items[m.cursor], true
}
func (m *Model) reload() {
	items, err := m.store.Load()
	if err != nil {
		m.message = err.Error()
		return
	}
	m.servers = items
	m.cursor = min(m.cursor, max(0, len(m.filtered())-1))
}

func (m *Model) connect() tea.Cmd {
	s, ok := m.selected()
	if !ok {
		return nil
	}
	plan, err := sshclient.Plan(s, "", false)
	if err != nil {
		m.message = err.Error()
		return nil
	}
	m.message = ""
	return tea.ExecProcess(plan.Process(context.Background()), func(err error) tea.Msg { return sessionEnded{err: err} })
}

func (m *Model) updateDelete(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc", "n":
		m.mode = "browse"
	case "enter", "y":
		name := m.original.Name
		err := m.store.Update(func(items []inventory.Server) ([]inventory.Server, error) {
			for i, s := range items {
				if strings.EqualFold(s.Name, name) {
					return append(items[:i], items[i+1:]...), nil
				}
			}
			return nil, fmt.Errorf("server %q no longer exists", name)
		})
		if err != nil {
			m.message = err.Error()
			return nil
		}
		m.mode = "browse"
		m.message = "Removed " + name + " from Hova"
		m.reload()
	}
	return nil
}
