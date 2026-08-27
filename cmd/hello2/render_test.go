package main

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

// Nothing may ever exceed the pane width — the viewport truncates, it does not
// wrap, so an overlong line silently loses text.
func TestRenderFitsWidth(t *testing.T) {
	for _, width := range []int{24, 52, 84} {
		for _, s := range sections {
			for _, line := range strings.Split(render(s.body, width), "\n") {
				if got := lipgloss.Width(line); got > width {
					t.Errorf("%s at width %d: line of %d cols: %q", s.name, width, got, line)
				}
			}
		}
	}
}

// Bullets hang: the first line is marked, continuation lines are indented.
func TestBulletHangs(t *testing.T) {
	body := "- " + strings.Repeat("headline ", 8) + "\n" + strings.Repeat("body ", 8)
	lines := strings.Split(render(body, 40), "\n")
	if len(lines) < 3 {
		t.Fatalf("expected a wrapped bullet, got %d lines", len(lines))
	}
	if !strings.Contains(lines[0], "▸ headline") {
		t.Errorf("first line not marked: %q", lines[0])
	}
	for _, l := range lines[1:] {
		if !strings.HasPrefix(stripANSI(l), "  ") {
			t.Errorf("continuation not indented: %q", l)
		}
	}
}

func TestLinkedBulletMakesHeadlineClickable(t *testing.T) {
	got := render("- [stripeek](https://github.com/progapandist/stripeek)\n  Description.", 52)
	if !strings.Contains(got, "\x1b]8;;https://github.com/progapandist/stripeek\x1b\\") {
		t.Fatalf("linked bullet is missing its OSC 8 target: %q", got)
	}
	if !strings.Contains(got, "stripeek") {
		t.Fatalf("linked bullet is missing its label: %q", got)
	}
}

// content.md drives the menu: one section per "# " heading, in file order.
func TestParseSections(t *testing.T) {
	got := parseSections("# One\n\n## Heading\n\nBody.\n\n# Two\n\nMore.\n")
	if len(got) != 2 {
		t.Fatalf("expected 2 sections, got %d: %#v", len(got), got)
	}
	if got[0].name != "One" || got[0].body != "## Heading\n\nBody." {
		t.Errorf("first section: %#v", got[0])
	}
	if got[1].name != "Two" || got[1].body != "More." {
		t.Errorf("second section: %#v", got[1])
	}
}

// A run of markdown links is a link block, prose that merely mentions one is not.
func TestLinkBlockNeedsEveryLineToBeALink(t *testing.T) {
	if !isLinkBlock("[Email](mailto:andrey@hey.com)\n[Site](https://progapanda.org)") {
		t.Error("a run of links should be a link block")
	}
	if isLinkBlock("See [Site](https://progapanda.org) for more.") {
		t.Error("a paragraph mentioning a link should stay a paragraph")
	}
}

// Source line breaks must not survive into the output.
func TestUnwrap(t *testing.T) {
	if got := unwrap("one\ntwo   three\n"); got != "one two three" {
		t.Errorf("got %q", got)
	}
}

func TestSpreadFitsAndAligns(t *testing.T) {
	got := spread(labelStyle.Render("left"), linkStyle.Render("right"), 20)
	if width := lipgloss.Width(got); width != 20 {
		t.Fatalf("expected 20 columns, got %d", width)
	}
	if !strings.HasSuffix(stripANSI(got), "right") {
		t.Errorf("right detail is not right-aligned: %q", got)
	}

	narrow := spread("a long left label", "right", 10)
	if narrow != "a long left label" {
		t.Errorf("expected narrow layout to omit right detail, got %q", narrow)
	}
}

func TestViewFitsTerminalWidth(t *testing.T) {
	var current tea.Model = model{}
	for _, size := range []tea.WindowSizeMsg{
		{Width: 120, Height: 40},
		{Width: 36, Height: 60}, // iPhone
		{Width: 60, Height: 18},
		{Width: 80, Height: 30},
		{Width: 120, Height: 40},
	} {
		updated, _ := current.Update(size)
		view := updated.(model).View()
		for i, line := range strings.Split(view, "\n") {
			if got := lipgloss.Width(line); got >= size.Width {
				t.Errorf("width %d, line %d: rendered into final column (%d columns)", size.Width, i+1, got)
			}
		}
		current = updated
	}
}

