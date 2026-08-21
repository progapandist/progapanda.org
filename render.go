package main

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// render turns the tiny markup in content.go into styled text wrapped to width.
// Blocks are separated by blank lines; the first two characters of a block pick
// the style. Source line breaks are ignored — text is re-wrapped to the real
// terminal width.
func render(body string, width int) string {
	if width < 20 {
		width = 20
	}

	var out []string
	for _, block := range strings.Split(body, "\n\n") {
		if strings.TrimSpace(block) == "" {
			continue
		}
		switch {
		case strings.HasPrefix(block, "# "):
			out = append(out, wrapStyled(block[2:], width, headingStyle, "", ""))
		case strings.HasPrefix(block, "- "):
			out = append(out, bulletBlock(block[2:], width))
		case strings.HasPrefix(block, "@ "):
			out = append(out, linkBlock(block, width))
		default:
			out = append(out, wrapStyled(block, width, bodyStyle, "", ""))
		}
	}
	return strings.Join(out, "\n\n")
}

// bulletBlock renders "▸ headline" with any following text hanging under it.
func bulletBlock(block string, width int) string {
	head, rest, _ := strings.Cut(block, "\n")
	s := wrapStyled(head, width, bulletStyle, "▸ ", "  ")
	if strings.TrimSpace(rest) != "" {
		s += "\n" + wrapStyled(rest, width, bodyStyle, "  ", "  ")
	}
	return s
}

// linkBlock keeps label and URL on one line when they fit, and drops the URL
// to its own (hard-wrapped) line when they don't. A URL must never be
// truncated — the viewport clips rather than wraps, and half a URL is useless.
func linkBlock(block string, width int) string {
	var links []string
	for _, line := range strings.Split(block, "\n") {
		label, url, found := strings.Cut(strings.TrimPrefix(line, "@ "), "|")
		if !found {
			continue
		}
		if lipgloss.Width(label+"  "+url) <= width {
			links = append(links, labelStyle.Render(label)+"  "+linkStyle.Render(url))
			continue
		}
		links = append(links, labelStyle.Render(label))
		for _, l := range strings.Split(lipgloss.NewStyle().Width(width).Render(url), "\n") {
			links = append(links, linkStyle.Render(strings.TrimRight(l, " ")))
		}
	}
	return strings.Join(links, "\n")
}

// wrapStyled re-wraps text to width and prefixes the first line with first and
// every later line with rest, which is what makes bullets hang.
func wrapStyled(text string, width int, style lipgloss.Style, first, rest string) string {
	wrapped := lipgloss.NewStyle().Width(width - len(rest)).Render(unwrap(text))
	var lines []string
	for i, l := range strings.Split(wrapped, "\n") {
		prefix := rest
		if i == 0 {
			prefix = first
		}
		lines = append(lines, style.Render(prefix+strings.TrimRight(l, " ")))
	}
	return strings.Join(lines, "\n")
}

// unwrap joins hard-wrapped source lines into one paragraph.
func unwrap(s string) string { return strings.Join(strings.Fields(s), " ") }
