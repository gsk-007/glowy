package render

import (
	"os"
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
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

func TestLineWidthsNeverExceedTermWidth(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))

	for _, mode := range []string{"formatted", "raw"} {
		for _, ln := range []bool{false, true} {
			for width := 40; width <= 120; width += 20 {
				res, err := Render(string(data), RenderOptions{
					Mode:        mode,
					LineNumbers: ln,
					Width:       width,
					Theme:       theme,
				})
				if err != nil {
					t.Fatalf("mode=%s ln=%v width=%d failed: %v", mode, ln, width, err)
				}
				for i, l := range res.Lines {
					w := lipgloss.Width(l)
					if w > width {
						t.Errorf("mode=%s ln=%v width=%d line %d width %d > %d: %q", mode, ln, width, i, w, width, l)
					}
				}
			}
		}
	}
}

func TestRenderPRD(t *testing.T) {
	data, err := os.ReadFile("../../PRD.md")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))

	for _, mode := range []string{"formatted", "raw"} {
		for _, ln := range []bool{false, true} {
			for _, width := range []int{60, 80, 100, 120} {
				res, err := Render(string(data), RenderOptions{
					Mode:        mode,
					LineNumbers: ln,
					Width:       width,
					Theme:       theme,
				})
				if err != nil {
					t.Fatalf("mode=%s ln=%v width=%d failed: %v", mode, ln, width, err)
				}
				for i, l := range res.Lines {
					w := lipgloss.Width(l)
					if w > width {
						t.Errorf("PRD mode=%s ln=%v width=%d line %d width %d > %d: %q", mode, ln, width, i, w, width, l)
					}
				}
			}
		}
	}
}

func TestTableWrappingPreservesAllContent(t *testing.T) {
	data, err := os.ReadFile("../../README.md")
	if err != nil {
		t.Fatal(err)
	}
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))

	res, err := Render(string(data), RenderOptions{
		Mode:        "formatted",
		LineNumbers: true,
		Width:       80,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("render README failed: %v", err)
	}

	plain := StripANSI(res.Content)

	// Verify wrapped table cells on the right column are completely preserved
	expectedPhrases := []string{
		"interactive TUI",
		"pager.",
		"formatted output.",
		"filenames and directory paths",
		"Created ↓",
		"Syntax-Highlighted Raw Markdown",
		"231 │",
	}
	for _, phrase := range expectedPhrases {
		if !strings.Contains(plain, phrase) {
			t.Errorf("expected rendered README to contain wrapped phrase %q, but it was missing", phrase)
		}
	}
}

func TestLongFrontmatterValueWrapsWithIntactBorders(t *testing.T) {
	doc := "---\ntitle: Very Long Frontmatter Value That Should Wrap Cleanly Inside The Rounded Box Without Clipping Borders\n---\n\n# Body\n"
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))

	res, err := Render(doc, RenderOptions{
		Mode:        "formatted",
		LineNumbers: false,
		Width:       50,
		Theme:       theme,
	})
	if err != nil {
		t.Fatalf("render failed: %v", err)
	}
	plain := StripANSI(res.Content)
	if !strings.Contains(plain, "Without") || !strings.Contains(plain, "Borders") {
		t.Fatalf("expected long frontmatter value to wrap without losing trailing words, got:\n%s", plain)
	}
	for i, l := range res.Lines {
		if lipgloss.Width(l) > 50 {
			t.Errorf("line %d width %d > 50: %q", i, lipgloss.Width(l), l)
		}
	}
}

