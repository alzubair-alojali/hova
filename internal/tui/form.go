package tui

import (
	"fmt"
	"reflect"
	"strconv"
	"strings"

	"github.com/alzubair-alojali/hova/internal/inventory"
	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
)

var fieldLabels = []string{"Name", "Host / IP", "SSH user", "Port", "Identity file", "Tags", "Notes"}
var placeholders = []string{"production", "203.0.113.10 or server.example.com", "Leave empty to use SSH defaults", "22", "~/.ssh/id_ed25519 (optional)", "work, production", "What is this server for?"}

func (m *Model) startForm(s inventory.Server, editing bool) {
	m.mode = "form"
	m.message = ""
	m.editing = editing
	m.original = s
	m.focus = 0
	values := []string{s.Name, s.Host, s.User, strconv.Itoa(s.Port), s.IdentityFile, strings.Join(s.Tags, ", "), s.Notes}
	m.inputs = make([]textinput.Model, len(fieldLabels))
	for i := range m.inputs {
		input := textinput.New()
		input.Prompt = "  "
		input.Placeholder = placeholders[i]
		input.CharLimit = 4096
		input.Width = max(8, m.width-16)
		input.TextStyle = normal
		input.PlaceholderStyle = muted
		input.PromptStyle = accent
		input.SetValue(values[i])
		m.inputs[i] = input
	}
	m.inputs[0].Focus()
}

func (m *Model) updateForm(msg tea.KeyMsg) tea.Cmd {
	switch msg.String() {
	case "esc":
		m.mode = "browse"
		m.message = ""
		return nil
	case "ctrl+s":
		m.saveForm()
		return nil
	case "tab", "enter", "down":
		if m.focus == len(m.inputs)-1 && msg.String() == "enter" {
			m.saveForm()
			return nil
		}
		m.inputs[m.focus].Blur()
		m.focus = (m.focus + 1) % len(m.inputs)
		return m.inputs[m.focus].Focus()
	case "shift+tab", "up":
		m.inputs[m.focus].Blur()
		m.focus = (m.focus + len(m.inputs) - 1) % len(m.inputs)
		return m.inputs[m.focus].Focus()
	}
	var cmd tea.Cmd
	m.inputs[m.focus], cmd = m.inputs[m.focus].Update(msg)
	return cmd
}

func (m *Model) saveForm() {
	s, err := m.formServer()
	if err != nil {
		m.message = err.Error()
		return
	}
	err = m.store.Update(func(items []inventory.Server) ([]inventory.Server, error) {
		if !m.editing {
			return append(items, s), nil
		}
		for i, item := range items {
			if strings.EqualFold(item.Name, m.original.Name) {
				if !reflect.DeepEqual(item, m.original) {
					return nil, fmt.Errorf("connection changed elsewhere; Esc to cancel, then r to refresh before editing again")
				}
				items[i] = s
				return items, nil
			}
		}
		return nil, fmt.Errorf("server %q no longer exists", m.original.Name)
	})
	if err != nil {
		m.message = err.Error()
		return
	}
	m.mode = "browse"
	m.search.Reset()
	m.message = "Saved " + s.Name
	m.reload()
	for i, item := range m.filtered() {
		if item.Name == s.Name {
			m.cursor = i
			break
		}
	}
}

func (m *Model) formServer() (inventory.Server, error) {
	value := func(i int) string { return strings.TrimSpace(m.inputs[i].Value()) }
	port, err := strconv.Atoi(value(3))
	if err != nil {
		return inventory.Server{}, fmt.Errorf("port must be a number between 1 and 65535")
	}
	var tags []string
	for _, tag := range strings.Split(value(5), ",") {
		if tag = strings.TrimSpace(tag); tag != "" {
			tags = append(tags, tag)
		}
	}
	s := inventory.Server{Name: value(0), Host: value(1), User: value(2), Port: port, IdentityFile: value(4), Tags: tags, Notes: value(6)}
	if m.editing && s.Host == m.original.Host && s.User == m.original.User && s.Port == m.original.Port && s.IdentityFile == m.original.IdentityFile {
		s.SSHAlias = m.original.SSHAlias
		s.SSHConfig = m.original.SSHConfig
	}
	return s, inventory.Validate(s)
}
