package ui

import (
	"fmt"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/discovery"
)

func TestBrowserModelNavigationAndFilter(t *testing.T) {
	theme := config.NewUITheme(config.DefaultSlateDark())
	now := time.Now()
	files := []discovery.MarkdownFile{
		{Name: "zeta.md", RelativePath: "zeta.md", ModifiedAt: now.Add(-10 * time.Minute)},
		{Name: "alpha.md", RelativePath: "docs/alpha.md", ModifiedAt: now},
		{Name: "beta.md", RelativePath: "beta.md", ModifiedAt: now.Add(-5 * time.Minute)},
	}

	model := NewBrowserModel(files, theme, "modified")
	if len(model.DisplayedFiles) != 3 {
		t.Fatalf("expected 3 files, got %d", len(model.DisplayedFiles))
	}

	// First by modified should be alpha.md
	if model.DisplayedFiles[0].Name != "alpha.md" {
		t.Fatalf("expected alpha.md first, got %s", model.DisplayedFiles[0].Name)
	}

	// Move cursor down
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'j'}})
	if model.Cursor != 1 {
		t.Fatalf("expected cursor 1, got %d", model.Cursor)
	}

	// Filter
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !model.Filtering {
		t.Fatal("expected filtering mode active")
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	if len(model.DisplayedFiles) != 1 || model.DisplayedFiles[0].Name != "zeta.md" {
		t.Fatalf("expected 1 filtered file 'zeta.md', got %+v", model.DisplayedFiles)
	}

	// Exit filter
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	if model.Filtering {
		t.Fatal("expected filtering mode exited on Esc")
	}
}

func TestBrowserStrictHeightZeroLayoutShift(t *testing.T) {
	theme := config.NewUITheme(config.DefaultSlateDark())
	files := []discovery.MarkdownFile{
		{Name: "doc1.md", RelativePath: "doc1.md"},
		{Name: "doc2.md", RelativePath: "doc2.md"},
	}

	model := NewBrowserModel(files, theme, "modified")
	targetHeight := 24
	targetWidth := 80
	model, _ = model.Update(tea.WindowSizeMsg{Width: targetWidth, Height: targetHeight})

	// View with 2 files
	view2 := model.View()
	lines2 := strings.Split(view2, "\n")
	if len(lines2) != targetHeight {
		t.Fatalf("expected exactly %d lines in view with 2 files, got %d", targetHeight, len(lines2))
	}

	// View with 0 matching files (filter down to 0)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})

	view0 := model.View()
	lines0 := strings.Split(view0, "\n")
	if len(lines0) != targetHeight {
		t.Fatalf("expected exactly %d lines in view with 0 files, got %d", targetHeight, len(lines0))
	}
}

func TestPagerModelStartsAtTop(t *testing.T) {
	theme := config.NewUITheme(config.DefaultSlateDark())
	var longContent strings.Builder
	for i := 1; i <= 200; i++ {
		longContent.WriteString(fmt.Sprintf("# Heading %d\n\nLine of text %d.\n\n", i, i))
	}

	pager, err := NewPagerModel("large.md", longContent.String(), theme, "formatted", false, true)
	if err != nil {
		t.Fatalf("NewPagerModel failed: %v", err)
	}

	// Deliver initial WindowSizeMsg
	pager, _ = pager.Update(tea.WindowSizeMsg{Width: 80, Height: 24})

	// Pager must start at line 0 (Top)
	if pager.Viewport.YOffset != 0 {
		t.Fatalf("expected pager YOffset to be 0 on initial load, got %d", pager.Viewport.YOffset)
	}

	// Check view line count matches Height exactly
	view := pager.View()
	lines := strings.Split(view, "\n")
	if len(lines) != 24 {
		t.Fatalf("expected exactly 24 lines in pager view, got %d", len(lines))
	}
}
