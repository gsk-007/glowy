package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	"github.com/charmbracelet/bubbles/viewport"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/fsnotify/fsnotify"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/render"
)

type fileReloadMsg struct {
	content string
}

// PagerModel represents the interactive full-screen document pager.
type PagerModel struct {
	FilePath      string
	FileName      string
	SourceContent string
	Rendered      *render.RenderResult

	Mode        string // "formatted" or "raw"
	LineNumbers bool
	WrapWidth   int // optional fixed wrap width override (-w or settings.json wordWrap)

	Theme    *config.UITheme
	Viewport viewport.Model

	Searching       bool
	SearchInput     textinput.Model
	SearchQuery     string
	SearchMatches   []SearchMatch
	CurrentMatchIdx int

	Watcher *fsnotify.Watcher
	Width   int
	Height  int

	ReturnToBrowser bool
	Canceled        bool
	Standalone      bool // True if opened directly via glowy -p
	InitialLoad     bool
}

// NewPagerModel creates a new PagerModel for the given file content.
func NewPagerModel(filePath string, source string, theme *config.UITheme, mode string, lineNumbers bool, standalone bool) (PagerModel, error) {
	if mode == "" {
		mode = "formatted"
	}

	name := filepath.Base(filePath)
	if filePath == "" || filePath == "-" {
		name = "stdin"
	}

	ti := textinput.New()
	ti.Placeholder = "Search document..."
	ti.Prompt = "/ "
	ti.CharLimit = 100

	vp := viewport.New(80, 24)
	vp.YOffset = 0
	vp.Style = lipgloss.NewStyle().Background(lipgloss.Color(theme.Scheme.Background))

	var watcher *fsnotify.Watcher
	if filePath != "" && filePath != "-" {
		if w, err := fsnotify.NewWatcher(); err == nil {
			_ = w.Add(filePath)
			watcher = w
		}
	}

	m := PagerModel{
		FilePath:        filePath,
		FileName:        name,
		SourceContent:   source,
		Mode:            mode,
		LineNumbers:     lineNumbers,
		Theme:           theme,
		Viewport:        vp,
		SearchInput:     ti,
		CurrentMatchIdx: -1,
		Watcher:         watcher,
		Standalone:      standalone,
		InitialLoad:     true,
	}

	return m, nil
}

func (m *PagerModel) reRender() error {
	m.Viewport.Style = lipgloss.NewStyle().Background(lipgloss.Color(m.Theme.Scheme.Background))

	targetWrapWidth := m.Width
	if m.WrapWidth > 0 && (targetWrapWidth <= 0 || m.WrapWidth < targetWrapWidth) {
		targetWrapWidth = m.WrapWidth
	}

	opts := render.RenderOptions{
		Mode:        m.Mode,
		LineNumbers: m.LineNumbers,
		Width:       targetWrapWidth,
		CanvasWidth: m.Width,
		Theme:       m.Theme,
	}

	res, err := render.Render(m.SourceContent, opts)
	if err != nil {
		return err
	}
	m.Rendered = res

	if m.SearchQuery != "" {
		m.applySearchHighlight()
	} else {
		m.Viewport.SetContent(res.Content)
	}

	if m.InitialLoad {
		m.Viewport.GotoTop()
		m.Viewport.YOffset = 0
		m.InitialLoad = false
	}

	return nil
}

func (m *PagerModel) applySearchHighlight() {
	if m.Rendered == nil {
		return
	}

	m.SearchMatches = FindMatches(m.Rendered.Lines, m.SearchQuery)
	if len(m.SearchMatches) > 0 {
		if m.CurrentMatchIdx < 0 || m.CurrentMatchIdx >= len(m.SearchMatches) {
			m.CurrentMatchIdx = 0
		}
	} else {
		m.CurrentMatchIdx = -1
	}

	highlighted := make([]string, len(m.Rendered.Lines))
	for i, line := range m.Rendered.Lines {
		hl := HighlightMatchesInLine(line, m.SearchQuery, m.Theme)
		highlighted[i] = render.ApplyLineBackground(hl, m.Width, m.Theme.Scheme.Background)
	}

	oldY := m.Viewport.YOffset
	m.Viewport.SetContent(strings.Join(highlighted, "\n"))
	if !m.InitialLoad {
		m.Viewport.SetYOffset(oldY)
	} else {
		m.Viewport.GotoTop()
	}
}

