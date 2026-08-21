package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const menuWidth = 22 // total, including border and padding

var (
	accent = lipgloss.Color("212")
	subtle = lipgloss.Color("244")
	bright = lipgloss.Color("252")

	box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 2)
	boxFocused = box.BorderForeground(accent)

	headingStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	bulletStyle  = lipgloss.NewStyle().Foreground(bright).Bold(true)
	bodyStyle    = lipgloss.NewStyle().Foreground(bright)
	labelStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	linkStyle    = lipgloss.NewStyle().Foreground(subtle).Underline(true)

	itemStyle     = lipgloss.NewStyle().Foreground(subtle).PaddingLeft(2)
	selectedStyle = lipgloss.NewStyle().Foreground(accent).Bold(true).PaddingLeft(0).SetString("▸ ")
	footerStyle   = lipgloss.NewStyle().Foreground(subtle)
	keyStyle      = lipgloss.NewStyle().Foreground(accent)
)

type model struct {
	cursor  int
	vp      viewport.Model
	w, h    int
	reading bool // focus is on the content pane, not the menu
	ready   bool
}

func (m model) Init() tea.Cmd { return nil }

// paneWidth is the usable text width inside the content box.
func (m model) paneWidth() int {
	w := m.w - menuWidth - box.GetHorizontalFrameSize()
	if w < 20 {
		w = 20
	}
	return w
}

func (m *model) setContent() {
	m.vp.SetContent(render(sections[m.cursor].body, m.paneWidth()))
	m.vp.GotoTop()
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.w, m.h = msg.Width, msg.Height
		// frame (2) + footer (1) + pane title (1) + rule (1)
		vpH := m.h - box.GetVerticalFrameSize() - 3
		if vpH < 3 {
			vpH = 3
		}
		if !m.ready {
			m.vp = viewport.New(m.paneWidth(), vpH)
			m.ready = true
		} else {
			m.vp.Width, m.vp.Height = m.paneWidth(), vpH
		}
		m.setContent()
		return m, nil

	case tea.KeyMsg:
		switch msg.String() {
		case "ctrl+c", "q":
			return m, tea.Quit
		case "esc":
			m.reading = false
			return m, nil
		case "enter":
			if sections[m.cursor].name == "Quit" {
				return m, tea.Quit
			}
			m.reading = true
			return m, nil
		case "up", "k":
			if m.reading {
				break
			}
			m.cursor = (m.cursor - 1 + len(sections)) % len(sections)
			m.setContent()
			return m, nil
		case "down", "j":
			if m.reading {
				break
			}
			m.cursor = (m.cursor + 1) % len(sections)
			m.setContent()
			return m, nil
		}
	}

	if !m.reading {
		return m, nil
	}
	var cmd tea.Cmd
	m.vp, cmd = m.vp.Update(msg)
	return m, cmd
}

func (m model) View() string {
	if !m.ready {
		return "\n  Starting up..."
	}

	var items []string
	for i, s := range sections {
		if i == m.cursor {
			items = append(items, selectedStyle.Render(s.name))
			continue
		}
		items = append(items, itemStyle.Render(s.name))
	}
	menuH := m.h - box.GetVerticalFrameSize() - 1
	menu := lipgloss.NewStyle().Width(menuWidth - box.GetHorizontalFrameSize()).
		Height(menuH).Render(strings.Join(items, "\n"))

	menuBox, contentBox := boxFocused, box
	if m.reading {
		menuBox, contentBox = box, boxFocused
	}

	content := lipgloss.JoinVertical(lipgloss.Left,
		headingStyle.Render(sections[m.cursor].name),
		lipgloss.NewStyle().Foreground(subtle).Render(strings.Repeat("─", m.paneWidth())),
		m.vp.View(),
	)
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		menuBox.Height(menuH).Render(menu),
		contentBox.Height(menuH).Render(content),
	)
	return panes + "\n" + m.footer()
}

func (m model) footer() string {
	k := func(key, what string) string {
		return keyStyle.Render(key) + footerStyle.Render(" "+what)
	}
	keys := []string{k("↑/↓", "navigate"), k("enter", "read"), k("esc", "menu"), k("q", "quit")}
	if m.reading {
		keys = []string{k("↑/↓", "scroll"), k("esc", "back to menu"), k("q", "quit")}
	}
	return footerStyle.Render(" Andy Baranov · ") + strings.Join(keys, footerStyle.Render(" · "))
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
