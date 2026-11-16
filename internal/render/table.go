package render

import (
	"bytes"
	"fmt"
	"html"
	"strconv"
	"strings"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/glowy/cli/internal/config"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/extension"
	astext "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

type parsedTable struct {
	placeholder string
	startLine   int
	endLine     int
	linePrefix  string
	headers     []string
	rows        [][]string
	alignments  []astext.Alignment
}

// preprocessTables parses GFM tables from body, renders each cell's inline AST
// into styled strings, and replaces each table block with a unique sentinel line
// so Glamour handles surrounding layout while lipgloss/table renders the tables.
func preprocessTables(body string, scheme config.Scheme) (string, []parsedTable) {
	if !strings.Contains(body, "|") {
		return body, nil
	}

	source := []byte(body)
	md := goldmark.New(goldmark.WithExtensions(extension.GFM))
	doc := md.Parser().Parse(text.NewReader(source))

	var tableNodes []*astext.Table
	_ = ast.Walk(doc, func(n ast.Node, entering bool) (ast.WalkStatus, error) {
		if !entering {
			return ast.WalkContinue, nil
		}
		if tbl, ok := n.(*astext.Table); ok {
			tableNodes = append(tableNodes, tbl)
			return ast.WalkSkipChildren, nil
		}
		return ast.WalkContinue, nil
	})

	if len(tableNodes) == 0 {
		return body, nil
	}

	tokenBase := uniqueSentinelPrefix(body)
	lines := strings.Split(body, "\n")
	lineStarts := computeLineStartOffsets(lines)

	var tables []parsedTable
	for idx, tbl := range tableNodes {
		startLine, endLine, colOffset, ok := tableLineSpan(tbl, source, lines, lineStarts)
		if !ok {
			continue
		}

		prefix := ""
		if colOffset > 0 && colOffset <= len(lines[startLine]) {
			prefix = lines[startLine][:colOffset]
		}

		headers, rows := extractStyledTableCells(tbl, source, scheme)
		if len(headers) == 0 && len(rows) == 0 {
			continue
		}

		tables = append(tables, parsedTable{
			placeholder: fmt.Sprintf("%s%dEND", tokenBase, idx),
			startLine:   startLine,
			endLine:     endLine,
			linePrefix:  prefix,
			headers:     headers,
			rows:        rows,
			alignments:  tbl.Alignments,
		})
	}

	if len(tables) == 0 {
		return body, nil
	}

	// Replace table line spans bottom-up so earlier line indices remain valid.
	for i := len(tables) - 1; i >= 0; i-- {
		pt := tables[i]
		var replacement []string
		if pt.startLine > 0 && strings.TrimSpace(lines[pt.startLine-1]) != "" {
			replacement = append(replacement, pt.linePrefix)
		}
		replacement = append(replacement, pt.linePrefix+pt.placeholder)
		if pt.endLine+1 < len(lines) && strings.TrimSpace(lines[pt.endLine+1]) != "" {
			replacement = append(replacement, pt.linePrefix)
		}

		updated := make([]string, 0, len(lines)-(pt.endLine-pt.startLine+1)+len(replacement))
		updated = append(updated, lines[:pt.startLine]...)
		updated = append(updated, replacement...)
		updated = append(updated, lines[pt.endLine+1:]...)
		lines = updated
	}

	return strings.Join(lines, "\n"), tables
}

// spliceRenderedTables replaces each table sentinel line in rendered Glamour output
// with a rounded-border, header-accented lipgloss/table fitted to availBodyWidth.
func spliceRenderedTables(renderedLines []string, tables []parsedTable, scheme config.Scheme, availBodyWidth int) []string {
	if len(tables) == 0 {
		return renderedLines
	}

	byPlaceholder := make(map[string]parsedTable, len(tables))
	for _, pt := range tables {
		byPlaceholder[pt.placeholder] = pt
	}

	out := make([]string, 0, len(renderedLines))
	for _, line := range renderedLines {
		plain := StripANSI(line)
		matched := false
		for token, pt := range byPlaceholder {
			idx := strings.Index(plain, token)
			if idx < 0 {
				continue
			}

			indent := plain[:idx]
			indentWidth := lipgloss.Width(indent)
			// Leave a 2-column right margin to match Glamour body margins.
			maxTableWidth := availBodyWidth - indentWidth - 2
			if maxTableWidth < 16 {
				maxTableWidth = availBodyWidth - indentWidth
			}
			if maxTableWidth < 12 {
				maxTableWidth = 12
			}

			tableStr := renderStyledTable(pt, scheme, maxTableWidth)
			for _, tl := range strings.Split(tableStr, "\n") {
				out = append(out, indent+tl)
			}
			matched = true
			break
		}
		if !matched {
			out = append(out, line)
		}
	}

	return out
}

func renderStyledTable(pt parsedTable, scheme config.Scheme, maxWidth int) string {
	borderColor := firstNonEmpty(scheme.Muted, scheme.Border)
	borderStyle := lipgloss.NewStyle()
	if borderColor != "" {
		borderStyle = borderStyle.Foreground(lipgloss.Color(borderColor))
	}

	numCols := len(pt.headers)
	for _, r := range pt.rows {
		if len(r) > numCols {
			numCols = len(r)
		}
	}

	build := func(width int) string {
		t := table.New().
			Border(lipgloss.RoundedBorder()).
			BorderTop(true).
			BorderBottom(true).
			BorderLeft(true).
			BorderRight(true).
			BorderHeader(true).
			BorderColumn(true).
			BorderRow(true).
			BorderStyle(borderStyle).
			Wrap(true).
			StyleFunc(func(row, col int) lipgloss.Style {
				st := lipgloss.NewStyle().Padding(0, 1)
				if col >= 0 && col < len(pt.alignments) {
					switch pt.alignments[col] {
					case astext.AlignCenter:
						st = st.Align(lipgloss.Center)
					case astext.AlignRight:
						st = st.Align(lipgloss.Right)
					default:
						st = st.Align(lipgloss.Left)
					}
				}
				return st
			})

		if width > 0 {
			t.Width(width)
		}
		if len(pt.headers) > 0 {
			t.Headers(pt.headers...)
		}
		for _, r := range pt.rows {
			if len(r) < numCols {
				padded := make([]string, numCols)
				copy(padded, r)
				r = padded
			}
			t.Row(r...)
		}
		return strings.TrimRight(t.Render(), "\n")
	}

	rendered := build(0)
	if maxWidth > 0 {
		firstLine := rendered
		if nl := strings.IndexByte(rendered, '\n'); nl >= 0 {
			firstLine = rendered[:nl]
		}
		if lipgloss.Width(firstLine) > maxWidth {
			rendered = build(maxWidth)
		}
	}
	return rendered
}

func extractStyledTableCells(tbl *astext.Table, source []byte, scheme config.Scheme) (headers []string, rows [][]string) {
	fgHex := firstNonEmpty(scheme.Foreground, "#E2E8F0")
	headerHex := firstNonEmpty(scheme.Markdown.H1, scheme.CursorAccent, fgHex)

	bodyBase := lipgloss.NewStyle().Foreground(lipgloss.Color(fgHex))
	headerBase := lipgloss.NewStyle().Foreground(lipgloss.Color(headerHex)).Bold(true)

	for child := tbl.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case astext.KindTableHeader:
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				headers = append(headers, renderCellInlineAST(cell, source, headerBase, scheme))
			}
		case astext.KindTableRow:
			var row []string
			for cell := child.FirstChild(); cell != nil; cell = cell.NextSibling() {
				row = append(row, renderCellInlineAST(cell, source, bodyBase, scheme))
			}
			rows = append(rows, row)
		}
	}
	return headers, rows
}

