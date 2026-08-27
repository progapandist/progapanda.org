package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

const (
	menuWidth    = 26 // total, including border and padding
	minPaneWidth = 20
	headerH      = 2
	menuItemsY   = headerH + 3 // top border, INDEX heading, and rule
)

var (
	accent    = lipgloss.Color("212")
	secondary = lipgloss.Color("81")
	subtle    = lipgloss.Color("241")
	muted     = lipgloss.Color("245")
	bright    = lipgloss.Color("255")
	ink       = lipgloss.Color("235")

	box = lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(subtle).
		Padding(0, 2)
	boxFocused = box.
			Border(lipgloss.ThickBorder()).
			BorderForeground(accent)

	headingStyle = lipgloss.NewStyle().Foreground(accent).Bold(true)
	bulletStyle  = lipgloss.NewStyle().Foreground(bright).Bold(true)
	bodyStyle    = lipgloss.NewStyle().Foreground(bright)
	labelStyle   = lipgloss.NewStyle().Foreground(accent).Bold(true)
	linkStyle    = lipgloss.NewStyle().Foreground(secondary).Underline(true)

	brandStyle    = lipgloss.NewStyle().Foreground(bright).Bold(true)
	kickerStyle   = lipgloss.NewStyle().Foreground(secondary).Bold(true)
	metaStyle     = lipgloss.NewStyle().Foreground(muted)
	itemStyle     = lipgloss.NewStyle().Foreground(muted)
	selectedStyle = lipgloss.NewStyle().
			Foreground(ink).
			Background(accent).
			Bold(true).
			Padding(0, 1)
	footerStyle = lipgloss.NewStyle().Foreground(muted)
	keyStyle    = lipgloss.NewStyle().Foreground(secondary).Bold(true)
	modeStyle   = lipgloss.NewStyle().Foreground(ink).Background(secondary).Bold(true).Padding(0, 1)
)

type model struct {
	cursor  int
	vp      viewport.Model
	w, h    int
	reading bool // focus is on the content pane, not the menu
	ready   bool
}

func (m model) Init() tea.Cmd { return nil }

// narrow reports whether the two panes cannot sit side by side. A phone gives
// us about 36 columns; the menu alone wants 26, so below this the panes stack.
func (m model) narrow() bool {
	return m.w < menuWidth+minPaneWidth+box.GetHorizontalFrameSize()
}

// paneWidth is the usable text width inside the content box.
func (m model) paneWidth() int {
	w := m.w - menuWidth - box.GetHorizontalFrameSize()
	if m.narrow() {
		w = m.w - box.GetHorizontalFrameSize()
	}
	if w < minPaneWidth {
		w = minPaneWidth
	}
	return w
}

// menuHeight is the height of the menu box: the full pane when side by side,
// only as tall as the list itself when stacked, so the content keeps the rest.
func (m model) menuHeight() int {
	if m.narrow() {
		return len(sections) + 2 // the INDEX heading and its rule
	}
	return max(5, m.h-headerH-box.GetVerticalFrameSize()-1)
}

func (m *model) setContent() {
	m.vp.SetContent(render(sections[m.cursor].body, m.paneWidth()))
	m.vp.GotoTop()
}

