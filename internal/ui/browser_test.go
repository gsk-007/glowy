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
		{Name: "alpha.md", RelativePath: "docs/alpha.md", Directory: "docs", ModifiedAt: now},
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

	// Filter by directory path (e.g. "docs")
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	if !model.Filtering {
		t.Fatal("expected filtering mode active")
	}

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	if len(model.DisplayedFiles) != 1 || model.DisplayedFiles[0].Name != "alpha.md" {
		t.Fatalf("expected 1 filtered file 'alpha.md' by path 'doc', got %+v", model.DisplayedFiles)
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

func TestHelpModalAndThemeModalOverlay(t *testing.T) {
	cfg := config.DefaultConfig()
	theme := config.NewUITheme(cfg.GetScheme("SlateDark"))

	// Verify Help Modal renders cleanly
	helpView := RenderHelpModal(theme, 80)
	if !strings.Contains(helpView, "KEYBOARD SHORTCUTS") {
		t.Fatalf("expected Help modal to contain title, got: %s", helpView)
	}

	// Verify Theme Modal renders cleanly
	themeModal := NewThemeModal(cfg.Schemes, "SlateDark")
	modalView := themeModal.Render(theme, 80)
	if !strings.Contains(modalView, "COLOR SCHEMES") || !strings.Contains(modalView, "SlateDark") {
		t.Fatalf("expected Theme modal to contain SlateDark, got: %s", modalView)
	}

	// Verify Overlay preserves height and keeps dimmed backdrop text visible around the modal
	var baseRows []string
	for i := 0; i < 24; i++ {
		baseRows = append(baseRows, fmt.Sprintf("LEFT_EDGE_%02d%sRIGHT_EDGE_%02d", i, strings.Repeat(" ", 55), i))
	}
	base := strings.Join(baseRows, "\n")
	overlaid := Overlay(base, modalView, 80, 24, theme.Scheme.Background)
	overlaidLines := strings.Split(overlaid, "\n")
	if len(overlaidLines) != 24 {
		t.Fatalf("expected exactly 24 lines after overlay, got %d", len(overlaidLines))
	}
	plainOverlaid := StripANSI(overlaid)
	if !strings.Contains(plainOverlaid, "LEFT_EDGE_10") || !strings.Contains(plainOverlaid, "RIGHT_EDGE_10") {
		t.Fatalf("expected dimmed backdrop flanks to remain visible beside modal, got:\n%s", plainOverlaid)
	}
	if !strings.Contains(plainOverlaid, "COLOR SCHEMES") {
		t.Fatalf("expected modal content to remain visible in overlay")
	}
}

func TestPagerStatusPillAndSearchVisibleAt80Cols(t *testing.T) {
	theme := config.NewUITheme(config.DefaultSlateDark())
	pager, err := NewPagerModel("README.md", "# Hello World\n\nSearchable content here.\n", theme, "formatted", true, true)
	if err != nil {
		t.Fatalf("NewPagerModel failed: %v", err)
	}

	pager, _ = pager.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	view := StripANSI(pager.View())
	if !strings.Contains(view, "[FMT]") || !strings.Contains(view, "[LN: ON]") {
		t.Fatalf("expected header badges [FMT] and [LN: ON] to be visible on 80-col screen, got:\n%s", view)
	}
	if !strings.Contains(view, "TOP") {
		t.Fatalf("expected bottom-right status pill TOP to be visible on 80-col screen, got:\n%s", view)
	}

	// Activate search '/'
	pager, _ = pager.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'/'}})
	for _, r := range "world" {
		pager, _ = pager.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	pager, _ = pager.Update(tea.KeyMsg{Type: tea.KeyEnter})

	viewMatch := StripANSI(pager.View())
	if !strings.Contains(viewMatch, "1/1 matches") {
		t.Fatalf("expected '1/1 matches' pill to be visible on 80-col screen, got:\n%s", viewMatch)
	}
}

