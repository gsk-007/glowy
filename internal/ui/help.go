package ui

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/render"
)

// RenderHelpModal generates a bordered, formatted Help Menu modal.
func RenderHelpModal(theme *config.UITheme, width int) string {
	boxW := 64
	if width < 70 {
		boxW = width - 6
	}
	if boxW < 30 {
		boxW = 30
	}

	title := theme.Selected.Render(" KEYBOARD SHORTCUTS ")
	secStyle := theme.FolderHeader
	keyStyle := theme.Cursor
	descStyle := theme.RowNormal

	row := func(keys, desc string) string {
		return "  " + keyStyle.Render(keys) + strings.Repeat(" ", 22-len(keys)) + descStyle.Render(desc)
	}

	var lines []string
	lines = append(lines, "")
	lines = append(lines, " "+secStyle.Render("NAVIGATION"))
	lines = append(lines, row("↑/↓, k/j", "Scroll 1 line / Move cursor"))
	lines = append(lines, row("u/d, PgUp/PgDn", "Half page up / down"))
	lines = append(lines, row("g/G, Home/End", "Jump to top / bottom"))
	lines = append(lines, "")
	lines = append(lines, " "+secStyle.Render("FILE BROWSER"))
	lines = append(lines, row("Enter", "Open selected Markdown document"))
	lines = append(lines, row("/", "Fuzzy search by filename or path"))
	lines = append(lines, row("s", "Cycle sort: Modified • Name • Created"))
	lines = append(lines, "")
	lines = append(lines, " "+secStyle.Render("DOCUMENT PAGER"))
	lines = append(lines, row("/", "In-document search"))
	lines = append(lines, row("n / N", "Jump to next / previous match"))
	lines = append(lines, row("r", "Toggle Formatted vs Raw Markdown"))
	lines = append(lines, row("l", "Toggle raw Line Numbers gutter"))
	lines = append(lines, "")
	lines = append(lines, " "+secStyle.Render("GLOBAL"))
	lines = append(lines, row("t", "Switch Color Theme (Live Preview)"))
	lines = append(lines, row("?", "Toggle this Help Menu"))
	lines = append(lines, row("Esc / Backspace", "Back to browser / Clear search"))
	lines = append(lines, row("q", "Quit application"))
	lines = append(lines, "")
	lines = append(lines, "  "+theme.Muted.Render("[ Press ? or Esc to close ]"))

	content := strings.Join(lines, "\n")

	modalBox := lipgloss.NewStyle().
		BorderStyle(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(theme.Scheme.Border)).
		Background(lipgloss.Color(theme.Scheme.Background)).
		Padding(0, 1).
		Width(boxW)

	return title + "\n" + modalBox.Render(content)
}

// Overlay draws modal centered over baseView while strictly preserving line count.
func Overlay(baseView, modalView string, width, height int, bgHex ...string) string {
	baseLines := strings.Split(baseView, "\n")
	modalLines := strings.Split(modalView, "\n")
	modalHeight := len(modalLines)
	if modalHeight == 0 || height <= 0 {
		return baseView
	}

	topRow := (height - modalHeight) / 2
	if topRow < 0 {
		topRow = 0
	}

	result := make([]string, len(baseLines))
	copy(result, baseLines)

	bg := ""
	if len(bgHex) > 0 {
		bg = bgHex[0]
	}

	for i, modalLine := range modalLines {
		targetRow := topRow + i
		if targetRow >= len(result) {
			break
		}
		modalW := lipgloss.Width(modalLine)
		leftPad := (width - modalW) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightPad := width - leftPad - modalW
		if rightPad < 0 {
			rightPad = 0
		}

		line := strings.Repeat(" ", leftPad) + modalLine + strings.Repeat(" ", rightPad)
		if bg != "" {
			line = render.ApplyLineBackground(line, width, bg)
		}
		result[targetRow] = line
	}

	return strings.Join(result, "\n")
}
