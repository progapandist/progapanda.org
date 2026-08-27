package main

import (
	_ "embed"
	"strings"
)

// All the prose lives in content.md so the copy can be edited without touching
// Go. It is embedded, so the binary stays a single self-contained file.
//
//go:embed content.md
var contentMarkdown string

type section struct {
	name string
	body string
}

var sections = parseSections(contentMarkdown)

// parseSections splits the markdown on "# " headings: each one starts a new
// section, its text becomes the menu label, and everything below it is the
// body. "## " headings stay in the body — see render().
func parseSections(md string) []section {
	var out []section
	for _, chunk := range strings.Split("\n"+md, "\n# ")[1:] {
		name, body, _ := strings.Cut(chunk, "\n")
		out = append(out, section{strings.TrimSpace(name), strings.TrimSpace(body)})
	}
	return out
}
