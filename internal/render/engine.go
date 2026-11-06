package render

import (
	"bytes"
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/alecthomas/chroma/v2"
	"github.com/alecthomas/chroma/v2/formatters"
	"github.com/alecthomas/chroma/v2/lexers"
	"github.com/charmbracelet/glamour"
	"github.com/charmbracelet/lipgloss"
	"github.com/glowy/cli/internal/config"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"golang.org/x/term"
)

// RenderOptions holds configurations for rendering a Markdown document.
type RenderOptions struct {
	Mode        string          // "formatted" or "raw"
	LineNumbers bool            // show left raw line number gutter
	Width       int             // wrap target width (0 = auto-detect)
	CanvasWidth int             // full terminal canvas width for background padding (0 = same as Width)
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

	canvasWidth := opts.CanvasWidth
	if canvasWidth < termWidth {
		canvasWidth = termWidth
	}

	// Normalize CRLF to LF and expand tabs to 4 spaces so visual width calculations are exact
	source = strings.ReplaceAll(source, "\r\n", "\n")
	source = strings.ReplaceAll(source, "\t", "    ")
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

	// Calculate exact wrap width for Glamour so that text + margins + gutter NEVER exceed termWidth.
	// Glamour adds defaultMargin (2 spaces) and for lists up to 4 spaces indentation.
	glamourMargin := 4
	gutterWidth := 0
	if opts.LineNumbers {
		digits := len(strconv.Itoa(totalRawLines))
		if digits < 2 {
			digits = 2
		}
		gutterWidth = digits + 3 // "%*d │ "
	}

	// Right margin safety buffer (2 columns) so text never touches or overflows the right boundary
	rightPadding := 2

	wrapWidth := termWidth - glamourMargin - gutterWidth - rightPadding
	if wrapWidth < 20 {
		wrapWidth = 20
	}

	mode := strings.ToLower(opts.Mode)
	if mode == "" {
		mode = "formatted"
	}

	if mode == "raw" {
		return renderRaw(source, totalRawLines, termWidth, canvasWidth, gutterWidth, opts)
	}

	return renderFormatted(body, fm, fmLinesCount, totalRawLines, wrapWidth, termWidth, canvasWidth, gutterWidth, opts)
}

func renderFormatted(body string, fm *Frontmatter, fmLinesCount, totalRawLines, wrapWidth, termWidth, canvasWidth, gutterWidth int, opts RenderOptions) (*RenderResult, error) {
	styleCfg := opts.Theme.Scheme.ToGlamourStyle()
	renderer, err := glamour.NewTermRenderer(
		glamour.WithStyles(styleCfg),
		glamour.WithWordWrap(wrapWidth),
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

	availBodyWidth := termWidth - gutterWidth
	if availBodyWidth < 12 {
		availBodyWidth = 12
	}

	// Prepend frontmatter box if present
	if fm != nil && len(fm.Fields) > 0 {
		fmBox := RenderFrontmatter(fm, opts.Theme, availBodyWidth)
		if fmBox != "" {
			fmLines := strings.Split(fmBox, "\n")
			var wrappedFmLines []string
			for _, fl := range fmLines {
				if lipgloss.Width(fl) > availBodyWidth {
					wrappedFmLines = append(wrappedFmLines, strings.Split(wrap.String(fl, availBodyWidth), "\n")...)
				} else {
					wrappedFmLines = append(wrappedFmLines, fl)
				}
			}
			fmRenderedLinesCount = len(wrappedFmLines)
			allLines = append(allLines, wrappedFmLines...)
			allLines = append(allLines, "") // Blank spacing line
		}
	}

	// Clean up leading newlines Glamour introduces
	trimmedMarkdown := strings.TrimLeft(renderedMarkdown, "\n")
	trimmedMarkdown = strings.TrimRight(trimmedMarkdown, "\n")
	if trimmedMarkdown != "" {
		bodyLines := strings.Split(trimmedMarkdown, "\n")
		// Safety pass before attaching gutter: wrap any lines that exceed availBodyWidth
		// (e.g. long unbroken URLs or code lines) so ComposeFormattedGutter naturally
		// assigns blankGutter to wrapped continuation lines without string-splitting on "│".
		for _, bl := range bodyLines {
			if lipgloss.Width(bl) > availBodyWidth {
				wrapped := strings.Split(wrap.String(wordwrap.String(bl, availBodyWidth), availBodyWidth), "\n")
				allLines = append(allLines, wrapped...)
			} else {
				allLines = append(allLines, bl)
			}
		}
	}

	if opts.LineNumbers {
		astBlockLines := ExtractASTBlockLines([]byte(body))
		allLines = ComposeFormattedGutter(allLines, fmLinesCount, fmRenderedLinesCount, astBlockLines, totalRawLines, opts.Theme)
	}

	for i, l := range allLines {
		allLines[i] = ApplyLineBackground(l, canvasWidth, opts.Theme.Scheme.Background)
	}

	joined := strings.Join(allLines, "\n")
	return &RenderResult{
		Content:          joined,
		Lines:            allLines,
		TotalRawLines:    totalRawLines,
		FrontmatterLines: fmLinesCount,
	}, nil
}

func renderRaw(source string, totalRawLines, termWidth, canvasWidth, gutterWidth int, opts RenderOptions) (*RenderResult, error) {
	digits := 2
	if opts.LineNumbers {
		digits = len(strconv.Itoa(totalRawLines))
		if digits < 2 {
			digits = 2
		}
	}

	availRawWidth := termWidth - gutterWidth - 2
	if availRawWidth < 12 {
		availRawWidth = 12
	}

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

	var wrappedLines []string
	blankGutter := opts.Theme.Gutter.Render(fmt.Sprintf("%*s │ ", digits, ""))

	for i, hLine := range highlightedLines {
		sublines := strings.Split(wrap.String(wordwrap.String(hLine, availRawWidth), availRawWidth), "\n")
		for subIdx, sub := range sublines {
			if opts.LineNumbers {
				if subIdx == 0 {
					gutterRendered := opts.Theme.Gutter.Render(fmt.Sprintf("%*d │ ", digits, i+1))
					wrappedLines = append(wrappedLines, gutterRendered+sub)
				} else {
					wrappedLines = append(wrappedLines, blankGutter+sub)
				}
			} else {
				wrappedLines = append(wrappedLines, sub)
			}
		}
	}

	for i, l := range wrappedLines {
		wrappedLines[i] = ApplyLineBackground(l, canvasWidth, opts.Theme.Scheme.Background)
	}

	joined := strings.Join(wrappedLines, "\n")
	return &RenderResult{
		Content:          joined,
		Lines:            wrappedLines,
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
