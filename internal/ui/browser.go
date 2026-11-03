package ui

import (
	"fmt"
	"os"
	"path/filepath"
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

// BrowserModel represents the simplified, unified TUI File Browser state.
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
	Cwd          string
}

// NewBrowserModel initializes a unified File Browser model.
func NewBrowserModel(files []discovery.MarkdownFile, theme *config.UITheme, defaultSort string) BrowserModel {
	if defaultSort == "" {
		defaultSort = "modified"
	}

	ti := textinput.New()
	ti.Placeholder = "Type to search files by name or path..."
	ti.Prompt = "❯ "
	ti.CharLimit = 64

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cwd = filepath.ToSlash(cwd)

	m := BrowserModel{
		AllFiles:     files,
		SortMode:     defaultSort,
		Cursor:       0,
		ScrollOffset: 0,
		FilterInput:  ti,
		Theme:        theme,
		Cwd:          cwd,
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
	bodyHeight := m.Height - 3 // Header (2 lines) + Footer (1 line)
	if bodyHeight <= 0 {
		bodyHeight = 10
	}

	if m.Cursor < m.ScrollOffset {
		m.ScrollOffset = m.Cursor
	}
	if m.Cursor >= m.ScrollOffset+bodyHeight {
		m.ScrollOffset = m.Cursor - bodyHeight + 1
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
			pageSize := (m.Height - 3) / 2
			if pageSize < 1 {
				pageSize = 5
			}
			m.Cursor -= pageSize
			if m.Cursor < 0 {
				m.Cursor = 0
			}
			m.adjustScroll()

		case "pgdown", "d":
			pageSize := (m.Height - 3) / 2
			if pageSize < 1 {
				pageSize = 5
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

	// Strict height calculations
	headerLinesCount := 2
	footerLinesCount := 1
	bodyHeight := m.Height - headerLinesCount - footerLinesCount
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// ---------------------------------------------------------
	// 1. TOP SEGMENTED UTILITY BAR (Line 0)
	// ---------------------------------------------------------
	badge := m.Theme.Badge.Render(" GLOWY ")
	cwdText := " 📁 " + m.Cwd
	leftPart := badge + m.Theme.Muted.Render(cwdText)

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
		countTag := fmt.Sprintf("%d files", len(m.DisplayedFiles))
		rightPart = fmt.Sprintf(" %s • %s • %s ", m.Theme.Muted.Render(sortTag), m.Theme.Muted.Render(themeTag), m.Theme.Cursor.Render(countTag))
	}

	leftLen := len(StripANSI(leftPart))
	rightLen := len(StripANSI(rightPart))
	spacesLine0 := m.Width - leftLen - rightLen
	if spacesLine0 < 1 {
		spacesLine0 = 1
	}
	line0 := leftPart + strings.Repeat(" ", spacesLine0) + rightPart
	line0Rendered := m.Theme.Base.Width(m.Width).MaxHeight(1).Render(line0)

	// ---------------------------------------------------------
	// 2. COLUMN HEADER (Line 1)
	// ---------------------------------------------------------
	col1W := int(float64(m.Width) * 0.45)
	if col1W < 24 {
		col1W = 24
	}
	col3W := 14
	col2W := m.Width - col1W - col3W
	if col2W < 12 {
		col2W = 12
	}

	colHeaderStr := fmt.Sprintf("   %-*s %-*s %*s", col1W-3, "FILE", col2W-1, "DIRECTORY", col3W-1, "UPDATED")
	line1Rendered := m.Theme.TableHeader.Width(m.Width).MaxHeight(1).Render(colHeaderStr)

	// ---------------------------------------------------------
	// 3. MIDDLE BODY CONTAINER (Strictly bodyHeight lines)
	// ---------------------------------------------------------
	var bodyRows []string

	if len(m.DisplayedFiles) == 0 {
		msg := "No markdown files match filter"
		if m.FilterInput.Value() != "" {
			msg = fmt.Sprintf("No markdown files match filter %q", m.FilterInput.Value())
		}
		topPadding := bodyHeight / 2
		for i := 0; i < topPadding; i++ {
			bodyRows = append(bodyRows, "")
		}
		centered := fmt.Sprintf("%*s", (m.Width+len(msg))/2, msg)
		bodyRows = append(bodyRows, m.Theme.Muted.Render(centered))
	} else {
		endIdx := m.ScrollOffset + bodyHeight
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

			fileName := truncateString(f.Name, col1W-4)
			dirName := truncateString(f.Directory, col2W-2)

			if isSelected {
				selPrefix := " ▍ "
				rawRow := fmt.Sprintf("%s%-*s %-*s %*s ", selPrefix, col1W-4, fileName, col2W-2, dirName, col3W-2, timeStr)
				bodyRows = append(bodyRows, m.Theme.RowSelected.Render(padToWidth(rawRow, m.Width)))
			} else {
				prefix := "   "
				filePart := fmt.Sprintf("%s%-*s", prefix, col1W-4, fileName)
				dirPart := fmt.Sprintf(" %-*s", col2W-2, dirName)
				datePart := fmt.Sprintf(" %*s ", col3W-2, timeStr)

				fileRend := m.Theme.RowNormal.Render(filePart)
				dirRend := m.Theme.ColDir.Render(dirPart)
				dateRend := m.Theme.ColDate.Render(datePart)
				bodyRows = append(bodyRows, padToWidth(fileRend+dirRend+dateRend, m.Width))
			}
		}
	}

	// Pad middle body container strictly to bodyHeight lines
	for len(bodyRows) < bodyHeight {
		bodyRows = append(bodyRows, strings.Repeat(" ", m.Width))
	}

	// ---------------------------------------------------------
	// 4. BOTTOM COMMAND BAR (Line m.Height - 1)
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
	allOutputLines := []string{line0Rendered, line1Rendered}
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
