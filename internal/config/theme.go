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

// ToGlamourStyle converts the Scheme's Markdown color definitions into a Glamour ansi.StyleConfig.
func (s Scheme) ToGlamourStyle() ansi.StyleConfig {
	base := styles.DarkStyleConfig
	if strings.Contains(strings.ToLower(s.Name), "light") {
		base = styles.LightStyleConfig
	}

	cfg := base
	if s.Background != "" {
		cfg.Document.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.Paragraph.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.Heading.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.H1.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.H2.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.H3.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.H4.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.H5.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.H6.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.List.StyleBlock.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.Table.StyleBlock.StylePrimitive.BackgroundColor = stringPtr(s.Background)
		cfg.BlockQuote.StylePrimitive.BackgroundColor = stringPtr(s.Background)
	}

	if s.Foreground != "" {
		cfg.Document.StylePrimitive.Color = stringPtr(s.Foreground)
	}

	if s.Markdown.H1 != "" {
		cfg.H1.StylePrimitive.Color = stringPtr(s.Markdown.H1)
	}
	if s.Markdown.H2 != "" {
		cfg.H2.StylePrimitive.Color = stringPtr(s.Markdown.H2)
	}
	if s.Markdown.H3 != "" {
		cfg.H3.StylePrimitive.Color = stringPtr(s.Markdown.H3)
	}
	if s.Markdown.CodeInlineFg != "" {
		cfg.Code.StylePrimitive.Color = stringPtr(s.Markdown.CodeInlineFg)
	}
	if s.Markdown.CodeBlockBg != "" {
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
