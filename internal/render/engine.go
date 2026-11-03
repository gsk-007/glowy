package render

import (
	"bytes"
	"os"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/glamour"
	"github.com/glowy/cli/internal/config"
	"golang.org/x/term"
)

// RenderOptions holds configurations for rendering a Markdown document.
type RenderOptions struct {
	Mode        string          // "formatted" or "raw"
	LineNumbers bool            // show left raw line number gutter
	Width       int             // terminal width (0 = auto-detect)
	Theme       *config.UITheme // active UITheme
}

// RenderResult contains the rendered output lines and metadata.
type RenderResult struct {
	Content          string
	Lines            []string
	TotalRawLines    int
	FrontmatterLines int
}

// DetectTerminalWidth returns the current terminal width or a safe fallback.
func DetectTerminalWidth() int {
	fd := int(os.Stdout.Fd())
	if !term.IsTerminal(fd) {
		fd = int(os.Stderr.Fd())
	}
	if term.IsTerminal(fd) {
		if w, _, err := term.GetSize(fd); err == nil && w > 10 {
			return w
		}
	}
	return 80
}

// Render processes the given Markdown source and produces formatted or raw rendered lines.
func Render(source string, opts RenderOptions) (*RenderResult, error) {
	if opts.Theme == nil {
		cfg := config.DefaultConfig()
		opts.Theme = config.NewUITheme(cfg.GetScheme(cfg.DefaultScheme))
	}

	termWidth := opts.Width
	if termWidth <= 0 {
		termWidth = DetectTerminalWidth()
	}

	// Normalize CRLF to LF
	source = strings.ReplaceAll(source, "\r\n", "\n")
	rawTrimmed := strings.TrimRight(source, "\n")
	rawLines := strings.Split(rawTrimmed, "\n")
	totalRawLines := len(rawLines)
	if totalRawLines == 0 {
		totalRawLines = 1
	}

	// Extract frontmatter
	fm, body := ExtractFrontmatter(source)
	fmLinesCount := 0
	if fm != nil {
		fmLinesCount = fm.RawLines
	}

	// Account for gutter width if line numbers are enabled
	renderWidth := termWidth
	if renderWidth < 20 {
		renderWidth = 80
	}
	if opts.LineNumbers {
		gutterEst := 7 // e.g. " 123 │ "
		if renderWidth > gutterEst+20 {
			renderWidth -= gutterEst
		}
	}

	mode := strings.ToLower(opts.Mode)
	if mode == "" {
		mode = "formatted"
	}

	if mode == "raw" {
		return renderRaw(source, rawLines, totalRawLines, opts)
	}

	return renderFormatted(body, fm, fmLinesCount, totalRawLines, renderWidth, opts)
}

func renderFormatted(body string, fm *Frontmatter, fmLinesCount, totalRawLines, renderWidth int, opts RenderOptions) (*RenderResult, error) {
	styleCfg := opts.Theme.Scheme.ToGlamourStyle()
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styleCfg),
		glamour.WithWordWrap(renderWidth),
	)
	if err != nil {
		return nil, err
	}

	renderedMarkdown, err := renderer.Render(body)
	if err != nil {
		return nil, err
	}

	var allLines []string
	fmRenderedLinesCount := 0

	// Prepend frontmatter box if present
	if fm != nil && len(fm.Fields) > 0 {
		fmBox := RenderFrontmatter(fm, opts.Theme, renderWidth)
		if fmBox != "" {
			fmLines := strings.Split(fmBox, "\n")
			fmRenderedLinesCount = len(fmLines)
			allLines = append(allLines, fmLines...)
			allLines = append(allLines, "") // Blank spacing line
		}
	}

	// Clean up leading newlines Glamour introduces
	trimmedMarkdown := strings.TrimLeft(renderedMarkdown, "\n")
	trimmedMarkdown = strings.TrimRight(trimmedMarkdown, "\n")
	if trimmedMarkdown != "" {
		bodyLines := strings.Split(trimmedMarkdown, "\n")
		allLines = append(allLines, bodyLines...)
	}

	if opts.LineNumbers {
		astBlockLines := ExtractASTBlockLines([]byte(body))
		allLines = ComposeFormattedGutter(allLines, fmLinesCount, fmRenderedLinesCount, astBlockLines, totalRawLines, opts.Theme)
	}

	termWidth := opts.Width
	if termWidth <= 0 {
		termWidth = DetectTerminalWidth()
	}
	for i, l := range allLines {
		allLines[i] = ApplyLineBackground(l, termWidth, opts.Theme.Scheme.Background)
	}

	joined := strings.Join(allLines, "\n")
	return &RenderResult{
		Content:          joined,
		Lines:            allLines,
		TotalRawLines:    totalRawLines,
		FrontmatterLines: fmLinesCount,
	}, nil
}

