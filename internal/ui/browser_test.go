package ui

import (
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

func TestPagerModelModes(t *testing.T) {
	theme := config.NewUITheme(config.DefaultSlateDark())
	source := "# Test Heading\n\nContent paragraph."

	pager, err := NewPagerModel("test.md", source, theme, "formatted", false, true)
	if err != nil {
		t.Fatalf("NewPagerModel failed: %v", err)
	}

	// Window resize
	pager, _ = pager.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	if pager.Rendered == nil {
		t.Fatal("expected rendered result after resize")
	}

	// Toggle mode 'r'
	pager, _ = pager.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	if pager.Mode != "raw" {
		t.Fatalf("expected raw mode, got %s", pager.Mode)
	}

	// Toggle line numbers 'l'
	pager, _ = pager.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'l'}})
	if !pager.LineNumbers {
		t.Fatal("expected line numbers enabled")
	}
}
