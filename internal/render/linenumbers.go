package render

import (
	"bytes"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"

	"github.com/glowy/cli/internal/config"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

var ansiRegex = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI removes ANSI escape codes from a string to inspect visible characters.
func StripANSI(s string) string {
	return ansiRegex.ReplaceAllString(s, "")
}

func lineAtOffset(body []byte, offset int) int {
	if offset < 0 {
		return 1
	}
	if offset > len(body) {
		offset = len(body)
	}
	return bytes.Count(body[:offset], []byte("\n")) + 1
}

// nodeByteBounds finds the earliest start byte offset and latest stop byte offset within an AST node subtree.
func nodeByteBounds(n ast.Node) (minStart int, maxStop int, found bool) {
	minStart = math.MaxInt
	maxStop = -1

	record := func(start, stop int) {
		if start < minStart {
			minStart = start
		}
		if stop > maxStop {
			maxStop = stop
		}
		found = true
	}

	_ = ast.Walk(n, func(curr ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if curr.Type() == ast.TypeBlock {
			if fcb, ok := curr.(*ast.FencedCodeBlock); ok && fcb.Info != nil {
				seg := fcb.Info.Segment
				record(seg.Start, seg.Stop)
			}
			lines := curr.Lines()
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				record(seg.Start, seg.Stop)
			}
		}
		if t, ok := curr.(*ast.Text); ok {
			seg := t.Segment
			record(seg.Start, seg.Stop)
		}
		return ast.WalkContinue, nil
	})

	return minStart, maxStop, found
}

func isThematicBreakLine(line string) bool {
	trimmed := strings.TrimSpace(line)
	if len(trimmed) < 3 {
		return false
	}
	ch := trimmed[0]
	if ch != '-' && ch != '*' && ch != '_' {
		return false
	}
	count := 0
	for i := 0; i < len(trimmed); i++ {
		if trimmed[i] == ch {
			count++
		} else if trimmed[i] != ' ' && trimmed[i] != '\t' {
			return false
		}
	}
	return count >= 3
}

// ExtractASTBlockLines extracts 1-indexed source line numbers for rendered blocks.
func ExtractASTBlockLines(body []byte) []int {
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	reader := text.NewReader(body)
	doc := md.Parser().Parse(reader)

	rawLines := strings.Split(string(body), "\n")
	var blockLines []int
	lastLine := 1

	for child := doc.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindThematicBreak:
			foundLine := lastLine
			for idx := lastLine - 1; idx < len(rawLines); idx++ {
				if isThematicBreakLine(rawLines[idx]) {
					foundLine = idx + 1
					break
				}
			}
			blockLines = append(blockLines, foundLine)
			lastLine = foundLine + 1

		case ast.KindFencedCodeBlock, ast.KindCodeBlock:
			fcb, isFenced := child.(*ast.FencedCodeBlock)
			startLine := lastLine
			if isFenced && fcb.Info != nil {
				startLine = lineAtOffset(body, fcb.Info.Segment.Start)
			} else if child.Lines().Len() > 0 {
				firstContentLine := lineAtOffset(body, child.Lines().At(0).Start)
				if isFenced && firstContentLine > 1 {
					startLine = firstContentLine - 1
				} else {
					startLine = firstContentLine
				}
			}

			lines := child.Lines()
			if lines.Len() == 0 {
				blockLines = append(blockLines, startLine)
				lastLine = startLine + 1
				continue
			}

			// Glamour outputs blank lines inside code blocks when the source code block
			// contains empty lines. Track each non-blank sub-chunk inside the code block
			// so ComposeFormattedGutter stays aligned with every block transition.
			inCodeChunk := false
			firstChunk := true
			for i := 0; i < lines.Len(); i++ {
				seg := lines.At(i)
				segText := strings.TrimSpace(string(body[seg.Start:seg.Stop]))
				segLine := lineAtOffset(body, seg.Start)
				if segLine > lastLine {
					lastLine = segLine
				}
				if segText == "" {
					inCodeChunk = false
					continue
				}
				if !inCodeChunk {
					inCodeChunk = true
					if firstChunk {
						blockLines = append(blockLines, startLine)
						firstChunk = false
					} else {
						blockLines = append(blockLines, segLine)
					}
				}
			}
			if firstChunk {
				blockLines = append(blockLines, startLine)
			}
			lastLine++

		default:
			minStart, maxStop, ok := nodeByteBounds(child)
			if ok {
				startLine := lineAtOffset(body, minStart)
				endLine := lineAtOffset(body, maxStop)
				blockLines = append(blockLines, startLine)
				if endLine >= lastLine {
					lastLine = endLine + 1
				}
			} else {
				blockLines = append(blockLines, lastLine)
				lastLine++
			}
		}
	}

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