func TestClickingMenuItemSelectsIt(t *testing.T) {
	current, _ := model{reading: true}.Update(tea.WindowSizeMsg{Width: 120, Height: 40})
	updated, _ := current.Update(tea.MouseMsg{
		X:      10,
		Y:      menuItemsY + 3,
		Button: tea.MouseButtonLeft,
		Action: tea.MouseActionPress,
	})
	got := updated.(model)

	if got.cursor != 3 {
		t.Fatalf("expected fourth menu item, got cursor %d", got.cursor)
	}
	if got.reading {
		t.Fatal("expected menu to regain focus after click")
	}
}

func TestMenuClickIgnoresOutsideCoordinates(t *testing.T) {
	for _, point := range []struct{ x, y int }{
		{x: menuWidth, y: menuItemsY},
		{x: 10, y: menuItemsY - 1},
		{x: 10, y: menuItemsY + len(sections)},
	} {
		if index, ok := (model{w: 120}).menuIndexAt(point.x, point.y); ok {
			t.Errorf("menuIndexAt(%d, %d) unexpectedly returned %d", point.x, point.y, index)
		}
	}
}

// Stacked, the menu spans the whole width, so taps past the old 26-column
// menu box must still land on an item.
func TestNarrowMenuIsTappableAcrossFullWidth(t *testing.T) {
	narrow, _ := model{}.Update(tea.WindowSizeMsg{Width: 36, Height: 60})
	m := narrow.(model)
	if !m.narrow() {
		t.Fatal("36 columns should stack the panes")
	}
	index, ok := m.menuIndexAt(30, menuItemsY+2)
	if !ok || index != 2 {
		t.Errorf("tap at x=30 gave (%d, %v), want (2, true)", index, ok)
	}
	if _, ok := m.menuIndexAt(30, menuItemsY+len(sections)); ok {
		t.Error("tap below the last item should miss")
	}
}

// A swipe arrives as a wheel event and must scroll the content even in browse
// mode — tapping a menu item leaves focus on the menu, and a phone shows both.
func TestWheelScrollsContentWithoutFocus(t *testing.T) {
	sized, _ := model{}.Update(tea.WindowSizeMsg{Width: 36, Height: 24})
	m := sized.(model)
	if m.reading {
		t.Fatal("expected to start in browse mode")
	}
	if m.vp.YOffset != 0 {
		t.Fatalf("expected to start at the top, got offset %d", m.vp.YOffset)
	}

	scrolled, _ := m.Update(tea.MouseMsg{
		Button: tea.MouseButtonWheelDown,
		Action: tea.MouseActionPress,
	})
	if got := scrolled.(model).vp.YOffset; got == 0 {
		t.Error("wheel down did not scroll the content pane")
	}
}

// Tab moves focus both ways, and never ends the session — even on Quit, where
// enter does.
func TestTabTogglesFocus(t *testing.T) {
	sized, _ := model{}.Update(tea.WindowSizeMsg{Width: 90, Height: 30})
	m := sized.(model)
	tab := tea.KeyMsg{Type: tea.KeyTab}

	for i, want := range []bool{true, false, true} {
		updated, _ := m.Update(tab)
		m = updated.(model)
		if m.reading != want {
			t.Fatalf("press %d: reading = %v, want %v", i+1, m.reading, want)
		}
	}

	m.cursor = len(sections) - 1 // Quit
	m.reading = false
	updated, cmd := m.Update(tab)
	if cmd != nil {
		t.Error("tab on the Quit entry must not quit")
	}
	if !updated.(model).reading {
		t.Error("tab on the Quit entry should still open it")
	}
}

func TestContentUsesSingleCellCharacters(t *testing.T) {
	for _, section := range sections {
		for _, r := range section.body {
			if width := lipgloss.Width(string(r)); width > 1 {
				t.Errorf("%s contains %q, which occupies %d terminal cells", section.name, r, width)
			}
		}
	}
}

func stripANSI(s string) string {
	for {
		i := strings.Index(s, "\x1b[")
		if i < 0 {
			return s
		}
		j := strings.IndexByte(s[i:], 'm')
		if j < 0 {
			return s
		}
		s = s[:i] + s[i+j+1:]
	}
}
