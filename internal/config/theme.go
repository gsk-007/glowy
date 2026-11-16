package config

import (
	"strings"

	"github.com/charmbracelet/glamour/ansi"
	"github.com/charmbracelet/glamour/styles"
	"github.com/charmbracelet/lipgloss"
)

// UITheme contains all Lipgloss styles for rendering the TUI and CLI components.
type UITheme struct {
	Scheme Scheme

	Base         lipgloss.Style
	Muted        lipgloss.Style
	Selected     lipgloss.Style
	FolderHeader lipgloss.Style
	Cursor       lipgloss.Style

	Badge        lipgloss.Style
	TabActive    lipgloss.Style
	TabInactive  lipgloss.Style
	TableHeader  lipgloss.Style
	RowSelected  lipgloss.Style
	RowNormal    lipgloss.Style
	ColFileSel   lipgloss.Style
	ColDir       lipgloss.Style
	ColDate      lipgloss.Style
	TreeBranch   lipgloss.Style
	Divider      lipgloss.Style
	KeyBadge     lipgloss.Style

	BrowserTitle  lipgloss.Style
	BrowserFilter lipgloss.Style
	BrowserItem   lipgloss.Style
	BrowserMatch  lipgloss.Style

	PagerHeader lipgloss.Style
	PagerText   lipgloss.Style
	PagerBadge  lipgloss.Style

	StatusBar   lipgloss.Style
	StatusPill  lipgloss.Style
	StatusMode  lipgloss.Style
	StatusMatch lipgloss.Style

	FrontmatterBox lipgloss.Style
	FrontmatterKey lipgloss.Style
	FrontmatterVal lipgloss.Style

	Gutter lipgloss.Style

	SearchHighlight lipgloss.Style
}

func stringPtr(s string) *string {
	return &s
}

func boolPtr(b bool) *bool {
	return &b
}

// ToGlamourStyle converts the Scheme's Markdown color definitions into a Glamour ansi.StyleConfig.
func (s Scheme) ToGlamourStyle() ansi.StyleConfig {
	base := styles.DarkStyleConfig
	if strings.Contains(strings.ToLower(s.Name), "light") {
		base = styles.LightStyleConfig
	}

	cfg := base
	// Note: Do NOT set StylePrimitive.BackgroundColor on Document, Paragraph,
	// Heading, List, Table, or BlockQuote here. Setting BackgroundColor on
	// Glamour StyleBlocks activates reflow/padding.Writer inside Glamour's
	// BlockStack, which pads inner blocks and table cells with trailing spaces
	// before tablewriter wraps them, truncating wrapped cell text on the right.
	// Full-canvas background painting is handled cleanly after rendering by
	// render.ApplyLineBackground.

	if s.Foreground != "" {
		cfg.Document.StylePrimitive.Color = stringPtr(s.Foreground)
	}

	// Clear Glamour's default "#" / "##" prefixes and H1 background pill while keeping
	// subtle 1-line vertical spacing across heading levels.
	cfg.H1.StylePrimitive.BackgroundColor = nil
	cfg.H1.StylePrimitive.Prefix = ""
	cfg.H1.StylePrimitive.Suffix = ""
	cfg.H1.StylePrimitive.Bold = boolPtr(true)
	cfg.H1.StylePrimitive.BlockPrefix = ""
	cfg.H1.StylePrimitive.BlockSuffix = "\n"
	if s.Markdown.H1 != "" {
		cfg.H1.StylePrimitive.Color = stringPtr(s.Markdown.H1)
	}

	cfg.H2.StylePrimitive.Prefix = ""
	cfg.H2.StylePrimitive.Suffix = ""
	cfg.H2.StylePrimitive.Bold = boolPtr(true)
	cfg.H2.StylePrimitive.BlockPrefix = ""
	cfg.H2.StylePrimitive.BlockSuffix = "\n"
	if s.Markdown.H2 != "" {
		cfg.H2.StylePrimitive.Color = stringPtr(s.Markdown.H2)
	}

	cfg.H3.StylePrimitive.Prefix = ""
	cfg.H3.StylePrimitive.Suffix = ""
	cfg.H3.StylePrimitive.Bold = boolPtr(true)
	cfg.H3.StylePrimitive.BlockPrefix = ""
	cfg.H3.StylePrimitive.BlockSuffix = "\n"
	if s.Markdown.H3 != "" {
		cfg.H3.StylePrimitive.Color = stringPtr(s.Markdown.H3)
	}

	cfg.H4.StylePrimitive.Prefix = ""
	cfg.H4.StylePrimitive.Suffix = ""
	cfg.H4.StylePrimitive.Bold = boolPtr(true)
	cfg.H4.StylePrimitive.BlockPrefix = ""
	cfg.H4.StylePrimitive.BlockSuffix = "\n"
	if s.Foreground != "" {
		cfg.H4.StylePrimitive.Color = stringPtr(s.Foreground)
	}

	cfg.H5.StylePrimitive.Prefix = ""
	cfg.H5.StylePrimitive.Suffix = ""
	cfg.H5.StylePrimitive.Bold = boolPtr(true)
	cfg.H5.StylePrimitive.Italic = boolPtr(true)
	cfg.H5.StylePrimitive.BlockPrefix = ""
	cfg.H5.StylePrimitive.BlockSuffix = "\n"
	if s.Foreground != "" {
		cfg.H5.StylePrimitive.Color = stringPtr(s.Foreground)
	}

	cfg.H6.StylePrimitive.Prefix = ""
	cfg.H6.StylePrimitive.Suffix = ""
	cfg.H6.StylePrimitive.Bold = boolPtr(false)
	cfg.H6.StylePrimitive.Italic = boolPtr(true)
	cfg.H6.StylePrimitive.BlockPrefix = ""
	cfg.H6.StylePrimitive.BlockSuffix = "\n"
	if s.Muted != "" {
		cfg.H6.StylePrimitive.Color = stringPtr(s.Muted)
	}
	if s.Markdown.CodeInlineFg != "" {
		cfg.Code.StylePrimitive.Color = stringPtr(s.Markdown.CodeInlineFg)
	}
	if s.Markdown.CodeBlockBg != "" {
		cfg.Code.StylePrimitive.BackgroundColor = stringPtr(s.Markdown.CodeBlockBg)
		cfg.CodeBlock.StylePrimitive.BackgroundColor = stringPtr(s.Markdown.CodeBlockBg)
		if cfg.CodeBlock.Chroma != nil {
			chromaCopy := *cfg.CodeBlock.Chroma
			chromaCopy.Background.BackgroundColor = stringPtr(s.Markdown.CodeBlockBg)
			cfg.CodeBlock.Chroma = &chromaCopy
		}
	}
	if s.Markdown.QuoteFg != "" {
		cfg.BlockQuote.StylePrimitive.Color = stringPtr(s.Markdown.QuoteFg)
	}
	if s.Markdown.LinkFg != "" {
		cfg.Link.Color = stringPtr(s.Markdown.LinkFg)
		cfg.LinkText.Color = stringPtr(s.Markdown.LinkFg)
	}

	return cfg
}

