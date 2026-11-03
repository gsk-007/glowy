package ui

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
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

type folderRow struct {
	isHeader bool
	dirName  string
	count    int
	file     *discovery.MarkdownFile
}

// BrowserModel represents the TUI File Browser state.
type BrowserModel struct {
	AllFiles       []discovery.MarkdownFile
	DisplayedFiles []discovery.MarkdownFile
	FolderRows     []folderRow
	FolderView     bool
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

// NewBrowserModel initializes a new File Browser model.
func NewBrowserModel(files []discovery.MarkdownFile, theme *config.UITheme, defaultSort string) BrowserModel {
	if defaultSort == "" {
		defaultSort = "modified"
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter files..."
	ti.Prompt = "❯ "
	ti.CharLimit = 64

	cwd, err := os.Getwd()
	if err != nil {
		cwd = "."
	}
	cwd = filepath.ToSlash(cwd)

	m := BrowserModel{
		AllFiles:     files,
		FolderView:   false,
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

	// Build folder rows
	var fRows []folderRow
	groups := discovery.GroupByDirectory(candidates)
	var dirs []string
	for dir := range groups {
		dirs = append(dirs, dir)
	}
	sort.Strings(dirs)

	for _, dir := range dirs {
		filesInDir := groups[dir]
		discovery.SortFiles(filesInDir, m.SortMode)
		fRows = append(fRows, folderRow{
			isHeader: true,
			dirName:  dir,
			count:    len(filesInDir),
		})
		for i := range filesInDir {
			fRows = append(fRows, folderRow{
				isHeader: false,
				dirName:  dir,
				file:     &filesInDir[i],
			})
		}
	}
	m.FolderRows = fRows

	totalItems := m.itemCount()
	if m.Cursor >= totalItems {
		m.Cursor = totalItems - 1
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}

	// In folder view, ensure cursor does not rest on a header if possible
	if m.FolderView && len(m.FolderRows) > 0 && m.FolderRows[m.Cursor].isHeader {
		m.advanceCursor(1)
	}

	m.adjustScroll()
}

func (m *BrowserModel) itemCount() int {
	if m.FolderView {
		return len(m.FolderRows)
	}
	return len(m.DisplayedFiles)
}

func (m *BrowserModel) advanceCursor(step int) {
	total := m.itemCount()
	if total == 0 {
		m.Cursor = 0
		return
	}

	newCursor := m.Cursor + step
	if !m.FolderView {
		if newCursor < 0 {
			newCursor = 0
		}
		if newCursor >= total {
			newCursor = total - 1
		}
		m.Cursor = newCursor
		m.adjustScroll()
		return
	}

	// In Folder View, skip headers if moving
	for newCursor >= 0 && newCursor < total {
		if !m.FolderRows[newCursor].isHeader {
			m.Cursor = newCursor
			m.adjustScroll()
			return
		}
		newCursor += step
	}
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
	bodyHeight := m.Height - 4 // Top bar (2) + Column header (1) + Bottom bar (1)
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

		case "1":
			m.FolderView = false
			m.applyFilterAndSort()

		case "2":
			m.FolderView = true
			m.applyFilterAndSort()

		case "tab", "v":
			m.FolderView = !m.FolderView
			m.applyFilterAndSort()

		case "s":
			m.cycleSort()

		case "up", "k":
			m.advanceCursor(-1)

		case "down", "j":
			m.advanceCursor(1)

		case "pgup", "u":
			pageSize := (m.Height - 4) / 2
			if pageSize < 1 {
				pageSize = 5
			}
			m.advanceCursor(-pageSize)

		case "pgdown", "d":
			pageSize := (m.Height - 4) / 2
			if pageSize < 1 {
				pageSize = 5
			}
			m.advanceCursor(pageSize)

		case "home", "g":
			m.Cursor = 0
			if m.FolderView && len(m.FolderRows) > 0 && m.FolderRows[0].isHeader {
				m.advanceCursor(1)
			} else {
				m.adjustScroll()
			}

		case "end", "G":
			total := m.itemCount()
			if total > 0 {
				m.Cursor = total - 1
				if m.FolderView && m.FolderRows[m.Cursor].isHeader {
					m.advanceCursor(-1)
				} else {
					m.adjustScroll()
				}
			}

		case "enter":
			if !m.FolderView {
				if len(m.DisplayedFiles) > 0 && m.Cursor < len(m.DisplayedFiles) {
					selected := m.DisplayedFiles[m.Cursor]
					m.SelectedFile = &selected
					return m, nil
				}
			} else {
				if len(m.FolderRows) > 0 && m.Cursor < len(m.FolderRows) {
					row := m.FolderRows[m.Cursor]
					if !row.isHeader && row.file != nil {
						m.SelectedFile = row.file
						return m, nil
					}
				}
			}
		}
	}

	return m, nil
}

func (m BrowserModel) View() string {
	if m.Width == 0 || m.Height == 0 {
		return ""
	}

	// Calculate strict layout heights
	headerLinesCount := 3
	footerLinesCount := 1
	bodyHeight := m.Height - headerLinesCount - footerLinesCount
	if bodyHeight < 1 {
		bodyHeight = 1
	}

	// ---------------------------------------------------------
	// 1. TOP SEGMENTED UTILITY BAR (Lines 0 and 1)
	// ---------------------------------------------------------

	// Line 0: Badge + CWD + File count
	badge := m.Theme.Badge.Render(" GLOWY ")
	cwdText := " 📁 " + m.Cwd
	leftPart := badge + m.Theme.Muted.Render(cwdText)

	fileCountStr := fmt.Sprintf("%d markdown files ", len(m.DisplayedFiles))
	rightPart := m.Theme.Muted.Render(fileCountStr)

	leftLen := len(StripANSI(leftPart))
	rightLen := len(StripANSI(rightPart))
	spacesLine0 := m.Width - leftLen - rightLen
	if spacesLine0 < 1 {
		spacesLine0 = 1
	}
	line0 := leftPart + strings.Repeat(" ", spacesLine0) + rightPart
	line0Rendered := m.Theme.Base.Width(m.Width).MaxHeight(1).Render(line0)

	// Line 1: Mode Pills + Sort Indicator / Filter prompt
	var tabFiles, tabFolders string
	if !m.FolderView {
		tabFiles = m.Theme.TabActive.Render(" [ 1: FILES ] ")
		tabFolders = m.Theme.TabInactive.Render(" [ 2: FOLDERS ] ")
	} else {
		tabFiles = m.Theme.TabInactive.Render(" [ 1: FILES ] ")
		tabFolders = m.Theme.TabActive.Render(" [ 2: FOLDERS ] ")
	}
	tabs := " " + tabFiles + tabFolders

	var rightControl string
	if m.Filtering {
		rightControl = m.Theme.BrowserFilter.Render(" FILTER ❯ ") + m.FilterInput.View() + " "
	} else if m.FilterInput.Value() != "" {
		rightControl = fmt.Sprintf(" Filter: %s (press / to edit, esc to clear) ", m.FilterInput.Value())
		rightControl = m.Theme.BrowserMatch.Render(rightControl)
	} else {
		sortLabel := "Modified ↓"
		switch m.SortMode {
		case "name":
			sortLabel = "Name A-Z"
		case "created":
			sortLabel = "Created ↓"
		}
		rightControl = fmt.Sprintf(" Sort [s]: %s ", sortLabel)
		rightControl = m.Theme.StatusMode.Render(rightControl)
	}

	tabLen := len(StripANSI(tabs))
	rcLen := len(StripANSI(rightControl))
	spacesLine1 := m.Width - tabLen - rcLen
	if spacesLine1 < 1 {
		spacesLine1 = 1
	}
	line1 := tabs + strings.Repeat(" ", spacesLine1) + rightControl
	line1Rendered := m.Theme.Base.Width(m.Width).MaxHeight(1).Render(line1)

	// ---------------------------------------------------------
	// 2. COLUMN HEADER (Line 2)
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

	var colHeaderStr string
	if !m.FolderView {
		colHeaderStr = fmt.Sprintf("   %-*s %-*s %*s", col1W-3, "FILE", col2W-1, "DIRECTORY", col3W-1, "UPDATED")
	} else {
		colHeaderStr = fmt.Sprintf("   %-*s %*s", m.Width-col3W-4, "FOLDER / FILE", col3W-1, "UPDATED")
	}
	line2Rendered := m.Theme.TableHeader.Width(m.Width).MaxHeight(1).Render(colHeaderStr)

	// ---------------------------------------------------------
	// 3. MIDDLE BODY CONTAINER (Strictly bodyHeight lines)
	// ---------------------------------------------------------
	var bodyRows []string

	if !m.FolderView {
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

				var rawRow string
				if isSelected {
					selPrefix := " ▍ "
					rawRow = fmt.Sprintf("%s%-*s %-*s %*s ", selPrefix, col1W-4, fileName, col2W-2, dirName, col3W-2, timeStr)
					// Pad or crop to exact m.Width
					rawRow = padToWidth(rawRow, m.Width)
					bodyRows = append(bodyRows, m.Theme.RowSelected.Render(rawRow))
				} else {
					prefix := "   "
					filePart := fmt.Sprintf("%s%-*s", prefix, col1W-4, fileName)
					dirPart := fmt.Sprintf(" %-*s", col2W-2, dirName)
					datePart := fmt.Sprintf(" %*s ", col3W-2, timeStr)

					fileRend := m.Theme.RowNormal.Render(filePart)
					dirRend := m.Theme.ColDir.Render(dirPart)
					dateRend := m.Theme.ColDate.Render(datePart)
					lineFull := padToWidth(fileRend+dirRend+dateRend, m.Width)
					bodyRows = append(bodyRows, lineFull)
				}
			}
		}
	} else {
		// Folder View
		if len(m.FolderRows) == 0 {
			msg := "No markdown files found."
			topPadding := bodyHeight / 2
			for i := 0; i < topPadding; i++ {
				bodyRows = append(bodyRows, "")
			}
			centered := fmt.Sprintf("%*s", (m.Width+len(msg))/2, msg)
			bodyRows = append(bodyRows, m.Theme.Muted.Render(centered))
		} else {
			endIdx := m.ScrollOffset + bodyHeight
			if endIdx > len(m.FolderRows) {
				endIdx = len(m.FolderRows)
			}

			for i := m.ScrollOffset; i < endIdx; i++ {
				row := m.FolderRows[i]
				if row.isHeader {
					hdr := fmt.Sprintf(" ├─ 📁 %s (%d)", row.dirName, row.count)
					hdrPadded := padToWidth(hdr, m.Width)
					bodyRows = append(bodyRows, m.Theme.FolderHeader.Render(hdrPadded))
				} else {
					isSelected := (i == m.Cursor)
					f := row.file

					var t time.Time
					if m.SortMode == "created" {
						t = f.CreatedAt
					} else {
						t = f.ModifiedAt
					}
					timeStr := formatRelativeTime(t)

					availW := m.Width - col3W - 7
					if availW < 10 {
						availW = 10
					}
					fileName := truncateString(f.Name, availW)

					if isSelected {
						selRow := fmt.Sprintf(" │  ▍ %-*s %*s ", availW, fileName, col3W-1, timeStr)
						bodyRows = append(bodyRows, m.Theme.RowSelected.Render(padToWidth(selRow, m.Width)))
					} else {
						filePart := fmt.Sprintf(" │    %-*s", availW, fileName)
						datePart := fmt.Sprintf(" %*s ", col3W-1, timeStr)
						lineFull := m.Theme.RowNormal.Render(filePart) + m.Theme.ColDate.Render(datePart)
						bodyRows = append(bodyRows, padToWidth(lineFull, m.Width))
					}
				}
			}
		}
	}

	// Pad middle body container to strictly bodyHeight lines
	for len(bodyRows) < bodyHeight {
		bodyRows = append(bodyRows, strings.Repeat(" ", m.Width))
	}

	// ---------------------------------------------------------
	// 4. BOTTOM COMMAND BAR (Line m.Height - 1)
	// ---------------------------------------------------------
	// Key badges
	kOpen := m.Theme.KeyBadge.Render("↵") + " open"
	kFilter := m.Theme.KeyBadge.Render("/") + " filter"
	kTab := m.Theme.KeyBadge.Render("tab") + " view"
	kNums := m.Theme.KeyBadge.Render("1/2") + " mode"
	kSort := m.Theme.KeyBadge.Render("s") + " sort"
	kNav := m.Theme.KeyBadge.Render("↑/↓") + " nav"
	kQuit := m.Theme.KeyBadge.Render("q") + " quit"

	commandsLeft := fmt.Sprintf(" %s  %s  %s  %s  %s  %s  %s ", kOpen, kFilter, kTab, kNums, kSort, kNav, kQuit)

	posIdx := 0
	totalCount := len(m.DisplayedFiles)
	if !m.FolderView {
		if totalCount > 0 {
			posIdx = m.Cursor + 1
		}
	} else {
		// Map cursor to file position
		if len(m.FolderRows) > 0 && m.Cursor < len(m.FolderRows) {
			fileCount := 0
			for idx := 0; idx <= m.Cursor; idx++ {
				if !m.FolderRows[idx].isHeader {
					fileCount++
				}
			}
			posIdx = fileCount
		}
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
	allOutputLines := []string{line0Rendered, line1Rendered, line2Rendered}
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
		if days == 1 {
			return "1d ago"
		}
		return fmt.Sprintf("%dd ago", days)
	}
	return t.Format("2006-01-02")
}
