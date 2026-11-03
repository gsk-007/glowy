package ui

import (
	"fmt"
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

// BrowserModel represents the TUI File Browser state.
type BrowserModel struct {
	AllFiles       []discovery.MarkdownFile
	DisplayedFiles []discovery.MarkdownFile
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
}

// NewBrowserModel initializes a new File Browser model.
func NewBrowserModel(files []discovery.MarkdownFile, theme *config.UITheme, defaultSort string) BrowserModel {
	if defaultSort == "" {
		defaultSort = "modified"
	}

	ti := textinput.New()
	ti.Placeholder = "Type to filter files..."
	ti.Prompt = "/ "
	ti.CharLimit = 64

	sorted := make([]discovery.MarkdownFile, len(files))
	copy(sorted, files)
	discovery.SortFiles(sorted, defaultSort)

	m := BrowserModel{
		AllFiles:       files,
		DisplayedFiles: sorted,
		FolderView:     false,
		SortMode:       defaultSort,
		Cursor:         0,
		ScrollOffset:   0,
		FilterInput:    ti,
		Theme:          theme,
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

	if !m.FolderView {
		discovery.SortFiles(candidates, m.SortMode)
		m.DisplayedFiles = candidates
	} else {
		// Group by directory
		discovery.SortFiles(candidates, m.SortMode)
		groups := discovery.GroupByDirectory(candidates)

		var dirs []string
		for dir := range groups {
			dirs = append(dirs, dir)
		}
		sort.Strings(dirs)

		var grouped []discovery.MarkdownFile
		for _, dir := range dirs {
			grouped = append(grouped, groups[dir]...)
		}
		m.DisplayedFiles = grouped
	}

	if m.Cursor >= len(m.DisplayedFiles) {
		m.Cursor = len(m.DisplayedFiles) - 1
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
	visibleLines := m.Height - 4 // Header + status lines
	if visibleLines <= 0 {
		visibleLines = 10
	}

	if m.Cursor < m.ScrollOffset {
		m.ScrollOffset = m.Cursor
	}
	if m.Cursor >= m.ScrollOffset+visibleLines {
		m.ScrollOffset = m.Cursor - visibleLines + 1
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

		case "tab", "v":
			m.FolderView = !m.FolderView
			m.applyFilterAndSort()

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
			pageSize := (m.Height - 4) / 2
			if pageSize < 1 {
				pageSize = 5
			}
			m.Cursor -= pageSize
			if m.Cursor < 0 {
				m.Cursor = 0
			}
			m.adjustScroll()

		case "pgdown", "d":
			pageSize := (m.Height - 4) / 2
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
	if m.Width == 0 {
		return ""
	}

	var sb strings.Builder

	// Top Title Bar
	viewName := "Flat View"
	if m.FolderView {
		viewName = "Folder View"
	}
	sortName := "Modified"
	switch m.SortMode {
	case "name":
		sortName = "Name (A-Z)"
	case "created":
		sortName = "Created"
	}

	titleText := fmt.Sprintf(" glowy │ %s │ Sort: %s │ %d files", viewName, sortName, len(m.DisplayedFiles))
	sb.WriteString(m.Theme.BrowserTitle.Width(m.Width).Render(titleText))
	sb.WriteString("\n")

	// Filter bar (if filtering or has filter text)
	if m.Filtering {
		filterLine := m.Theme.BrowserFilter.Render(m.FilterInput.View())
		sb.WriteString(filterLine)
		sb.WriteString("\n")
	} else if m.FilterInput.Value() != "" {
		activeFilter := fmt.Sprintf("Filter: %s (press / to edit, esc to clear)", m.FilterInput.Value())
		sb.WriteString(m.Theme.Muted.Render(activeFilter))
		sb.WriteString("\n")
	}

	// File items
	visibleLines := m.Height - 4
	if m.Filtering || m.FilterInput.Value() != "" {
		visibleLines--
	}
	if visibleLines < 1 {
		visibleLines = 5
	}

	if len(m.DisplayedFiles) == 0 {
		sb.WriteString("\n  " + m.Theme.Muted.Render("No Markdown files found.") + "\n")
	} else {
		endIdx := m.ScrollOffset + visibleLines
		if endIdx > len(m.DisplayedFiles) {
			endIdx = len(m.DisplayedFiles)
		}

		currentDirHeader := ""
		for i := m.ScrollOffset; i < endIdx; i++ {
			f := m.DisplayedFiles[i]

			// If in folder view, display directory header when changing directories
			if m.FolderView && f.Directory != currentDirHeader {
				currentDirHeader = f.Directory
				header := fmt.Sprintf(" [%s]", currentDirHeader)
				sb.WriteString(m.Theme.FolderHeader.Render(header) + "\n")
			}

			isSelected := (i == m.Cursor)
			timeStr := formatRelativeTime(f.ModifiedAt)

			var itemLine string
			if m.FolderView {
				itemLine = fmt.Sprintf("   %-36s  %10s", f.Name, timeStr)
			} else {
				dirPrefix := ""
				if f.Directory != "." {
					dirPrefix = f.Directory + "/"
				}
				itemLine = fmt.Sprintf("  %-40s  %10s", dirPrefix+f.Name, timeStr)
			}

			if isSelected {
				cursorMark := "> "
				rendered := m.Theme.Selected.Render(cursorMark + strings.TrimSpace(itemLine))
				sb.WriteString(rendered + "\n")
			} else {
				sb.WriteString(m.Theme.BrowserItem.Render("  "+strings.TrimSpace(itemLine)) + "\n")
			}
		}
	}

	// Bottom Help Status Bar
	help := " [j/k/↑/↓] Navigate  [Enter] Open  [/] Filter  [s] Sort  [Tab] View  [q] Quit"
	sb.WriteString("\n" + m.Theme.StatusBar.Width(m.Width).Render(help))

	return sb.String()
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
