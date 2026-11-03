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

// HighlightMatchesInLine highlights all case-insensitive occurrences of query in a single ANSI line.
func HighlightMatchesInLine(line string, query string, theme *config.UITheme) string {
	q := strings.TrimSpace(query)
	if q == "" {
		return line
	}

	// Find matches in plain text
	plain := ansiPattern.ReplaceAllString(line, "")
	lowerPlain := strings.ToLower(plain)
	lowerQ := strings.ToLower(q)

	if !strings.Contains(lowerPlain, lowerQ) {
		return line
	}

	// Map each byte in line to plain text position, or identify ANSI sequences
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

	// For non-ANSI tokens, find and replace matches
	// Track the active ANSI formatting so surrounding text retains its color after match reset
	var result strings.Builder
	currentANSI := ""

	for _, tok := range tokens {
		if tok.isANSI {
			if tok.text == "\x1b[0m" {
				currentANSI = ""
			} else {
				currentANSI = tok.text
			}
			result.WriteString(tok.text)
			continue
		}

		tokLower := strings.ToLower(tok.text)
		tokIdx := 0
		for {
			m := strings.Index(tokLower[tokIdx:], lowerQ)
			if m == -1 {
				result.WriteString(tok.text[tokIdx:])
				break
			}

			startMatch := tokIdx + m
			endMatch := startMatch + len(lowerQ)

			result.WriteString(tok.text[tokIdx:startMatch])
			matched := tok.text[startMatch:endMatch]
			result.WriteString(theme.SearchHighlight.Render(matched))
			if currentANSI != "" {
				result.WriteString(currentANSI)
			}

			tokIdx = endMatch
		}
	}

	return result.String()
}
