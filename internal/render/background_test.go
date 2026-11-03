package render

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
)

func TestHexToRGB(t *testing.T) {
	r, g, b, ok := HexToRGB("#FAFAFA")
	if !ok || r != 250 || g != 250 || b != 250 {
		t.Fatalf("expected 250, 250, 250, got %d, %d, %d (ok=%v)", r, g, b, ok)
	}

	r, g, b, ok = HexToRGB("#121417")
	if !ok || r != 18 || g != 20 || b != 23 {
		t.Fatalf("expected 18, 20, 23, got %d, %d, %d (ok=%v)", r, g, b, ok)
	}

	r, g, b, ok = HexToRGB("FFF")
	if !ok || r != 255 || g != 255 || b != 255 {
		t.Fatalf("expected 255, 255, 255, got %d, %d, %d (ok=%v)", r, g, b, ok)
	}

	_, _, _, ok = HexToRGB("invalid")
	if ok {
		t.Fatal("expected invalid hex to fail")
	}
}

func TestApplyLineBackground(t *testing.T) {
	bgHex := "#FAFAFA"
	expectedBgSeq := "\x1b[48;2;250;250;250m"

	// 1. Plain text line padding
	plain := "Hello World"
	res := ApplyLineBackground(plain, 20, bgHex)
	if !strings.HasPrefix(res, expectedBgSeq) {
		t.Fatalf("expected line to start with bgSeq, got: %q", res)
	}
	if !strings.HasSuffix(res, "\x1b[0m") {
		t.Fatalf("expected line to end with reset \\x1b[0m, got: %q", res)
	}
	if lipgloss.Width(res) != 20 {
		t.Fatalf("expected visible width 20, got %d", lipgloss.Width(res))
	}

	// 2. Inner ANSI reset replacement
	innerReset := "Start \x1b[31mRed\x1b[0m End"
	resInner := ApplyLineBackground(innerReset, 30, bgHex)
	if !strings.Contains(resInner, "\x1b[0m"+expectedBgSeq) {
		t.Fatalf("expected inner reset to be followed by bgSeq, got: %q", resInner)
	}
	if lipgloss.Width(resInner) != 30 {
		t.Fatalf("expected visible width 30, got %d", lipgloss.Width(resInner))
	}

	// 3. Empty line full-width padding
	resEmpty := ApplyLineBackground("", 15, bgHex)
	if !strings.HasPrefix(resEmpty, expectedBgSeq) || !strings.HasSuffix(resEmpty, "\x1b[0m") {
		t.Fatalf("expected empty line to be wrapped with bgSeq and reset, got: %q", resEmpty)
	}
	if lipgloss.Width(resEmpty) != 15 {
		t.Fatalf("expected visible width 15, got %d", lipgloss.Width(resEmpty))
	}
}