func (m *PagerModel) scrollToMatch(idx int) {
	if idx < 0 || idx >= len(m.SearchMatches) {
		return
	}
	match := m.SearchMatches[idx]
	targetLine := match.LineIndex

	targetY := targetLine - (m.Viewport.Height / 3)
	if targetY < 0 {
		targetY = 0
	}
	m.Viewport.SetYOffset(targetY)
}

func watchFileCmd(watcher *fsnotify.Watcher, path string) tea.Cmd {
	if watcher == nil || path == "" || path == "-" {
		return nil
	}
	return func() tea.Msg {
		for {
			select {
			case event, ok := <-watcher.Events:
				if !ok {
					return nil
				}
				if event.Op&(fsnotify.Write|fsnotify.Create|fsnotify.Rename|fsnotify.Remove) != 0 {
					// Debounce rapid edit bursts
					time.Sleep(80 * time.Millisecond)

					// Re-add watch in case of atomic file replacement on Windows
					_ = watcher.Remove(path)
					_ = watcher.Add(path)

					if content, err := os.ReadFile(path); err == nil {
						return fileReloadMsg{content: string(content)}
					}
				}
			case _, ok := <-watcher.Errors:
				if !ok {
					return nil
				}
			}
		}
	}
}

func (m PagerModel) Init() tea.Cmd {
	return tea.Batch(
		watchFileCmd(m.Watcher, m.FilePath),
	)
}

