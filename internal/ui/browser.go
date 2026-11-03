package ui

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/bubbles/textinput"
	tea "github.com/charmbracelet/bubbletea"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/discovery"
	"github.com/sahilm/fuzzy"
)

type fileListSource []discovery.MarkdownFile

func (fl fileListSource) String(i int) string {
	return fl[i].RelativePath
}

func (fl fileListSource) Len() int {
	return len(fl)
}

// BrowserModel represents the Card Feed TUI File Browser state.
type BrowserModel struct {
	AllFiles       []discovery.MarkdownFile
	DisplayedFiles []discovery.MarkdownFile
	SortMode       string // "modified", "name", "created"
	Cursor         int
	ScrollOffset   int
	Width          int
	Height         int

	Filtering   bool
	FilterInput textinput.Model

	Theme        *config.UITheme
	SelectedFile *discovery.MarkdownFile
	Canceled     bool
}

// NewBrowserModel initializes the Card Feed File Browser model.
func NewBrowserModel(files []discovery.MarkdownFile, theme *config.UITheme, defaultSort string) BrowserModel {
	if defaultSort == "" {
		defaultSort = "modified"
	}

	ti := textinput.New()
	ti.Placeholder = "Search by name or path..."
	ti.Prompt = "❯ "
	ti.CharLimit = 64

	m := BrowserModel{
		AllFiles:     files,
		SortMode:     defaultSort,
		Cursor:       0,
		ScrollOffset: 0,
		FilterInput:  ti,
		Theme:        theme,
	}

	m.applyFilterAndSort()
	return m
}

func (m *BrowserModel) applyFilterAndSort() {
	query := strings.TrimSpace(m.FilterInput.Value())

	var candidates []discovery.MarkdownFile
	if query == "" {
		candidates = make([]discovery.MarkdownFile, len(m.AllFiles))
		copy(candidates, m.AllFiles)
	} else {
		matches := fuzzy.FindFrom(query, fileListSource(m.AllFiles))
		for _, match := range matches {
			candidates = append(candidates, m.AllFiles[match.Index])
		}
	}

	discovery.SortFiles(candidates, m.SortMode)
	m.DisplayedFiles = candidates

	totalItems := len(m.DisplayedFiles)
	if m.Cursor >= totalItems {
		m.Cursor = totalItems - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}

	m.adjustScroll()
}

func (m *BrowserModel) cycleSort() {
	switch m.SortMode {
	case "modified":
		m.SortMode = "name"
	case "name":
		m.SortMode = "created"
	case "created":
		m.SortMode = "modified"
	default:
		m.SortMode = "modified"
	}
	m.applyFilterAndSort()
}

func (m *BrowserModel) adjustScroll() {
	bodyHeight := m.Height - 2 // Top bar (1) + Bottom bar (1)
	if bodyHeight <= 0 {
		bodyHeight = 9
	}
	visibleCards := bodyHeight / 3
	if visibleCards < 1 {
		visibleCards = 1
	}

	if m.Cursor < m.ScrollOffset {
		m.ScrollOffset = m.Cursor
	}
	if m.Cursor >= m.ScrollOffset+visibleCards {
		m.ScrollOffset = m.Cursor - visibleCards + 1
	}
	if m.ScrollOffset < 0 {
		m.ScrollOffset = 0
	}
}

func (m BrowserModel) Init() tea.Cmd {
	return nil
}

func (m BrowserModel) Update(msg tea.Msg) (BrowserModel, tea.Cmd) {
	var cmd tea.Cmd

	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		m.adjustScroll()
		return m, nil

	case tea.KeyMsg:
		if m.Filtering {
			switch msg.String() {
			case "esc":
				m.Filtering = false
				m.FilterInput.Blur()
				return m, nil
			case "enter":
				m.Filtering = false
				m.FilterInput.Blur()
				return m, nil
			default:
				m.FilterInput, cmd = m.FilterInput.Update(msg)
				m.applyFilterAndSort()
				return m, cmd
			}
		}

		switch msg.String() {
		case "q", "ctrl+c":
			m.Canceled = true
			return m, tea.Quit

		case "esc":
			if m.FilterInput.Value() != "" {
				m.FilterInput.Reset()
				m.applyFilterAndSort()
			} else {
				m.Canceled = true
				return m, tea.Quit
			}

		case "/":
			m.Filtering = true
			m.FilterInput.Focus()
			return m, textinput.Blink

		case "s":
			m.cycleSort()

		case "up", "k":
			if m.Cursor > 0 {
				m.Cursor--
				m.adjustScroll()
			}

		case "down", "j":
			if m.Cursor < len(m.DisplayedFiles)-1 {
				m.Cursor++
				m.adjustScroll()
			}

		case "pgup", "u":
			pageSize := (m.Height - 2) / 6
			if pageSize < 1 {
				pageSize = 2
			}
			m.Cursor -= pageSize
			if m.Cursor < 0 {
				m.Cursor = 0
			}
			m.adjustScroll()

		case "pgdown", "d":
			pageSize := (m.Height - 2) / 6
			if pageSize < 1 {
				pageSize = 2
			}
			m.Cursor += pageSize
			if m.Cursor >= len(m.DisplayedFiles) {
				m.Cursor = len(m.DisplayedFiles) - 1
			}
			m.adjustScroll()

		case "home", "g":
			m.Cursor = 0
			m.adjustScroll()

		case "end", "G":
			if len(m.DisplayedFiles) > 0 {
				m.Cursor = len(m.DisplayedFiles) - 1
				m.adjustScroll()
			}

		case "enter":
			if len(m.DisplayedFiles) > 0 && m.Cursor < len(m.DisplayedFiles) {
				selected := m.DisplayedFiles[m.Cursor]
				m.SelectedFile = &selected
				return m, nil
			}
		}
	}

	return m, nil
}

