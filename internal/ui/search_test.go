package ui

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/glowy/cli/internal/config"
	"github.com/muesli/termenv"
)

func TestSearchHighlight(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	theme := config.NewUITheme(config.DefaultSlateDark())
	line := "Hello \x1b[38;2;56;189;248mWorld\x1b[0m and Welcome!"
	query := "world"

	matches := FindMatches([]string{line}, query)
	if len(matches) != 1 {
		t.Fatalf("expected 1 match, got %d", len(matches))
	}

	highlighted := HighlightMatchesInLine(line, query, theme)
	t.Logf("Original:    %q", line)
	t.Logf("Highlighted: %q", highlighted)
	if !strings.Contains(highlighted, "World") || highlighted == line {
		t.Fatalf("expected highlight to style World differently from original")
	}
}

func TestSearchHighlightAnsiEscapeSafety(t *testing.T) {
	lipgloss.SetColorProfile(termenv.TrueColor)
	theme := config.NewUITheme(config.DefaultSlateDark())

	// Line containing ANSI codes where "38", "2", "m" occur inside the escape sequence
	line := "\x1b[38;2;56;189;248mThis is blue text\x1b[0m"

	// Searching for "38" should NOT match the ANSI sequence
	matches38 := FindMatches([]string{line}, "38")
	if len(matches38) != 0 {
		t.Fatalf("expected 0 matches for '38' inside ANSI code, got %d", len(matches38))
	}
	highlighted38 := HighlightMatchesInLine(line, "38", theme)
	if highlighted38 != line {
		t.Fatalf("expected line unchanged when matching '38', got %q", highlighted38)
	}

	// Searching for "m" should only match 'm' in visible text (none in "This is blue text")
	matchesM := FindMatches([]string{line}, "m")
	if len(matchesM) != 0 {
		t.Fatalf("expected 0 matches for 'm' inside ANSI code, got %d", len(matchesM))
	}
	highlightedM := HighlightMatchesInLine(line, "m", theme)
	if highlightedM != line {
		t.Fatalf("expected line unchanged when matching 'm', got %q", highlightedM)
	}

	// Searching for "blue" should highlight "blue" while preserving the surrounding blue text formatting
	matchesBlue := FindMatches([]string{line}, "blue")
	if len(matchesBlue) != 1 {
		t.Fatalf("expected 1 match for 'blue', got %d", len(matchesBlue))
	}
	highlightedBlue := HighlightMatchesInLine(line, "blue", theme)
	if !strings.Contains(highlightedBlue, "blue") || !strings.Contains(highlightedBlue, "text") {
		t.Fatalf("expected highlight to contain blue and text: %q", highlightedBlue)
	}
	// Verify that the trailing text still has the active ANSI sequence re-emitted
	if !strings.Contains(highlightedBlue, "\x1b[38;2;56;189;248m text") {
		t.Fatalf("expected active ANSI color to be preserved for subsequent text: %q", highlightedBlue)
	}
}
