package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/glowy/cli/internal/config"
)

// ThemeModal manages in-TUI live theme selection and persistence.
type ThemeModal struct {
	Schemes        []config.Scheme
	Cursor         int
	OriginalScheme string
	Width          int
}

// NewThemeModal initializes the ThemeModal with all available schemes from config.
func NewThemeModal(schemes []config.Scheme, activeScheme string) ThemeModal {
	cursor := 0
	for i, s := range schemes {
		if strings.EqualFold(s.Name, activeScheme) {
			cursor = i
			break
		}
	}

	return ThemeModal{
		Schemes:        schemes,
		Cursor:         cursor,
		OriginalScheme: activeScheme,
	}
}

// SelectedScheme returns the scheme currently under the cursor.
func (m *ThemeModal) SelectedScheme() config.Scheme {
	if len(m.Schemes) == 0 {
		return config.DefaultSlateDark()
	}
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= len(m.Schemes) {
		m.Cursor = len(m.Schemes) - 1
	}
	return m.Schemes[m.Cursor]
}

// MoveCursor adjusts cursor by delta and returns the new preview scheme.
func (m *ThemeModal) MoveCursor(delta int) config.Scheme {
	total := len(m.Schemes)
	if total == 0 {
		return config.DefaultSlateDark()
	}
	m.Cursor += delta
	if m.Cursor < 0 {
		m.Cursor = 0
	}
	if m.Cursor >= total {
		m.Cursor = total - 1
	}
	return m.SelectedScheme()
}

// Render draws the Theme Picker modal box.
func (m *ThemeModal) Render(theme *config.UITheme, screenWidth int) string {
	boxW := 48
	if screenWidth < 54 {
		boxW = screenWidth - 6
	}
	if boxW < 24 {
		boxW = 24
	}

	title := theme.Selected.Render(" COLOR SCHEMES ")

	var rows []string
	rows = append(rows, "")

	for i, s := range m.Schemes {
		isSelected := (i == m.Cursor)
		isActive := strings.EqualFold(s.Name, m.OriginalScheme)

		swatchBg := lipgloss.NewStyle().Foreground(lipgloss.Color(s.Background)).Render("██")
		swatchFg := lipgloss.NewStyle().Foreground(lipgloss.Color(s.Foreground)).Render("██")
		swatchAccent := lipgloss.NewStyle().Foreground(lipgloss.Color(s.CursorAccent)).Render("██")
		swatches := fmt.Sprintf("%s%s%s", swatchBg, swatchFg, swatchAccent)

		activeTag := ""
		if isActive {
			activeTag = theme.Muted.Render(" (active)")
		}

		if isSelected {
			line := fmt.Sprintf(" ▍ %-14s %s%s", s.Name, swatches, activeTag)
			rows = append(rows, theme.RowSelected.Render(line))
		} else {
			line := fmt.Sprintf("   %-14s %s%s", s.Name, swatches, activeTag)
			rows = append(rows, theme.RowNormal.Render(line))
		}
	}

	rows = append(rows, "")
	rows = append(rows, " "+theme.Muted.Render("↑/↓ browse • Enter save • Esc cancel"))

	modalBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Scheme.Border)).
		Background(lipgloss.Color(theme.Scheme.Background)).
		Padding(0, 1).
		Width(boxW)

	return title + "\n" + modalBox.Render(strings.Join(rows, "\n"))
}