func (m model) menuIndexAt(x, y int) (int, bool) {
	right := menuWidth - 1
	if m.narrow() {
		right = m.w - 1
	}
	index := y - menuItemsY
	if x <= 0 || x >= right || index < 0 || index >= len(sections) {
		return 0, false
	}
	return index, true
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		// Leave Xterm's final column unused. Writing into that cell puts
		// browser terminals into a wrap-pending state, which can stagger
		// the right border as the renderer advances to the next line.
		m.w, m.h = max(1, msg.Width-1), msg.Height
		// Header (2) + frame (2) + footer (1) + pane title/rule (2).
		vpH := m.h - headerH - box.GetVerticalFrameSize() - 3
		if m.narrow() {
			// Stacked, so the menu box is above rather than beside us.
			vpH -= m.menuHeight() + box.GetVerticalFrameSize()
		}
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
	case tea.MouseMsg:
		if msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress {
			if index, ok := m.menuIndexAt(msg.X, msg.Y); ok && m.ready {
				if sections[index].name == "Quit" {
					return m, tea.Quit
				}
				m.cursor = index
				m.reading = false
				m.setContent()
				return m, nil
			}
		}
	}

	// A wheel event scrolls the content pane whether or not it has focus. The
	// frontend turns a touch swipe into these, and on a phone both panes are on
	// screen at once, so requiring focus first would leave a tapped section
	// unscrollable. The menu is short enough never to need scrolling itself.
	if mouse, ok := msg.(tea.MouseMsg); ok {
		switch mouse.Button {
		case tea.MouseButtonWheelUp, tea.MouseButtonWheelDown:
			var cmd tea.Cmd
			m.vp, cmd = m.vp.Update(msg)
			return m, cmd
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
		number := fmt.Sprintf("%02d", i+1)
		if i == m.cursor {
			label := fit(number+"  "+s.name, menuWidth-box.GetHorizontalFrameSize()-2)
			items = append(items, selectedStyle.Render(label))
			continue
		}
		items = append(items, itemStyle.Render(metaStyle.Render(number)+"  "+s.name))
	}
	menuH := m.menuHeight()
	menuContent := lipgloss.JoinVertical(lipgloss.Left,
		kickerStyle.Render("INDEX"),
		metaStyle.Render("──────────────"),
		strings.Join(items, "\n"),
	)
	menuInnerWidth := menuWidth - box.GetHorizontalFrameSize()
	if m.narrow() {
		menuInnerWidth = m.paneWidth()
	}
	menu := lipgloss.NewStyle().Width(menuInnerWidth).
		Height(menuH).Render(menuContent)

	menuBox, contentBox := boxFocused, box
	if m.reading {
		menuBox, contentBox = box, boxFocused
	}

	position := fmt.Sprintf("%02d / %02d", m.cursor+1, len(sections))
	if m.reading {
		position = fmt.Sprintf("%3.0f%%", m.vp.ScrollPercent()*100)
	}
	title := spread(
		headingStyle.Render(strings.ToUpper(sections[m.cursor].name)),
		metaStyle.Render(position),
		m.paneWidth(),
	)
	content := lipgloss.JoinVertical(lipgloss.Left,
		title,
		lipgloss.NewStyle().Foreground(subtle).Render(strings.Repeat("━", m.paneWidth())),
		m.vp.View(),
	)
	panes := lipgloss.JoinHorizontal(lipgloss.Top,
		menuBox.Height(menuH).Render(menu),
		contentBox.Height(menuH).Render(content),
	)
	if m.narrow() {
		panes = lipgloss.JoinVertical(lipgloss.Left,
			menuBox.Height(menuH).Render(menu),
			contentBox.Height(m.vp.Height+2).Render(content),
		)
	}
	return m.header() + "\n" + panes + "\n" + m.footer()
}

func (m model) header() string {
	left := brandStyle.Render("◆ PROGAPANDA") + metaStyle.Render("  /  PORTFOLIO.TUI")
	right := kickerStyle.Render("● ONLINE")
	rule := lipgloss.NewStyle().Foreground(subtle).Render(strings.Repeat("─", max(0, m.w)))
	return spread(left, right, m.w) + "\n" + rule
}

func (m model) footer() string {
	k := func(key, what string) string {
		return keyStyle.Render(key) + footerStyle.Render(" "+what)
	}
	mode := modeStyle.Render(" BROWSE ")
	keys := []string{k("↑/↓", "navigate"), k("enter", "open"), k("q", "quit")}
	if m.reading {
		mode = modeStyle.Render(" READING ")
		keys = []string{k("↑/↓", "scroll"), k("esc", "index"), k("q", "quit")}
	}
	hints := strings.Join(keys, footerStyle.Render("   "))
	return spread(mode, hints, m.w)
}

// spread places left and right at opposite edges of a line, accounting for
// ANSI styling. If the terminal is too narrow, the right-hand detail is hidden.
func spread(left, right string, width int) string {
	gap := width - lipgloss.Width(left) - lipgloss.Width(right)
	if gap < 1 {
		return left
	}
	return left + strings.Repeat(" ", gap) + right
}

// fit pads a menu label so the selected background forms a consistent pill.
func fit(s string, width int) string {
	if gap := width - lipgloss.Width(s); gap > 0 {
		return s + strings.Repeat(" ", gap)
	}
	return s
}

func main() {
	p := tea.NewProgram(model{}, tea.WithAltScreen(), tea.WithMouseCellMotion())
	if _, err := p.Run(); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}