func renderRaw(source string, rawLines []string, totalRawLines int, opts RenderOptions) (*RenderResult, error) {
	lexer := lexers.Get("markdown")
	if lexer == nil {
		lexer = lexers.Fallback
	}
	lexer = chroma.Coalesce(lexer)

	formatter := formatters.Get("terminal16m")
	if formatter == nil {
		formatter = formatters.Fallback
	}

	style := buildChromaStyle(opts.Theme.Scheme)

	iterator, err := lexer.Tokenise(nil, source)
	if err != nil {
		return nil, err
	}

	var buf bytes.Buffer
	if err := formatter.Format(&buf, style, iterator); err != nil {
		return nil, err
	}

	highlighted := buf.String()
	highlightedLines := strings.Split(strings.TrimRight(highlighted, "\n"), "\n")

	if opts.LineNumbers {
		highlightedLines = ComposeRawGutter(highlightedLines, opts.Theme)
	}

	termWidth := opts.Width
	if termWidth <= 0 {
		termWidth = DetectTerminalWidth()
	}
	for i, l := range highlightedLines {
		highlightedLines[i] = ApplyLineBackground(l, termWidth, opts.Theme.Scheme.Background)
	}

	joined := strings.Join(highlightedLines, "\n")
	return &RenderResult{
		Content:          joined,
		Lines:            highlightedLines,
		TotalRawLines:    totalRawLines,
		FrontmatterLines: 0,
	}, nil
}

func buildChromaStyle(s config.Scheme) *chroma.Style {
	builder := chroma.NewStyleBuilder("glowy-" + s.Name)
	builder.Add(chroma.Background, s.Foreground+" bg:"+s.Background)
	builder.Add(chroma.Text, s.Foreground)
	builder.Add(chroma.Comment, s.Muted+" italic")
	builder.Add(chroma.Keyword, s.CursorAccent+" bold")
	builder.Add(chroma.GenericHeading, s.Markdown.H1+" bold")
	builder.Add(chroma.GenericSubheading, s.Markdown.H2+" bold")
	builder.Add(chroma.String, s.Markdown.CodeInlineFg)
	builder.Add(chroma.LiteralString, s.Markdown.CodeInlineFg)
	builder.Add(chroma.LiteralStringBacktick, s.Markdown.CodeInlineFg)
	builder.Add(chroma.NameTag, s.Markdown.H3)
	builder.Add(chroma.GenericPrompt, s.CursorAccent)
	builder.Add(chroma.GenericDeleted, "#EF4444")
	builder.Add(chroma.GenericInserted, "#22C55E")
	builder.Add(chroma.Number, "#F59E0B")
	builder.Add(chroma.Operator, s.Markdown.LinkFg)

	style, err := builder.Build()
	if err != nil {
		return chroma.MustNewStyle("fallback", chroma.StyleEntries{
			chroma.Background: s.Foreground + " bg:" + s.Background,
			chroma.Comment:    s.Muted,
			chroma.Keyword:    s.CursorAccent,
		})
	}
	return style
}
