// Preview regenerates the README terminal illustration from Hova's real view.
package main

import (
	"fmt"
	"html"
	"os"
	"regexp"
	"strconv"
	"strings"

	"github.com/alzubair-alojali/hova/internal/inventory"
	"github.com/alzubair-alojali/hova/internal/tui"
	"github.com/charmbracelet/lipgloss"
	"github.com/muesli/termenv"
)

var sgr = regexp.MustCompile(`\x1b\[([0-9;]*)m`)

type style struct {
	foreground, background string
	bold                   bool
}

func main() {
	lipgloss.SetColorProfile(termenv.TrueColor)
	model := tui.New(inventory.Store{}, []inventory.Server{
		{Name: "production", Host: "203.0.113.10", User: "deploy", Port: 22, IdentityFile: "~/.ssh/work_ed25519", Tags: []string{"work", "production"}, Notes: "The main application server"},
		{Name: "staging", Host: "203.0.113.20", User: "ubuntu", Port: 2222, Tags: []string{"work", "preview"}},
		{Name: "personal", Host: "198.51.100.30", User: "root", Port: 22, Tags: []string{"side projects"}},
		{Name: "database", Host: "db.internal", User: "postgres", Port: 22, Tags: []string{"work", "private network"}},
	})
	var svg strings.Builder
	svg.WriteString(`<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" viewBox="0 0 1200 800" role="img" aria-label="Hova terminal workspace with illustrative server data"><rect width="1200" height="800" rx="16" fill="#0c1712"/><path d="M0 48h1200" stroke="#294133"/><circle cx="27" cy="25" r="5" fill="#b47172"/><circle cx="47" cy="25" r="5" fill="#b4a172"/><circle cx="67" cy="25" r="5" fill="#8be9ba"/><text x="600" y="30" text-anchor="middle" font-family="monospace" font-size="12" fill="#84968f">hova — your SSH workspace</text>`)
	for row, line := range strings.Split(model.View(), "\n") {
		renderLine(&svg, line, 80+row*22)
	}
	svg.WriteString(`</svg>`)
	if err := os.WriteFile("assets/terminal.svg", []byte(svg.String()), 0644); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func renderLine(svg *strings.Builder, line string, y int) {
	current := style{foreground: "#e3eae8"}
	position, x := 0, 24
	for _, loc := range sgr.FindAllStringSubmatchIndex(line, -1) {
		x = renderText(svg, line[position:loc[0]], x, y, current)
		current = applyStyle(current, line[loc[2]:loc[3]])
		position = loc[1]
	}
	renderText(svg, line[position:], x, y, current)
}

func renderText(svg *strings.Builder, text string, x, y int, s style) int {
	width := lipgloss.Width(text) * 11
	if text == "" {
		return x
	}
	if s.background != "" {
		fmt.Fprintf(svg, `<rect x="%d" y="%d" width="%d" height="22" fill="%s"/>`, x, y-16, width, s.background)
	}
	weight := "400"
	if s.bold {
		weight = "700"
	}
	fmt.Fprintf(svg, `<text x="%d" y="%d" textLength="%d" lengthAdjust="spacingAndGlyphs" xml:space="preserve" font-family="Cascadia Code,Consolas,monospace" font-size="18" font-weight="%s" fill="%s">%s</text>`, x, y, width, weight, s.foreground, html.EscapeString(text))
	return x + width
}

func applyStyle(s style, code string) style {
	parts := strings.Split(code, ";")
	for i := 0; i < len(parts); i++ {
		switch parts[i] {
		case "", "0":
			s = style{foreground: "#e3eae8"}
		case "1":
			s.bold = true
		case "22":
			s.bold = false
		case "39":
			s.foreground = "#e3eae8"
		case "49":
			s.background = ""
		case "38", "48":
			if i+4 < len(parts) && parts[i+1] == "2" {
				r, _ := strconv.Atoi(parts[i+2])
				g, _ := strconv.Atoi(parts[i+3])
				b, _ := strconv.Atoi(parts[i+4])
				color := fmt.Sprintf("#%02x%02x%02x", r, g, b)
				if parts[i] == "38" {
					s.foreground = color
				} else {
					s.background = color
				}
				i += 4
			}
		}
	}
	return s
}