func renderCellInlineAST(node ast.Node, source []byte, style lipgloss.Style, scheme config.Scheme) string {
	var sb strings.Builder
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			txt := html.UnescapeString(string(n.Segment.Value(source)))
			if txt != "" {
				sb.WriteString(style.Render(txt))
			}
			if n.SoftLineBreak() || n.HardLineBreak() {
				sb.WriteString(style.Render(" "))
			}
		case *ast.String:
			txt := html.UnescapeString(string(n.Value))
			if txt != "" {
				sb.WriteString(style.Render(txt))
			}
		case *ast.CodeSpan:
			codeText := collectPlainText(n, source)
			codeStyle := style.Copy()
			if scheme.Markdown.CodeInlineFg != "" {
				codeStyle = codeStyle.Foreground(lipgloss.Color(scheme.Markdown.CodeInlineFg))
			}
			sb.WriteString(codeStyle.Render(codeText))
		case *ast.Emphasis:
			emphStyle := style.Copy()
			if n.Level >= 2 {
				emphStyle = emphStyle.Bold(true)
			} else {
				emphStyle = emphStyle.Italic(true)
			}
			sb.WriteString(renderCellInlineAST(n, source, emphStyle, scheme))
		case *astext.Strikethrough:
			strikeStyle := style.Copy().Strikethrough(true)
			sb.WriteString(renderCellInlineAST(n, source, strikeStyle, scheme))
		case *ast.Link:
			linkStyle := style.Copy().Underline(true)
			if scheme.Markdown.LinkFg != "" {
				linkStyle = linkStyle.Foreground(lipgloss.Color(scheme.Markdown.LinkFg))
			}
			rendered := renderCellInlineAST(n, source, linkStyle, scheme)
			if rendered == "" && len(n.Destination) > 0 {
				rendered = linkStyle.Render(string(n.Destination))
			}
			sb.WriteString(rendered)
		case *ast.AutoLink:
			linkStyle := style.Copy().Underline(true)
			if scheme.Markdown.LinkFg != "" {
				linkStyle = linkStyle.Foreground(lipgloss.Color(scheme.Markdown.LinkFg))
			}
			sb.WriteString(linkStyle.Render(string(n.Label(source))))
		case *ast.Image:
			alt := collectPlainText(n, source)
			if alt != "" {
				sb.WriteString(style.Render(alt))
			}
		default:
			sb.WriteString(renderCellInlineAST(child, source, style, scheme))
		}
	}
	return strings.TrimSpace(sb.String())
}

