package ui

import (
	"regexp"
	"strings"

	"github.com/glowy/cli/internal/config"
)

var ansiPattern = regexp.MustCompile(`\x1b\[[0-9;]*[a-zA-Z]`)

// StripANSI strips ANSI color and style escape sequences from a string.
func StripANSI(s string) string {
	return ansiPattern.ReplaceAllString(s, "")
}

type SearchMatch struct {
	LineIndex int
	StartCol  int
	EndCol    int
}

func FindMatches(lines []string, query string) []SearchMatch {
	q := strings.TrimSpace(query)
	if q == "" {
		return nil
	}
	lowerQ := strings.ToLower(q)

	var matches []SearchMatch
	for lineIdx, line := range lines {
		plain := ansiPattern.ReplaceAllString(line, "")
		lowerPlain := strings.ToLower(plain)

		start := 0
		for {
			idx := strings.Index(lowerPlain[start:], lowerQ)
			if idx == -1 {
				break
			}
			matchStart := start + idx
			matchEnd := matchStart + len(lowerQ)
			matches = append(matches, SearchMatch{
				LineIndex: lineIdx,
				StartCol:  matchStart,
				EndCol:    matchEnd,
			})
			start = matchEnd
		}
	}
	return matches
}

// HighlightMatchesInLine highlights all case-insensitive occurrences of query in a single ANSI line,
// including matches that span across multiple ANSI style boundaries.
func HighlightMatchesInLine(line string, query string, theme *config.UITheme) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return line
	}

	plain := ansiPattern.ReplaceAllString(line, "")
	lowerPlain := strings.ToLower(plain)
	lowerQ := strings.ToLower(q)

	if len(lowerPlain) != len(plain) {
		// Fallback if Unicode case-folding changes byte length
		if !strings.Contains(lowerPlain, lowerQ) {
			return line
		}
	}

	var spans [][2]int
	start := 0
	for {
		idx := strings.Index(lowerPlain[start:], lowerQ)
		if idx == -1 {
			break
		}
		mStart := start + idx
		mEnd := mStart + len(lowerQ)
		spans = append(spans, [2]int{mStart, mEnd})
		start = mEnd
	}
	if len(spans) == 0 {
		return line
	}

	type token struct {
		isANSI bool
		text   string
	}

	var tokens []token
	locs := ansiPattern.FindAllStringIndex(line, -1)
	last := 0
	for _, loc := range locs {
		if loc[0] > last {
			tokens = append(tokens, token{isANSI: false, text: line[last:loc[0]]})
		}
		tokens = append(tokens, token{isANSI: true, text: line[loc[0]:loc[1]]})
		last = loc[1]
	}
	if last < len(line) {
		tokens = append(tokens, token{isANSI: false, text: line[last:]})
	}

	var result strings.Builder
	var currentANSI strings.Builder
	plainPos := 0
	spanIdx := 0

	for _, tok := range tokens {
		if tok.isANSI {
			if tok.text == "\x1b[0m" || tok.text == "\x1b[m" {
				currentANSI.Reset()
			} else {
				currentANSI.WriteString(tok.text)
			}
			result.WriteString(tok.text)
			continue
		}

		tokLen := len(tok.text)
		tokStartPlain := plainPos
		tokEndPlain := plainPos + tokLen
		plainPos = tokEndPlain

		if len(tok.text) == 0 {
			continue
		}

		localIdx := 0
		for localIdx < tokLen {
			for spanIdx < len(spans) && spans[spanIdx][1] <= tokStartPlain+localIdx {
				spanIdx++
			}
			if spanIdx >= len(spans) {
				result.WriteString(tok.text[localIdx:])
				break
			}

			mStart := spans[spanIdx][0]
			mEnd := spans[spanIdx][1]

			if mStart >= tokEndPlain {
				result.WriteString(tok.text[localIdx:])
				break
			}

			// Emit unhighlighted prefix within this token before match starts
			if mStart > tokStartPlain+localIdx {
				prefixEnd := mStart - tokStartPlain
				result.WriteString(tok.text[localIdx:prefixEnd])
				localIdx = prefixEnd
			}

			// Emit highlighted portion within this token
			hlEndPlain := mEnd
			if hlEndPlain > tokEndPlain {
				hlEndPlain = tokEndPlain
			}
			hlEndLocal := hlEndPlain - tokStartPlain
			if hlEndLocal > localIdx {
				matched := tok.text[localIdx:hlEndLocal]
				result.WriteString(theme.SearchHighlight.Render(matched))
				if currentANSI.Len() > 0 {
					result.WriteString(currentANSI.String())
				}
				localIdx = hlEndLocal
			}
		}
	}

	return result.String()
}
