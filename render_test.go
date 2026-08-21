package main

import (
	"strings"
	"testing"

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

// Source line breaks must not survive into the output.
func TestUnwrap(t *testing.T) {
	if got := unwrap("one\ntwo   three\n"); got != "one two three" {
		t.Errorf("got %q", got)
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