func assertNoUnguardedResets(t *testing.T, line string, expectedBgSeq string, expectedWidth int, label string) {
	t.Helper()
	if !strings.HasPrefix(line, expectedBgSeq) {
		t.Fatalf("%s: expected line to start with %q, got %q", label, expectedBgSeq, line)
	}
	if !strings.HasSuffix(line, "\x1b[0m") {
		t.Fatalf("%s: expected line to end with \\x1b[0m, got %q", label, line)
	}
	// Strip the final reset and verify every inner reset is immediately followed by a 24-bit background sequence
	inner := strings.TrimSuffix(line, "\x1b[0m")
	idx := 0
	for {
		pos := strings.Index(inner[idx:], "\x1b[0m")
		if pos < 0 {
			break
		}
		after := inner[idx+pos+len("\x1b[0m"):]
		if !strings.HasPrefix(after, "\x1b[48;2;") {
			t.Fatalf("%s: found unguarded \\x1b[0m reset without background sequence at %q", label, inner[idx+pos:])
		}
		idx += pos + len("\x1b[0m")
	}
	if w := len([]rune(StripANSI(line))); w != expectedWidth {
		t.Fatalf("%s: expected visible width %d, got %d", label, expectedWidth, w)
	}
}

func TestLightThemeBarsHaveNoBackgroundGaps(t *testing.T) {
	cfg := config.DefaultConfig()
	paper := config.NewUITheme(cfg.GetScheme("PaperLight"))
	bgSeq := "\x1b[48;2;250;250;250m"
	sbBgSeq := "\x1b[48;2;241;245;249m"

	// 1. Pager top & bottom bars
	pager, err := NewPagerModel("docs/sample.md", "# Title\n\n| A | B |\n| - | - |\n| 1 | 2 |\n", paper, "formatted", false, true)
	if err != nil {
		t.Fatalf("NewPagerModel failed: %v", err)
	}
	pager, _ = pager.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	pLines := strings.Split(pager.View(), "\n")
	if len(pLines) != 24 {
		t.Fatalf("expected 24 lines in pager view, got %d", len(pLines))
	}
	assertNoUnguardedResets(t, pLines[0], sbBgSeq, 80, "pager header")
	assertNoUnguardedResets(t, pLines[len(pLines)-1], sbBgSeq, 80, "pager footer")

	// 2. Browser top & bottom bars
	files := []discovery.MarkdownFile{
		{Name: "README.md", RelativePath: "README.md", ModifiedAt: time.Now()},
	}
	browser := NewBrowserModel(files, paper, "modified")
	browser, _ = browser.Update(tea.WindowSizeMsg{Width: 80, Height: 24})
	bLines := strings.Split(browser.View(), "\n")
	if len(bLines) != 24 {
		t.Fatalf("expected 24 lines in browser view, got %d", len(bLines))
	}
	assertNoUnguardedResets(t, bLines[0], bgSeq, 80, "browser header")
	assertNoUnguardedResets(t, bLines[len(bLines)-1], sbBgSeq, 80, "browser footer")
}

func TestPagerBackKeyInFolderMode(t *testing.T) {
	theme := config.NewUITheme(config.DefaultPaperLight())

	// Folder mode (standalone=false): pressing 'b' returns to browser
	pagerFolder, err := NewPagerModel("README.md", "# Hello\n", theme, "formatted", false, false)
	if err != nil {
		t.Fatalf("NewPagerModel failed: %v", err)
	}
	pagerFolder, _ = pagerFolder.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	if !strings.Contains(StripANSI(pagerFolder.View()), "b/esc back") {
		t.Fatalf("expected footer hint to advertise 'b/esc back' in folder mode")
	}
	pagerFolder, _ = pagerFolder.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if !pagerFolder.ReturnToBrowser {
		t.Fatalf("expected 'b' in folder mode to set ReturnToBrowser=true")
	}

	// Standalone mode (standalone=true): pressing 'b' should not quit or return
	pagerSolo, err := NewPagerModel("README.md", "# Hello\n", theme, "formatted", false, true)
	if err != nil {
		t.Fatalf("NewPagerModel failed: %v", err)
	}
	pagerSolo, _ = pagerSolo.Update(tea.WindowSizeMsg{Width: 100, Height: 24})
	pagerSolo, _ = pagerSolo.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
	if pagerSolo.ReturnToBrowser || pagerSolo.Canceled {
		t.Fatalf("expected 'b' in standalone mode to be a no-op, got ReturnToBrowser=%v Canceled=%v", pagerSolo.ReturnToBrowser, pagerSolo.Canceled)
	}
}

