package ui

import (
	"fmt"
	"strconv"
	"strings"
	"unicode/utf8"

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
	lines = append(lines, row("b / Esc / Backspace", "Back to browser / Clear search"))
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

// Overlay draws modal centered over baseView with a translucent dimmed backdrop
// that keeps underlying content subtly visible around and behind the dialog.
func Overlay(baseView, modalView string, width, height int, bgHex ...string) string {
	baseLines := strings.Split(baseView, "\n")
	modalLines := strings.Split(modalView, "\n")
	modalHeight := len(modalLines)
	if modalHeight == 0 || height <= 0 || width <= 0 {
		return baseView
	}

	topRow := (height - modalHeight) / 2
	if topRow < 0 {
		topRow = 0
	}

	bg := "#121417"
	if len(bgHex) > 0 && bgHex[0] != "" {
		bg = bgHex[0]
	}
	scrim := computeBackdropScrim(bg)

	result := make([]string, len(baseLines))
	for r, baseLine := range baseLines {
		if r < topRow || r >= topRow+modalHeight {
			seg := ansiSliceColumns(baseLine, 0, width)
			result[r] = dimBackdropSegment(seg, scrim)
			continue
		}

		modalLine := modalLines[r-topRow]
		modalW := lipgloss.Width(modalLine)
		leftPad := (width - modalW) / 2
		if leftPad < 0 {
			leftPad = 0
		}
		rightStart := leftPad + modalW
		if rightStart > width {
			rightStart = width
		}

		leftBg := dimBackdropSegment(ansiSliceColumns(baseLine, 0, leftPad), scrim)
		rightBg := dimBackdropSegment(ansiSliceColumns(baseLine, rightStart, width), scrim)
		result[r] = leftBg + modalLine + rightBg
	}

	return strings.Join(result, "\n")
}

type backdropScrim struct {
	scrimR, scrimG, scrimB uint8
	defFgR, defFgG, defFgB uint8
	fgAlpha                float64
	bgAlpha                float64
}

func computeBackdropScrim(bgHex string) backdropScrim {
	r, g, b, ok := render.HexToRGB(bgHex)
	if !ok {
		r, g, b = 18, 20, 23
	}
	// Perceived luminance to pick a dark or frosted-light scrim
	lum := 0.299*float64(r) + 0.587*float64(g) + 0.114*float64(b)
	if lum >= 128 {
		// Frosted cool-slate mist over light themes
		sr, sg, sb := blendRGB(r, g, b, 148, 163, 184, 0.28)
		return backdropScrim{
			scrimR: sr, scrimG: sg, scrimB: sb,
			defFgR: 30, defFgG: 41, defFgB: 59,
			fgAlpha: 0.34,
			bgAlpha: 0.45,
		}
	}
	// Deepened twilight veil over dark themes
	sr, sg, sb := blendRGB(r, g, b, 0, 0, 0, 0.48)
	return backdropScrim{
		scrimR: sr, scrimG: sg, scrimB: sb,
		defFgR: 226, defFgG: 232, defFgB: 240,
		fgAlpha: 0.30,
		bgAlpha: 0.45,
	}
}

func blendRGB(bgR, bgG, bgB, fgR, fgG, fgB uint8, alpha float64) (uint8, uint8, uint8) {
	inv := 1.0 - alpha
	return uint8(float64(bgR)*inv + float64(fgR)*alpha + 0.5),
		uint8(float64(bgG)*inv + float64(fgG)*alpha + 0.5),
		uint8(float64(bgB)*inv + float64(fgB)*alpha + 0.5)
}

// dimBackdropSegment rewrites SGR codes in an ANSI segment so foreground and background
// colors blend smoothly into the translucent modal scrim while dropping bold/underline focus.
func dimBackdropSegment(seg string, s backdropScrim) string {
	if seg == "" {
		return ""
	}

	curFgR, curFgG, curFgB := s.defFgR, s.defFgG, s.defFgB
	curBgR, curBgG, curBgB := s.scrimR, s.scrimG, s.scrimB

	emitStyle := func() string {
		fr, fg, fb := blendRGB(s.scrimR, s.scrimG, s.scrimB, curFgR, curFgG, curFgB, s.fgAlpha)
		br, bg, bb := blendRGB(s.scrimR, s.scrimG, s.scrimB, curBgR, curBgG, curBgB, s.bgAlpha)
		return fmt.Sprintf("\x1b[0;38;2;%d;%d;%d;48;2;%d;%d;%dm", fr, fg, fb, br, bg, bb)
	}

	var sb strings.Builder
	sb.Grow(len(seg) + 64)
	sb.WriteString(emitStyle())

	locs := ansiPattern.FindAllStringIndex(seg, -1)
	last := 0
	for _, loc := range locs {
		if loc[0] > last {
			sb.WriteString(seg[last:loc[0]])
		}
		seq := seg[loc[0]:loc[1]]
		if strings.HasSuffix(seq, "m") && strings.HasPrefix(seq, "\x1b[") {
			paramsStr := seq[2 : len(seq)-1]
			curFgR, curFgG, curFgB, curBgR, curBgG, curBgB = applySGRColors(
				paramsStr,
				curFgR, curFgG, curFgB,
				curBgR, curBgG, curBgB,
				s,
			)
			sb.WriteString(emitStyle())
		}
		last = loc[1]
	}
	if last < len(seg) {
		sb.WriteString(seg[last:])
	}
	sb.WriteString("\x1b[0m")
	return sb.String()
}

func applySGRColors(paramsStr string, fgR, fgG, fgB, bgR, bgG, bgB uint8, s backdropScrim) (uint8, uint8, uint8, uint8, uint8, uint8) {
	if paramsStr == "" {
		return s.defFgR, s.defFgG, s.defFgB, s.scrimR, s.scrimG, s.scrimB
	}
	parts := strings.Split(paramsStr, ";")
	nums := make([]int, len(parts))
	for i, p := range parts {
		if p == "" {
			nums[i] = 0
		} else if v, err := strconv.Atoi(p); err == nil {
			nums[i] = v
		}
	}

	for i := 0; i < len(nums); i++ {
		switch nums[i] {
		case 0:
			fgR, fgG, fgB = s.defFgR, s.defFgG, s.defFgB
			bgR, bgG, bgB = s.scrimR, s.scrimG, s.scrimB
		case 39:
			fgR, fgG, fgB = s.defFgR, s.defFgG, s.defFgB
		case 49:
			bgR, bgG, bgB = s.scrimR, s.scrimG, s.scrimB
		case 38:
			if i+4 < len(nums) && nums[i+1] == 2 {
				fgR, fgG, fgB = uint8(nums[i+2]), uint8(nums[i+3]), uint8(nums[i+4])
				i += 4
			}
		case 48:
			if i+4 < len(nums) && nums[i+1] == 2 {
				bgR, bgG, bgB = uint8(nums[i+2]), uint8(nums[i+3]), uint8(nums[i+4])
				i += 4
			}
		}
	}
	return fgR, fgG, fgB, bgR, bgG, bgB
}

// ansiSliceColumns extracts display columns [startCol, endCol) from an ANSI line,
// preserving active SGR styles and padding with spaces if the line ends early.
func ansiSliceColumns(line string, startCol, endCol int) string {
	targetWidth := endCol - startCol
	if targetWidth <= 0 {
		return ""
	}

	var activeStyle strings.Builder
	var out strings.Builder
	col := 0
	started := false

	locs := ansiPattern.FindAllStringIndex(line, -1)
	locIdx := 0
	i := 0
	for i < len(line) {
		if locIdx < len(locs) && i == locs[locIdx][0] {
			seq := line[locs[locIdx][0]:locs[locIdx][1]]
			if seq == "\x1b[0m" || seq == "\x1b[m" {
				activeStyle.Reset()
			} else {
				activeStyle.WriteString(seq)
			}
			if started && col < endCol {
				out.WriteString(seq)
			}
			i = locs[locIdx][1]
			locIdx++
			continue
		}

		r, size := utf8.DecodeRuneInString(line[i:])
		rw := lipgloss.Width(string(r))

		if col+rw <= startCol {
			col += rw
			i += size
			continue
		}
		if col >= endCol {
			break
		}

		if !started {
			started = true
			if activeStyle.Len() > 0 {
				out.WriteString(activeStyle.String())
			}
		}

		if col < startCol {
			// Wide rune straddling startCol: replace visible portion with space
			overlap := col + rw - startCol
			if overlap > targetWidth {
				overlap = targetWidth
			}
			out.WriteString(strings.Repeat(" ", overlap))
			col += rw
			i += size
			continue
		}

		if col+rw > endCol {
			// Wide rune straddling endCol: pad remaining column with space
			rem := endCol - col
			out.WriteString(strings.Repeat(" ", rem))
			col = endCol
			break
		}

		out.WriteRune(r)
		col += rw
		i += size
	}

	emitted := lipgloss.Width(out.String())
	if emitted < targetWidth {
		out.WriteString(strings.Repeat(" ", targetWidth-emitted))
	}
	if activeStyle.Len() > 0 || started {
		out.WriteString("\x1b[0m")
	}
	return out.String()
}