func (m PagerModel) Update(msg tea.Msg) (PagerModel, tea.Cmd) {
	var cmd tea.Cmd
	var cmds []tea.Cmd

	switch msg := msg.(type) {
	case fileReloadMsg:
		m.SourceContent = msg.content
		oldY := m.Viewport.YOffset
		_ = m.reRender()
		m.Viewport.SetYOffset(oldY)
		return m, watchFileCmd(m.Watcher, m.FilePath)

	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.Viewport.Width = msg.Width
		vpHeight := msg.Height - 2 // Strictly 1 line top header, 1 line status bar
		if vpHeight < 1 {
			vpHeight = 1
		}
		m.Viewport.Height = vpHeight

		if m.InitialLoad {
			_ = m.reRender()
			m.Viewport.GotoTop()
			m.Viewport.YOffset = 0
			m.InitialLoad = false
		} else {
			oldPct := m.Viewport.ScrollPercent()
			_ = m.reRender()
			newTotal := m.Viewport.TotalLineCount()
			newY := int(float64(newTotal-m.Viewport.Height) * oldPct)
			if newY < 0 {
				newY = 0
			}
			m.Viewport.SetYOffset(newY)
		}
		return m, nil

	case tea.KeyMsg:
		if m.Searching {
			switch msg.String() {
			case "esc":
				m.Searching = false
				m.SearchInput.Blur()
				return m, nil
			case "enter":
				m.Searching = false
				m.SearchInput.Blur()
				m.SearchQuery = strings.TrimSpace(m.SearchInput.Value())
				m.applySearchHighlight()
				if len(m.SearchMatches) > 0 {
					m.CurrentMatchIdx = 0
					m.scrollToMatch(m.CurrentMatchIdx)
				}
				return m, nil
			default:
				m.SearchInput, cmd = m.SearchInput.Update(msg)
				m.SearchQuery = strings.TrimSpace(m.SearchInput.Value())
				m.applySearchHighlight()
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			if m.Watcher != nil {
				_ = m.Watcher.Close()
				m.Watcher = nil
			}
			m.Canceled = true
			return m, tea.Quit

		case "esc", "backspace":
			if m.SearchQuery != "" {
				m.SearchQuery = ""
				m.SearchMatches = nil
				m.CurrentMatchIdx = -1
				m.SearchInput.Reset()
				if m.Rendered != nil {
					oldY := m.Viewport.YOffset
					m.Viewport.SetContent(m.Rendered.Content)
					m.Viewport.SetYOffset(oldY)
				}
				return m, nil
			}

			if m.Watcher != nil {
				_ = m.Watcher.Close()
				m.Watcher = nil
			}

			if m.Standalone {
				m.Canceled = true
				return m, tea.Quit
			}

			m.ReturnToBrowser = true
			return m, nil

		case "/":
			m.Searching = true
			m.SearchInput.Focus()
			return m, textinput.Blink

		case "n":
			if len(m.SearchMatches) > 0 {
				m.CurrentMatchIdx = (m.CurrentMatchIdx + 1) % len(m.SearchMatches)
				m.scrollToMatch(m.CurrentMatchIdx)
			}

		case "N":
			if len(m.SearchMatches) > 0 {
				m.CurrentMatchIdx--
				if m.CurrentMatchIdx < 0 {
					m.CurrentMatchIdx = len(m.SearchMatches) - 1
				}
				m.scrollToMatch(m.CurrentMatchIdx)
			}

		case "r":
			if m.Mode == "formatted" {
				m.Mode = "raw"
			} else {
				m.Mode = "formatted"
			}
			oldPct := m.Viewport.ScrollPercent()
			_ = m.reRender()
			newTotal := m.Viewport.TotalLineCount()
			newY := int(float64(newTotal-m.Viewport.Height) * oldPct)
			if newY < 0 {
				newY = 0
			}
			m.Viewport.SetYOffset(newY)

		case "l":
			m.LineNumbers = !m.LineNumbers
			oldY := m.Viewport.YOffset
			_ = m.reRender()
			m.Viewport.SetYOffset(oldY)

		case "up", "k":
			m.Viewport.LineUp(1)

		case "down", "j":
			m.Viewport.LineDown(1)

		case "pgup", "u":
			m.Viewport.HalfViewUp()

		case "pgdown", "d":
			m.Viewport.HalfViewDown()

		case "home", "g":
			m.Viewport.GotoTop()

		case "end", "G":
			m.Viewport.GotoBottom()
		}
	}

	return m, tea.Batch(cmds...)
}

func truncateLeft(s string, maxWidth int) string {
	if maxWidth <= 0 {
		return ""
	}
	if lipgloss.Width(s) <= maxWidth {
		return s
	}
	if maxWidth <= 1 {
		return "…"
	}
	runes := []rune(s)
	var rev []rune
	for i := len(runes) - 1; i >= 0; i-- {
		candidate := "…" + string(append([]rune{runes[i]}, rev...))
		if lipgloss.Width(candidate) > maxWidth {
			break
		}
		rev = append([]rune{runes[i]}, rev...)
	}
	return "…" + string(rev)
}

func (m PagerModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}

	// 1. Top Header Bar (1 line fixed)
	displayPath := m.FilePath
	if displayPath == "" || displayPath == "-" {
		displayPath = "stdin"
	} else {
		displayPath = filepath.ToSlash(displayPath)
	}

	fmtBadge := "[FMT]"
	if m.Mode == "raw" {
		fmtBadge = "[RAW]"
	}
	lnBadge := "[LN: OFF]"
	if m.LineNumbers {
		lnBadge = "[LN: ON]"
	}
	headerRight := fmt.Sprintf("%s %s ", m.Theme.Badge.Render(fmtBadge), m.Theme.TabActive.Render(lnBadge))
	rightLen := lipgloss.Width(headerRight)

	badgeText := m.Theme.Badge.Render("DOC")
	prefixLeft := fmt.Sprintf(" %s %s ", badgeText, m.Theme.Muted.Render("❯"))
	prefixLen := lipgloss.Width(prefixLeft)

	maxPathW := m.Width - prefixLen - rightLen - 2
	if maxPathW < 4 {
		maxPathW = 4
	}
	displayPath = truncateLeft(displayPath, maxPathW)
	headerLeft := prefixLeft + m.Theme.RowNormal.Bold(true).Render(displayPath)

	leftLen := lipgloss.Width(headerLeft)
	spaceLen := m.Width - leftLen - rightLen
	if spaceLen < 1 {
		spaceLen = 1
	}
	headerLine := headerLeft + strings.Repeat(" ", spaceLen) + headerRight
	headerRendered := m.Theme.PagerHeader.Width(m.Width).MaxHeight(1).Render(headerLine)

	// 2. Middle Viewport (strictly m.Height - 2 lines)
	vpHeight := m.Height - 2
	if vpHeight < 1 {
		vpHeight = 1
	}
	vpLines := strings.Split(m.Viewport.View(), "\n")
	for i, l := range vpLines {
		vpLines[i] = render.ApplyLineBackground(l, m.Width, m.Theme.Scheme.Background)
	}
	for len(vpLines) < vpHeight {
		vpLines = append(vpLines, render.ApplyLineBackground("", m.Width, m.Theme.Scheme.Background))
	}
	if len(vpLines) > vpHeight {
		vpLines = vpLines[:vpHeight]
	}
	viewportView := strings.Join(vpLines, "\n")

	// 3. Bottom Status & Command Bar (1 line fixed)
	scrollStatus := " TOP "
	if m.Viewport.YOffset > 0 && m.Viewport.AtBottom() {
		scrollStatus = " BOT "
	} else if m.Viewport.YOffset > 0 {
		pct := int(m.Viewport.ScrollPercent() * 100)
		scrollStatus = fmt.Sprintf(" Ln %d • %d%% ", m.Viewport.YOffset+1, pct)
	}

	var rightInfo string
	if m.Searching {
		rightInfo = fmt.Sprintf(" %s • %s ", m.SearchInput.View(), scrollStatus)
	} else if len(m.SearchMatches) > 0 {
		rightInfo = fmt.Sprintf(" %d/%d matches • %s ", m.CurrentMatchIdx+1, len(m.SearchMatches), scrollStatus)
	} else if m.SearchQuery != "" {
		rightInfo = fmt.Sprintf(" 0 matches • %s ", scrollStatus)
	} else {
		rightInfo = fmt.Sprintf(" %s ", scrollStatus)
	}
	rightRendered := m.Theme.StatusPill.Render(rightInfo)
	rLen := lipgloss.Width(rightRendered)

	hintsCandidates := []string{
		" ↑/↓ scroll • / search • r raw • l lines • t theme • ? help • esc back • q quit ",
		" ↑/↓ scroll • / search • r raw • l lines • ? help • q quit ",
		" ↑/↓ • / search • r raw • l lines • ? help • q ",
		" ? help • q quit ",
		" ",
	}

	var hintRendered string
	for _, candidate := range hintsCandidates {
		rendered := m.Theme.StatusBar.Render(candidate)
		if lipgloss.Width(rendered)+rLen+1 <= m.Width {
			hintRendered = rendered
			break
		}
	}
	if hintRendered == "" {
		hintRendered = m.Theme.StatusBar.Render(" ")
	}

	hLen := lipgloss.Width(hintRendered)
	footerSpaces := m.Width - hLen - rLen
	if footerSpaces < 1 {
		footerSpaces = 1
	}
	footerLine := hintRendered + strings.Repeat(" ", footerSpaces) + rightRendered
	footerRendered := m.Theme.StatusBar.Width(m.Width).MaxHeight(1).Render(footerLine)

	// Combine strictly into m.Height lines
	return headerRendered + "\n" + viewportView + "\n" + footerRendered
}
