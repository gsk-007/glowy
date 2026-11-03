package render

import (
	"strings"
	"testing"

	"github.com/glowy/cli/internal/config"
)

func TestExtractFrontmatter(t *testing.T) {
	doc := `---
title: Test Document
author: Tester
tags: [go, tui]
---

# Real Title
Body paragraph.
`
	fm, body := ExtractFrontmatter(doc)
	if fm == nil {
		t.Fatal("expected frontmatter to be detected")
	}
	if fm.RawLines != 5 {
		t.Fatalf("expected 5 raw lines, got %d", fm.RawLines)
	}
	if len(fm.Fields) != 3 {
		t.Fatalf("expected 3 fields, got %d", len(fm.Fields))
	}
	if fm.Fields[0][0] != "title" || fm.Fields[0][1] != "Test Document" {
		t.Fatalf("unexpected title field: %+v", fm.Fields[0])
	}

	if !strings.HasPrefix(strings.TrimSpace(body), "# Real Title") {
		t.Fatalf("body should start with '# Real Title', got: %q", body)
	}
}

func TestExtractFrontmatterCRLF(t *testing.T) {
	// Exact 6-line YAML frontmatter with Windows CRLF
	docCRLF := "---\r\ntitle: Windows Doc\r\nauthor: Dev\r\ndate: 2026-11-03\r\nversion: 1.0\r\n---\r\n# Heading 1\r\nContent line.\r\n"
	fm, body := ExtractFrontmatter(docCRLF)
	if fm == nil {
		t.Fatal("expected frontmatter to be detected with CRLF")
	}
	if fm.RawLines != 6 {
		t.Fatalf("expected 6 raw lines for frontmatter, got %d", fm.RawLines)
	}
	if len(fm.Fields) != 4 {
		t.Fatalf("expected 4 fields, got %d", len(fm.Fields))
	}
	if !strings.HasPrefix(strings.TrimSpace(body), "# Heading 1") {
		t.Fatalf("body should start with '# Heading 1', got: %q", body)
	}

	// Verify line number mapping: Heading 1 must be raw line 7
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))
	res, err := Render(docCRLF, RenderOptions{
		Mode:        "formatted",
		LineNumbers: true,
		Width:       80,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}

	// Look for line 7 in the gutter
	foundLine7 := false
	for _, l := range res.Lines {
		plain := StripANSI(l)
		if strings.Contains(plain, "7 │") && strings.Contains(plain, "Heading 1") {
			foundLine7 = true
			break
		}
	}
	if !foundLine7 {
		t.Fatalf("expected line 7 in gutter for 'Heading 1', output lines:\n%s", res.Content)
	}
}

func TestExtractFrontmatterNone(t *testing.T) {
	doc := "# Just Markdown\nNo frontmatter here.\n"
	fm, body := ExtractFrontmatter(doc)
	if fm != nil {
		t.Fatal("expected no frontmatter")
	}
	if body != doc {
		t.Fatal("body should equal original doc")
	}
}

func TestRenderFormattedAndRaw(t *testing.T) {
	doc := `---
title: Hello Glowy
---

# Section 1
This is a test paragraph with ` + "`code`" + `.

` + "```go\nfunc add(a, b int) int { return a + b }\n```" + `
`
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))

	// Test Formatted mode without line numbers
	resFmt, err := Render(doc, RenderOptions{
		Mode:        "formatted",
		LineNumbers: false,
		Width:       80,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("formatted render failed: %v", err)
	}
	if len(resFmt.Lines) == 0 {
		t.Fatal("formatted output lines should not be empty")
	}
	if !strings.Contains(resFmt.Content, "title:") || !strings.Contains(resFmt.Content, "Hello Glowy") {
		t.Fatalf("rendered output missing frontmatter: %s", resFmt.Content)
	}

	// Test Formatted mode with line numbers
	resFmtLN, err := Render(doc, RenderOptions{
		Mode:        "formatted",
		LineNumbers: true,
		Width:       80,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("formatted render with line numbers failed: %v", err)
	}
	if !strings.Contains(resFmtLN.Content, "│") {
		t.Fatalf("expected gutter character '│' in line numbers output: %s", resFmtLN.Content)
	}

	// Test Raw mode without line numbers
	resRaw, err := Render(doc, RenderOptions{
		Mode:        "raw",
		LineNumbers: false,
		Width:       80,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("raw render failed: %v", err)
	}
	if len(resRaw.Lines) == 0 {
		t.Fatal("raw output lines should not be empty")
	}

	// Test Raw mode with line numbers
	resRawLN, err := Render(doc, RenderOptions{
		Mode:        "raw",
		LineNumbers: true,
		Width:       80,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("raw render with line numbers failed: %v", err)
	}
	if !strings.Contains(resRawLN.Content, " 1 │") {
		t.Fatalf("expected raw line 1 in gutter: %s", resRawLN.Content)
	}
}

func TestFullCanvasBackgroundPainting(t *testing.T) {
	doc := "# Heading 1\nShort line.\n\nAnother paragraph.\n"
	cfg := config.DefaultConfig()
	paperLight := config.NewUITheme(cfg.GetScheme("PaperLight"))
	expectedBgSeq := "\x1b[48;2;250;250;250m"

	// 1. Formatted mode full canvas check
	resFmt, err := Render(doc, RenderOptions{
		Mode:        "formatted",
		LineNumbers: true,
		Width:       80,
		Theme:       paperLight,
	})
	if err != nil {
		t.Fatalf("render formatted failed: %v", err)
	}
	for i, line := range resFmt.Lines {
		if !strings.HasPrefix(line, expectedBgSeq) {
			t.Fatalf("line %d does not start with PaperLight bgSeq: %q", i, line)
		}
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Fatalf("line %d does not end with \\x1b[0m: %q", i, line)
		}
	}

	// 2. Raw mode full canvas check
	resRaw, err := Render(doc, RenderOptions{
		Mode:        "raw",
		LineNumbers: true,
		Width:       80,
		Theme:       paperLight,
	})
	if err != nil {
		t.Fatalf("render raw failed: %v", err)
	}
	for i, line := range resRaw.Lines {
		if !strings.HasPrefix(line, expectedBgSeq) {
			t.Fatalf("raw line %d does not start with PaperLight bgSeq: %q", i, line)
		}
		if !strings.HasSuffix(line, "\x1b[0m") {
			t.Fatalf("raw line %d does not end with \\x1b[0m: %q", i, line)
		}
	}
}
