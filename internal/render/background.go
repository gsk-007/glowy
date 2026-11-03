package render

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var resetRegex = regexp.MustCompile(`\x1b\[(?:0|49)?m`)

// HexToRGB parses a hex color string ("#RRGGBB", "RRGGBB", or "#RGB") into 8-bit R, G, B components.
func HexToRGB(hex string) (r, g, b uint8, ok bool) {
	hex = strings.TrimPrefix(hex, "#")
	if len(hex) == 3 {
		hex = string([]byte{hex[0], hex[0], hex[1], hex[1], hex[2], hex[2]})
	}
	if len(hex) != 6 {
		return 0, 0, 0, false
	}
	val, err := strconv.ParseUint(hex, 16, 32)
	if err != nil {
		return 0, 0, 0, false
	}
	return uint8((val >> 16) & 0xFF), uint8((val >> 8) & 0xFF), uint8(val & 0xFF), true
}

// ApplyLineBackground wraps a single line with the active 24-bit ANSI background sequence.
// It replaces inner full resets (\x1b[0m / \x1b[m / \x1b[49m) with the 24-bit background
// sequence to prevent terminal background bleed-through, pads the visible width of the line
// with spaces up to width while keeping the background active, and resets cleanly at the line end.
func ApplyLineBackground(line string, width int, bgHex string) string {
	r, g, b, ok := HexToRGB(bgHex)
	if !ok {
		return line
	}
	bgSeq := fmt.Sprintf("\x1b[48;2;%d;%d;%dm", r, g, b)

	line = strings.TrimRight(line, "\r")
	visWidth := lipgloss.Width(line)

	pad := 0
	if width > visWidth {
		pad = width - visWidth
	}

	replaced := resetRegex.ReplaceAllStringFunc(line, func(match string) string {
		if match == "\x1b[49m" {
			return bgSeq
		}
		return "\x1b[0m" + bgSeq
	})

	var sb strings.Builder
	sb.Grow(len(bgSeq)*2 + len(replaced) + pad + 8)
	if !strings.HasPrefix(replaced, bgSeq) {
		sb.WriteString(bgSeq)
	}
	sb.WriteString(replaced)
	if pad > 0 {
		sb.WriteString(strings.Repeat(" ", pad))
	}
	sb.WriteString("\x1b[0m")
	return sb.String()
}

// ApplyBackground applies ApplyLineBackground across every line in a multi-line string.
func ApplyBackground(content string, width int, bgHex string) string {
	if content == "" {
		return content
	}
	lines := strings.Split(content, "\n")
	for i, l := range lines {
		lines[i] = ApplyLineBackground(l, width, bgHex)
	}
	return strings.Join(lines, "\n")
}
