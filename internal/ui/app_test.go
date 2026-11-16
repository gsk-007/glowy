package ui

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/discovery"
)

func TestAppFolderModeControls(t *testing.T) {
	tempDir := t.TempDir()
	origAppData := os.Getenv("APPDATA")
	defer os.Setenv("APPDATA", origAppData)
	_ = os.Setenv("APPDATA", tempDir)

	samplePath := filepath.Join(tempDir, "sample.md")
	if err := os.WriteFile(samplePath, []byte("# Hello\n\nBody line.\n"), 0644); err != nil {
		t.Fatal(err)
	}

	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme(cfg.DefaultScheme))
	files := []discovery.MarkdownFile{
		{Name: "b.md", Path: samplePath, RelativePath: "b.md", ModifiedAt: time.Now()},
		{Name: "a.md", Path: samplePath, RelativePath: "a.md", ModifiedAt: time.Now().Add(-time.Minute)},
	}

	app, err := NewAppModel(files, nil, "", false, cfg, theme, "formatted", false, 0)
	if err != nil {
		t.Fatalf("NewAppModel failed: %v", err)
	}

	model, _ := app.Update(tea.WindowSizeMsg{Width: 100, Height: 30})
	app = model.(AppModel)

	// 1. Open Theme Modal with 't' (and uppercase 'T') in folder mode, confirm with Enter
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'T'}})
	app = model.(AppModel)
	if !app.ShowingTheme {
		t.Fatalf("expected ShowingTheme=true after pressing 'T' in folder mode")
	}
	if !strings.Contains(StripANSI(app.View()), "COLOR SCHEMES") {
		t.Fatalf("expected COLOR SCHEMES modal in view")
	}

	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(AppModel)
	if app.Theme.Scheme.Name != "SlateDark" {
		t.Fatalf("expected live preview to switch to SlateDark, got %s", app.Theme.Scheme.Name)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(AppModel)
	if app.ShowingTheme || app.Theme.Scheme.Name != "SlateDark" || app.Config.DefaultScheme != "SlateDark" {
		t.Fatalf("expected Enter to confirm SlateDark, got showing=%v theme=%s default=%s", app.ShowingTheme, app.Theme.Scheme.Name, app.Config.DefaultScheme)
	}

	// 2. Help Modal with '?'
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	app = model.(AppModel)
	if !app.ShowingHelp {
		t.Fatalf("expected ShowingHelp=true after pressing '?'")
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = model.(AppModel)
	if app.ShowingHelp {
		t.Fatalf("expected Esc to close Help modal")
	}

	// 3. Sort 's' / 'S'
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'S'}})
	app = model.(AppModel)
	if app.Browser.SortMode != "name" {
		t.Fatalf("expected SortMode='name', got %s", app.Browser.SortMode)
	}

	// 4. Filter '/' + arrow navigation while filtering
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	app = model.(AppModel)
	if !app.Browser.Filtering {
		t.Fatalf("expected Filtering=true after '/'")
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyDown})
	app = model.(AppModel)
	if app.Browser.Cursor != 1 {
		t.Fatalf("expected arrow down to move cursor to 1 while filtering, got %d", app.Browser.Cursor)
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEsc})
	app = model.(AppModel)
	if app.Browser.Filtering {
		t.Fatalf("expected Esc to exit filtering")
	}

	// 5. Open file with Enter -> toggle 'r'/'l' -> return with 'b' -> verify state preserved
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyEnter})
	app = model.(AppModel)
	if app.State != StatePager || app.Pager == nil {
		t.Fatalf("expected StatePager after Enter on file")
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'L'}})
	app = model.(AppModel)
	if !app.Pager.LineNumbers {
		t.Fatalf("expected 'L' to toggle line numbers on in pager")
	}
	model, _ = app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'B'}})
	app = model.(AppModel)
	if app.State != StateBrowser || app.Pager != nil || !app.LineNumbers {
		t.Fatalf("expected 'B' to return cleanly to browser preserving LineNumbers=true, got state=%v pager=%v ln=%v", app.State, app.Pager, app.LineNumbers)
	}
}
