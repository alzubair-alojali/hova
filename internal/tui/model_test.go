package tui

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/alzubair-alojali/hova/internal/inventory"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

func sampleModel() Model {
	return New(inventory.Store{Path: filepath.Join("unused", "servers.json")}, []inventory.Server{
		{Name: "alpha", Host: "203.0.113.10", Port: 22},
		{Name: "beta", Host: "203.0.113.20", Port: 22, Tags: []string{"work"}},
	})
}

func TestSearchChangesSelectedServer(t *testing.T) {
	m := sampleModel()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("beta")})
	if items := m.filtered(); len(items) != 1 || items[0].Name != "beta" {
		t.Fatalf("search: %#v", items)
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.search.Focused() {
		t.Fatal("escape did not leave search")
	}
}

func TestFormsCancelWithoutSavingAndRejectInvalidInput(t *testing.T) {
	m := sampleModel()
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.mode != "form" {
		t.Fatal("add did not open form")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	if m.message == "" || m.mode != "form" {
		t.Fatal("invalid form accepted or lost")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if m.mode != "browse" || len(m.servers) != 2 {
		t.Fatal("cancel changed inventory")
	}
}

func TestLayoutFitsNarrowTerminalsAndEmptyStates(t *testing.T) {
	for _, width := range []int{24, 40, 60, 80, 120} {
		m := sampleModel()
		m.Update(tea.WindowSizeMsg{Width: width, Height: 30})
		for _, line := range strings.Split(m.View(), "\n") {
			if lipgloss.Width(line) > width {
				t.Errorf("width %d overflow: %q (%d)", width, line, lipgloss.Width(line))
			}
		}
	}
	m := New(inventory.Store{}, nil)
	if !strings.Contains(m.View(), "Add your first server") {
		t.Fatal("empty state missing next action")
	}
}

func TestFooterAndActiveFieldStayVisible(t *testing.T) {
	for _, height := range []int{18, 24, 30, 32, 40} {
		m := sampleModel()
		m.Update(tea.WindowSizeMsg{Width: 100, Height: height})
		if !strings.Contains(m.View(), "q quit") {
			t.Errorf("browse footer hidden at height %d", height)
		}
		m.startForm(inventory.Server{Port: 22}, false)
		for i := 0; i < 7; i++ {
			m.focus = i
			if !strings.Contains(m.View(), "esc cancel") {
				t.Errorf("form footer hidden at height %d field %d", height, i)
			}
		}
	}
}

func TestTinyTerminalDisablesActionsUntilResize(t *testing.T) {
	m := sampleModel()
	m.Update(tea.WindowSizeMsg{Width: 80, Height: 12})
	if !strings.Contains(m.View(), "Resize") {
		t.Fatal("tiny terminal lacks resize guidance")
	}
	_, cmd := m.Update(tea.KeyMsg{Type: tea.KeyEnter})
	if cmd != nil {
		t.Fatal("blind connection allowed")
	}
	m.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	if m.mode != "browse" {
		t.Fatal("hidden form opened")
	}
}

func TestStaleFormDoesNotOverwriteNewerConnection(t *testing.T) {
	store := inventory.Store{Path: filepath.Join(t.TempDir(), "servers.json")}
	original := inventory.Server{Name: "prod", Host: "203.0.113.10", Port: 22}
	store.Update(func(items []inventory.Server) ([]inventory.Server, error) { return []inventory.Server{original}, nil })
	m := New(store, []inventory.Server{original})
	m.startForm(original, true)
	m.inputs[6].SetValue("my notes")
	store.Update(func(items []inventory.Server) ([]inventory.Server, error) {
		items[0].Host = "203.0.113.20"
		return items, nil
	})
	m.saveForm()
	items, err := store.Load()
	if err != nil || items[0].Host != "203.0.113.20" || m.mode != "form" || m.message == "" {
		t.Fatalf("stale form overwrote newer connection: %#v %v", items, err)
	}
}