func collectPlainText(node ast.Node, source []byte) string {
	var buf bytes.Buffer
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		switch n := child.(type) {
		case *ast.Text:
			buf.Write(n.Segment.Value(source))
			if n.SoftLineBreak() || n.HardLineBreak() {
				buf.WriteByte(' ')
			}
		case *ast.String:
			buf.Write(n.Value)
		default:
			buf.WriteString(collectPlainText(child, source))
		}
	}
	return buf.String()
}

func uniqueSentinelPrefix(body string) string {
	prefix := "GLOWYTBLSENTINEL"
	for i := 0; strings.Contains(body, prefix); i++ {
		prefix = "GLOWYTBLSENTINEL" + strconv.Itoa(i) + "X"
	}
	return prefix
}

func computeLineStartOffsets(lines []string) []int {
	starts := make([]int, len(lines))
	offset := 0
	for i, l := range lines {
		starts[i] = offset
		offset += len(l) + 1
	}
	return starts
}

func tableLineSpan(tbl *astext.Table, source []byte, lines []string, lineStarts []int) (startLine, endLine, colOffset int, ok bool) {
	first := tbl.FirstChild()
	if first == nil {
		return 0, 0, 0, false
	}

	headerPos := first.Pos()
	if headerPos < 0 {
		minStart, _, found := nodeByteBounds(tbl)
		if !found {
			return 0, 0, 0, false
		}
		headerPos = minStart
	}

	startLine = lineAtOffset(source, headerPos) - 1
	if startLine < 0 || startLine >= len(lines) {
		return 0, 0, 0, false
	}
	colOffset = headerPos - lineStarts[startLine]
	if colOffset < 0 {
		colOffset = 0
	}

	endLine = startLine + 1 // At least header + delimiter line
	if last := tbl.LastChild(); last != nil && last != first && last.Pos() >= 0 {
		lastLine := lineAtOffset(source, last.Pos()) - 1
		if lastLine > endLine {
			endLine = lastLine
		}
	}
	if endLine >= len(lines) {
		endLine = len(lines) - 1
	}

	return startLine, endLine, colOffset, true
}

func firstNonEmpty(vals ...string) string {
	for _, v := range vals {
		if strings.TrimSpace(v) != "" {
			return v
		}
	}
	return ""
}
