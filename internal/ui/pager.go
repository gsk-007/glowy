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
	}

	return m, nil
}

func (m *PagerModel) reRender() error {
	opts := render.RenderOptions{
		Mode:        m.Mode,
		LineNumbers: m.LineNumbers,
		Width:       m.Width,
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
		highlighted[i] = HighlightMatchesInLine(line, m.SearchQuery, m.Theme)
	}

	oldY := m.Viewport.YOffset
	m.Viewport.SetContent(strings.Join(highlighted, "\n"))
	m.Viewport.SetYOffset(oldY)
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
		m.Viewport.Height = msg.Height - 1 // Leave 1 line for status bar
		oldPct := m.Viewport.ScrollPercent()
		_ = m.reRender()
		newTotal := m.Viewport.TotalLineCount()
		newY := int(float64(newTotal-m.Viewport.Height) * oldPct)
		if newY < 0 {
			newY = 0
		}
		m.Viewport.SetYOffset(newY)
		return m, nil

	case tea.MouseMsg:
		m.Viewport, cmd = m.Viewport.Update(msg)
		return m, cmd

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

func (m PagerModel) View() string {
	if m.Width == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteString(m.Viewport.View())
	sb.WriteString("\n")

	// Render bottom status bar
	modeDisplay := "Formatted"
	if m.Mode == "raw" {
		modeDisplay = "Raw"
	}

	pct := int(m.Viewport.ScrollPercent() * 100)
	currentLine := m.Viewport.YOffset + 1

	statusParts := []string{
		fmt.Sprintf(" %s ", m.FileName),
		fmt.Sprintf(" %s ", modeDisplay),
		fmt.Sprintf(" Ln %d (%d%%) ", currentLine, pct),
	}

	if m.LineNumbers {
		statusParts = append(statusParts, " [LN:On] ")
	}

	if m.Searching {
		statusParts = append(statusParts, " "+m.SearchInput.View()+" ")
	} else if len(m.SearchMatches) > 0 {
		statusParts = append(statusParts, fmt.Sprintf(" %d/%d matches ", m.CurrentMatchIdx+1, len(m.SearchMatches)))
	}

	statusText := strings.Join(statusParts, "│")
	renderedStatus := m.Theme.StatusBar.Width(m.Width).Render(statusText)
	sb.WriteString(renderedStatus)

	return sb.String()
}
