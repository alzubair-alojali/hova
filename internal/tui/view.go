package tui

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/x/ansi"
)

var accent = lipgloss.NewStyle().Foreground(lipgloss.Color("#8BE9BA"))
var normal = lipgloss.NewStyle().Foreground(lipgloss.Color("#E3EAE8"))
var muted = lipgloss.NewStyle().Foreground(lipgloss.Color("#84968F"))
var title = accent.Bold(true)
var box = lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("#344B42"))
var selectedStyle = lipgloss.NewStyle().Foreground(lipgloss.Color("#B9F5D6")).Background(lipgloss.Color("#183A2C"))

func lookupNoColor() (string, bool)   { return os.LookupEnv("NO_COLOR") }
func clip(s string, width int) string { return ansi.Truncate(s, max(1, width), "…") }

func (m *Model) View() string {
	width := max(1, m.width-4)
	if m.height < 18 || m.width < 24 {
		return clip("Hova · Resize to at least 24 columns × 18 rows. Ctrl+C to exit.", m.width)
	}
	top := title.Render("◈  hova") + muted.Render("  /  YOUR SSH WORKSPACE")
	if width >= 75 {
		top += strings.Repeat(" ", max(2, width-lipgloss.Width(top)-17)) + muted.Render("LOCAL · PRIVATE")
	}
	var body string
	switch m.mode {
	case "form":
		body = m.formView(width)
	case "delete":
		body = m.deleteView(width)
	default:
		body = m.browseView(width)
	}
	if lines := strings.Split(body, "\n"); len(lines) > m.height-8 {
		body = strings.Join(lines[:max(1, m.height-8)], "\n")
	}
	hint := "↑↓ navigate   / search   ↵ connect   a add   e edit   d remove   r refresh   q quit"
	if width < 90 {
		hint = "↑↓ move  / search  ↵ connect  a add  e edit  d remove  q quit"
	}
	if width < 60 {
		hint = "/ search  ↵ connect  a add  q quit"
	}
	if width < 40 {
		hint = "a add · q quit"
	}
	if m.mode == "form" {
		hint = "tab / shift+tab move   ctrl+s save   esc cancel"
		if width < 50 {
			hint = "^S save · esc cancel"
		}
	}
	if m.mode == "delete" {
		hint = "enter confirm   esc cancel"
	}
	status := muted.Render("Ready when you are.")
	if m.message != "" {
		status = accent.Render(clip(m.message, width))
	}
	content := top + "\n" + muted.Render("Your servers. One command away.") + "\n\n" + body + "\n" + status + "\n\n" + muted.Render(clip(hint, width))
	lines := strings.Split(content, "\n")
	for i, line := range lines {
		lines[i] = "  " + clip(line, width)
	}
	if len(lines) > m.height-1 {
		lines = lines[:m.height-1]
	}
	return strings.Join(lines, "\n")
}

func (m *Model) browseView(width int) string {
	m.search.Width = max(4, width-6)
	search := box.Width(width - 2).Render(" " + m.search.View())
	if len(m.servers) == 0 {
		empty := "\n  " + title.Render("A home for every server.") + "\n\n  Add your first server\n  Press a to save a name, address and SSH settings.\n\n  Already have an SSH config?\n  Run hova import ssh to bring your aliases in.\n"
		return search + "\n\n" + box.Width(width-2).Render(empty)
	}
	rows := max(1, (m.height-17)/3)
	if width < 76 || m.height < 26 {
		return search + "\n\n" + m.listView(width, rows)
	}
	left := width * 53 / 100
	right := width - left - 2
	return search + "\n\n" + lipgloss.JoinHorizontal(lipgloss.Top, m.listView(left, rows), "  ", m.detailView(right, rows*3+2))
}

func (m *Model) listView(width, rows int) string {
	items := m.filtered()
	header := muted.Render(fmt.Sprintf("  SERVERS  %02d / %02d", len(items), len(m.servers)))
	lines := []string{header, ""}
	start := 0
	if m.cursor >= rows {
		start = m.cursor - rows + 1
	}
	if len(items) == 0 {
		lines = append(lines, "  No matching servers.", "  Esc clears your search.")
	}
	for i := start; i < min(len(items), start+rows); i++ {
		s := items[i]
		prefix := "  "
		if i == m.cursor {
			prefix = "› "
		}
		name := prefix + s.Name
		meta := "  " + s.Host
		tags := "  " + strings.Join(s.Tags, " · ")
		if i == m.cursor {
			lines = append(lines, selectedStyle.Width(width-2).Render(clip(name, width-2)), selectedStyle.Width(width-2).Render(clip(meta, width-2)), selectedStyle.Width(width-2).Render(clip(tags, width-2)))
		} else {
			lines = append(lines, normal.Render(clip(name, width-2)), muted.Render(clip(meta, width-2)), muted.Render(clip(tags, width-2)))
		}
	}
	for len(lines) < rows*3+2 {
		lines = append(lines, "")
	}
	return box.Width(width - 2).Render(strings.Join(lines, "\n"))
}

func (m *Model) detailView(width, height int) string {
	s, ok := m.selected()
	if !ok {
		return box.Width(width - 2).Height(height).Render("  Select a server")
	}
	value := func(label, v string) string {
		if v == "" {
			v = "SSH default"
		}
		return "  " + muted.Render(label) + "\n  " + normal.Render(clip(v, width-5))
	}
	target := s.Host
	if s.User != "" {
		target = s.User + "@" + target
	}
	lines := []string{muted.Render("  CONNECTION PROFILE"), "", "  " + title.Render(clip(s.Name, width-5)), "  " + muted.Render(clip(target, width-5)), "", value("PORT", strconv.Itoa(s.Port)), "", value("IDENTITY", s.IdentityFile)}
	if s.SSHAlias != "" {
		lines = append(lines, "", value("SSH ALIAS", s.SSHAlias))
	} else if s.Notes != "" {
		lines = append(lines, "", value("NOTES", s.Notes))
	}
	lines = append(lines, "", "  "+selectedStyle.Padding(0, 1).Render("↵  Connect"))
	return box.Width(width - 2).Height(height).MaxHeight(height + 2).Render(strings.Join(lines, "\n"))
}

func (m *Model) formView(width int) string {
	heading := "Add a server"
	if m.editing {
		heading = "Edit " + m.original.Name
	}
	lines := []string{" " + title.Render(heading), " " + muted.Render("A memorable name. A direct connection."), ""}
	visible := max(1, (m.height-14)/3)
	start := max(0, m.focus-visible+1)
	for i := start; i < min(len(m.inputs), start+visible); i++ {
		label := muted.Render(fieldLabels[i])
		if i == m.focus {
			label = accent.Render("› " + fieldLabels[i])
		}
		m.inputs[i].Width = max(5, width-8)
		lines = append(lines, " "+label, " "+m.inputs[i].View(), "")
	}
	lines = append(lines, muted.Render(fmt.Sprintf(" Field %d of %d · Enter moves forward", m.focus+1, len(m.inputs))))
	return box.Width(width - 2).Render(strings.Join(lines, "\n"))
}

func (m *Model) deleteView(width int) string {
	return box.Width(width - 2).Render("\n  " + title.Render("Remove "+m.original.Name+"?") + "\n\n  This removes the saved connection from Hova.\n  Your server and SSH config stay as they are.\n\n  Enter to remove · Esc to keep it\n")
}
