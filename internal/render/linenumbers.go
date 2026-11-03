package render

import (
	"bytes"
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/glowy/cli/internal/config"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape codes from a string to inspect visible characters.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

// ExtractASTBlockLines extracts 1-indexed source line numbers for all top-level AST blocks.
func ExtractASTBlockLines(body []byte) []int {
	md := goldmark.New()
	reader := text.NewReader(body)
	doc := md.Parser().Parse(reader)

	var blockLines []int
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if entering && n.Type() == ast.TypeBlock && n != doc {
			// Only top-level blocks directly under Document, or list items
			if n.Parent() == doc || n.Kind() == ast.KindListItem {
				lines := n.Lines()
				if lines.Len() > 0 {
					seg := lines.At(0)
					lineNum := bytes.Count(body[:seg.Start], []byte("\n")) + 1
					blockLines = append(blockLines, lineNum)
				}
			}
		}
		return ast.WalkContinue, nil
	})
	return blockLines
}

// ComposeRawGutter prepends right-aligned 1:1 raw line numbers to raw lines.
func ComposeRawGutter(lines []string, theme *config.UITheme) []string {
	total := len(lines)
	if total == 0 {
		return lines
	}
	width := len(strconv.Itoa(total))
	if width < 2 {
		width = 2
	}

	result := make([]string, total)
	for i, line := range lines {
		gutter := theme.Gutter.Render(fmt.Sprintf("%*d │ ", width, i+1))
		result[i] = gutter + line
	}
	return result
}

// ComposeFormattedGutter attaches raw line numbers to the start of rendered Markdown blocks.
func ComposeFormattedGutter(renderedLines []string, frontmatterLines int, fmRenderedCount int, astBlockLines []int, totalRawLines int, theme *config.UITheme) []string {
	if totalRawLines < 1 {
		totalRawLines = len(renderedLines)
	}
	width := len(strconv.Itoa(totalRawLines))
	if width < 2 {
		width = 2
	}

	blankGutter := theme.Gutter.Render(fmt.Sprintf("%*s │ ", width, ""))
	result := make([]string, len(renderedLines))

	astIdx := 0
	inBlock := false

	for i, line := range renderedLines {
		// Frontmatter block handling
		if i < fmRenderedCount {
			if i == 0 {
				result[i] = theme.Gutter.Render(fmt.Sprintf("%*d │ ", width, 1)) + line
			} else {
				result[i] = blankGutter + line
			}
			continue
		}

		visible := strings.TrimSpace(StripANSI(line))
		if visible == "" {
			// Empty line between blocks
			inBlock = false
			result[i] = blankGutter + line
			continue
		}

		if !inBlock {
			// Start of a new rendered block
			inBlock = true
			if astIdx < len(astBlockLines) {
				rawLine := astBlockLines[astIdx] + frontmatterLines
				result[i] = theme.Gutter.Render(fmt.Sprintf("%*d │ ", width, rawLine)) + line
				astIdx++
			} else {
				result[i] = blankGutter + line
			}
		} else {
			// Continuation line within the current block
			result[i] = blankGutter + line
		}
	}

	return result
}