func (m BrowserModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}

	// 1. Fixed Header & Footer lines
	headerLinesCount := 1
	footerLinesCount := 1
	bodyHeight := m.Height - headerLinesCount - footerLinesCount
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// ---------------------------------------------------------
	// 1. MINIMALIST TOP BAR (Line 0 - 1 line fixed)
	// ---------------------------------------------------------
	badge := m.Theme.Badge.Render(" GLOWY ")
	fileCountStr := fmt.Sprintf(" %d markdown files", len(m.DisplayedFiles))
	leftPart := badge + m.Theme.Muted.Render(fileCountStr)

	sortLabel := "Modified ↓"
	switch m.SortMode {
	case "name":
		sortLabel = "Name A-Z"
	case "created":
		sortLabel = "Created ↓"
	}

	var rightPart string
	if m.Filtering {
		rightPart = m.Theme.BrowserFilter.Render(" FILTER ❯ ") + m.FilterInput.View() + " "
	} else if m.FilterInput.Value() != "" {
		filterBadge := fmt.Sprintf(" Filter: %s (press / to edit, esc to clear) ", m.FilterInput.Value())
		rightPart = m.Theme.BrowserMatch.Render(filterBadge)
	} else {
		sortTag := fmt.Sprintf("Sort [s]: %s", sortLabel)
		themeTag := fmt.Sprintf("Theme [t]: %s", m.Theme.Scheme.Name)
		rightPart = fmt.Sprintf(" %s • %s ", m.Theme.Muted.Render(sortTag), m.Theme.Muted.Render(themeTag))
	}

	leftLen := len(StripANSI(leftPart))
	rightLen := len(StripANSI(rightPart))
	spacesLine0 := m.Width - leftLen - rightLen
	if spacesLine0 < 1 {
		spacesLine0 = 1
	}
	headerLine := leftPart + strings.Repeat(" ", spacesLine0) + rightPart
	headerRendered := m.Theme.Base.Width(m.Width).MaxHeight(1).Render(headerLine)

	// ---------------------------------------------------------
	// 2. VERTICAL CARD FEED BODY (Strictly bodyHeight lines)
	// ---------------------------------------------------------
	var bodyRows []string

	if len(m.DisplayedFiles) == 0 {
		msg := "No markdown files match filter"
		if m.FilterInput.Value() != "" {
			msg = fmt.Sprintf("No markdown files match filter %q", m.FilterInput.Value())
		}
		topPadding := bodyHeight / 2
		for i := 0; i < topPadding; i++ {
			bodyRows = append(bodyRows, strings.Repeat(" ", m.Width))
		}
		centered := fmt.Sprintf("%*s", (m.Width+len(msg))/2, msg)
		bodyRows = append(bodyRows, m.Theme.Muted.Render(centered))
	} else {
		visibleCards := bodyHeight / 3
		if visibleCards < 1 {
			visibleCards = 1
		}

		endIdx := m.ScrollOffset + visibleCards
		if endIdx > len(m.DisplayedFiles) {
			endIdx = len(m.DisplayedFiles)
		}

		for i := m.ScrollOffset; i < endIdx; i++ {
			f := m.DisplayedFiles[i]
			isSelected := (i == m.Cursor)

			var t time.Time
			if m.SortMode == "created" {
				t = f.CreatedAt
			} else {
				t = f.ModifiedAt
			}
			timeStr := formatRelativeTime(t)

			timeColW := len(timeStr) + 2
			availNameW := m.Width - timeColW - 6
			if availNameW < 12 {
				availNameW = 12
			}
			fileName := truncateString(f.Name, availNameW)

			dirDisplay := f.Directory
			if dirDisplay == "" || dirDisplay == "." {
				dirDisplay = "."
			}
			availDirW := m.Width - 6
			if availDirW < 12 {
				availDirW = 12
			}
			dirText := truncateString(dirDisplay, availDirW)

			if isSelected {
				// Line 1: Accent bar + Filename + Timestamp
				line1Raw := fmt.Sprintf(" ▍ %-*s %*s ", availNameW, fileName, timeColW-2, timeStr)
				line1Pad := padToWidth(line1Raw, m.Width)
				bodyRows = append(bodyRows, m.Theme.RowSelected.Bold(true).Render(line1Pad))

				// Line 2: Accent bar + Muted Directory subtext
				line2Raw := fmt.Sprintf(" ▍ %s", dirText)
				line2Pad := padToWidth(line2Raw, m.Width)
				bodyRows = append(bodyRows, m.Theme.RowSelected.Render(line2Pad))

				// Line 3: Subtle separator
				sep := "   " + strings.Repeat("─", m.Width-6)
				bodyRows = append(bodyRows, m.Theme.Divider.Render(sep))
			} else {
				// Line 1: Normal Filename + Muted Timestamp
				line1File := fmt.Sprintf("   %-*s", availNameW, fileName)
				line1Date := fmt.Sprintf(" %*s ", timeColW-2, timeStr)
				row1 := m.Theme.RowNormal.Render(line1File) + m.Theme.ColDate.Render(line1Date)
				bodyRows = append(bodyRows, padToWidth(row1, m.Width))

				// Line 2: Muted Directory subtext
				line2Dir := fmt.Sprintf("   %s", dirText)
				bodyRows = append(bodyRows, m.Theme.ColDir.Render(padToWidth(line2Dir, m.Width)))

				// Line 3: Subtle separator
				sep := "   " + strings.Repeat("─", m.Width-6)
				bodyRows = append(bodyRows, m.Theme.Divider.Render(sep))
			}
		}
	}

	// Pad middle body container strictly to bodyHeight lines
	for len(bodyRows) < bodyHeight {
		bodyRows = append(bodyRows, strings.Repeat(" ", m.Width))
	}

	// ---------------------------------------------------------
	// 3. BOTTOM PINNED COMMAND BAR (Line m.Height - 1)
	// ---------------------------------------------------------
	kOpen := m.Theme.KeyBadge.Render("↵") + " open"
	kFilter := m.Theme.KeyBadge.Render("/") + " filter"
	kSort := m.Theme.KeyBadge.Render("s") + " sort"
	kTheme := m.Theme.KeyBadge.Render("t") + " theme"
	kHelp := m.Theme.KeyBadge.Render("?") + " help"
	kNav := m.Theme.KeyBadge.Render("↑/↓") + " nav"
	kQuit := m.Theme.KeyBadge.Render("q") + " quit"

	commandsLeft := fmt.Sprintf(" %s  %s  %s  %s  %s  %s  %s ", kOpen, kFilter, kSort, kTheme, kHelp, kNav, kQuit)

	posIdx := 0
	totalCount := len(m.DisplayedFiles)
	if totalCount > 0 {
		posIdx = m.Cursor + 1
	}
	commandsRight := fmt.Sprintf(" %s │ %d/%d ", m.Theme.Scheme.Name, posIdx, totalCount)
	rightRendered := m.Theme.StatusBar.Render(commandsRight)

	cLeftLen := len(StripANSI(commandsLeft))
	cRightLen := len(StripANSI(rightRendered))
	footerSpaces := m.Width - cLeftLen - cRightLen
	if footerSpaces < 1 {
		footerSpaces = 1
	}
	footerLine := commandsLeft + strings.Repeat(" ", footerSpaces) + rightRendered
	footerRendered := m.Theme.StatusBar.Width(m.Width).MaxHeight(1).Render(footerLine)

	// Combine into exactly m.Height lines
	allOutputLines := []string{headerRendered}
	allOutputLines = append(allOutputLines, bodyRows[:bodyHeight]...)
	allOutputLines = append(allOutputLines, footerRendered)

	return strings.Join(allOutputLines, "\n")
}

func padToWidth(s string, targetWidth int) string {
	plainLen := len([]rune(StripANSI(s)))
	if plainLen >= targetWidth {
		return s
	}
	return s + strings.Repeat(" ", targetWidth-plainLen)
}

func truncateString(s string, maxLen int) string {
	if maxLen <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= maxLen {
		return s
	}
	if maxLen <= 1 {
		return "…"
	}
	return string(runes[:maxLen-1]) + "…"
}

func formatRelativeTime(t time.Time) string {
	now := time.Now()
	diff := now.Sub(t)
	if diff < 0 {
		return "just now"
	}
	if diff < time.Minute {
		return "just now"
	}
	if diff < time.Hour {
		mins := int(diff.Minutes())
		return fmt.Sprintf("%dm ago", mins)
	}
	if diff < 24*time.Hour {
		hours := int(diff.Hours())
		return fmt.Sprintf("%dh ago", hours)
	}
	if diff < 30*24*time.Hour {
		days := int(diff.Hours() / 24)
		return fmt.Sprintf("%dd ago", days)
	}
	return t.Format("2006-01-02")
}