// NewUITheme builds a UITheme struct populated with Lipgloss styles according to the given Scheme.
func NewUITheme(s Scheme) *UITheme {
	bg := lipgloss.Color(s.Background)
	fg := lipgloss.Color(s.Foreground)
	cursor := lipgloss.Color(s.CursorAccent)
	selBg := lipgloss.Color(s.SelectionBackground)
	muted := lipgloss.Color(s.Muted)
	border := lipgloss.Color(s.Border)
	sbBg := lipgloss.Color(s.StatusBarBg)
	sbFg := lipgloss.Color(s.StatusBarFg)
	shBg := lipgloss.Color(s.SearchHighlightBg)
	shFg := lipgloss.Color(s.SearchHighlightFg)

	return &UITheme{
		Scheme: s,

		Base: lipgloss.NewStyle().
			Foreground(fg).
			Background(bg),

		Muted: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg),

		Selected: lipgloss.NewStyle().
			Foreground(cursor).
			Background(selBg).
			Bold(true),

		FolderHeader: lipgloss.NewStyle().
			Foreground(cursor).
			Background(bg).
			Bold(true),

		Cursor: lipgloss.NewStyle().
			Foreground(cursor).
			Bold(true),

		Badge: lipgloss.NewStyle().
			Background(cursor).
			Foreground(bg).
			Bold(true).
			Padding(0, 1),

		TabActive: lipgloss.NewStyle().
			Background(selBg).
			Foreground(fg).
			Bold(true).
			Padding(0, 1),

		TabInactive: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg).
			Padding(0, 1),

		TableHeader: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg).
			Bold(true),

		RowSelected: lipgloss.NewStyle().
			Background(selBg).
			Foreground(fg),

		RowNormal: lipgloss.NewStyle().
			Foreground(fg).
			Background(bg),

		ColFileSel: lipgloss.NewStyle().
			Foreground(cursor).
			Bold(true),

		ColDir: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg),

		ColDate: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg),

		TreeBranch: lipgloss.NewStyle().
			Foreground(border).
			Background(bg),

		Divider: lipgloss.NewStyle().
			Foreground(border).
			Background(bg),

		KeyBadge: lipgloss.NewStyle().
			Foreground(cursor).
			Bold(true),

		BrowserTitle: lipgloss.NewStyle().
			Foreground(fg).
			Background(sbBg).
			Bold(true).
			Padding(0, 1),

		BrowserFilter: lipgloss.NewStyle().
			Foreground(cursor).
			Bold(true),

		BrowserItem: lipgloss.NewStyle().
			Foreground(fg).
			Background(bg),

		BrowserMatch: lipgloss.NewStyle().
			Foreground(shBg).
			Bold(true),

		PagerHeader: lipgloss.NewStyle().
			Background(sbBg).
			Foreground(fg).
			Bold(true),

		PagerBadge: lipgloss.NewStyle().
			Background(selBg).
			Foreground(fg).
			Bold(true).
			Padding(0, 1),

		PagerText: lipgloss.NewStyle().
			Foreground(fg).
			Background(bg),

		StatusBar: lipgloss.NewStyle().
			Background(sbBg).
			Foreground(sbFg),

		StatusPill: lipgloss.NewStyle().
			Background(cursor).
			Foreground(bg).
			Bold(true).
			Padding(0, 1),

		StatusMode: lipgloss.NewStyle().
			Background(selBg).
			Foreground(fg).
			Padding(0, 1),

		StatusMatch: lipgloss.NewStyle().
			Background(shBg).
			Foreground(shFg).
			Bold(true).
			Padding(0, 1),

		FrontmatterBox: lipgloss.NewStyle().
			BorderStyle(lipgloss.RoundedBorder()).
			BorderForeground(border).
			Background(bg).
			Padding(0, 1).
			MarginBottom(1),

		FrontmatterKey: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg).
			Bold(true),

		FrontmatterVal: lipgloss.NewStyle().
			Foreground(fg).
			Background(bg),

		Gutter: lipgloss.NewStyle().
			Foreground(muted).
			Background(bg),

		SearchHighlight: lipgloss.NewStyle().
			Background(shBg).
			Foreground(shFg).
			Bold(true),
	}
}
