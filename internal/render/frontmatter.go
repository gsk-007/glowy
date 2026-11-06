package render

import (
	"fmt"
	"strings"

	"github.com/glowy/cli/internal/config"
	"github.com/muesli/reflow/wordwrap"
	"github.com/muesli/reflow/wrap"
	"gopkg.in/yaml.v3"
)

// Frontmatter holds the parsed frontmatter fields and the number of raw lines it occupied.
type Frontmatter struct {
	RawLines int
	Fields   [][2]string
}

// ExtractFrontmatter inspects content for top YAML frontmatter delimited by `---`.
// It returns the parsed Frontmatter (if found) and the remaining Markdown body.
func ExtractFrontmatter(content string) (*Frontmatter, string) {
	// Trim optional UTF-8 BOM and normalize Windows CRLF to LF
	content = strings.TrimPrefix(content, "\ufeff")
	content = strings.ReplaceAll(content, "\r\n", "\n")

	// Check if content starts with ---
	if !strings.HasPrefix(content, "---") {
		return nil, content
	}

	lines := strings.Split(content, "\n")
	if len(lines) < 2 {
		return nil, content
	}

	firstLine := lines[0]
	if firstLine != "---" {
		return nil, content
	}

	endIdx := -1
	for i := 1; i < len(lines); i++ {
		line := lines[i]
		if line == "---" || line == "..." {
			endIdx = i
			break
		}
	}

	if endIdx == -1 {
		// No matching closing delimiter
		return nil, content
	}

	yamlBlock := strings.Join(lines[1:endIdx], "\n")
	body := strings.Join(lines[endIdx+1:], "\n")

	fm := &Frontmatter{
		RawLines: endIdx + 1,
		Fields:   parseYAMLOrdered(yamlBlock),
	}

	return fm, body
}

// parseYAMLOrdered parses YAML into ordered key-value string pairs using yaml.Node.
func parseYAMLOrdered(raw string) [][2]string {
	var root yaml.Node
	if err := yaml.Unmarshal([]byte(raw), &root); err != nil {
		return nil
	}

	var results [][2]string
	if len(root.Content) == 0 {
		return results
	}

	docNode := root.Content[0]
	if docNode.Kind != yaml.MappingNode {
		return results
	}

	for i := 0; i+1 < len(docNode.Content); i += 2 {
		kNode := docNode.Content[i]
		vNode := docNode.Content[i+1]

		key := kNode.Value
		var val string

		switch vNode.Kind {
		case yaml.ScalarNode:
			val = vNode.Value
		case yaml.SequenceNode:
			var items []string
			for _, item := range vNode.Content {
				items = append(items, item.Value)
			}
			val = fmt.Sprintf("[%s]", strings.Join(items, ", "))
		case yaml.MappingNode:
			val = "{...}"
		default:
			val = vNode.Value
		}

		results = append(results, [2]string{key, val})
	}

	return results
}

// RenderFrontmatter formats the frontmatter fields into a stylish Lipgloss-bordered card.
func RenderFrontmatter(fm *Frontmatter, theme *config.UITheme, maxWidth int) string {
	if fm == nil || len(fm.Fields) == 0 {
		return ""
	}

	maxKeyLen := 0
	for _, entry := range fm.Fields {
		if len(entry[0]) > maxKeyLen {
			maxKeyLen = len(entry[0])
		}
	}

	boxWidth := 76
	if maxWidth > 10 && maxWidth < boxWidth {
		boxWidth = maxWidth
	}

	// Available inner width for value column: boxWidth - borders(2) - padding(2) - keyLen(maxKeyLen+2)
	valWidth := boxWidth - 4 - (maxKeyLen + 2)
	if valWidth < 12 {
		valWidth = 12
	}

	var rows []string
	keyIndent := strings.Repeat(" ", maxKeyLen+2)
	for _, entry := range fm.Fields {
		paddedKey := fmt.Sprintf("%-*s", maxKeyLen+1, entry[0]+":")
		keyStr := theme.FrontmatterKey.Render(paddedKey)

		wrappedVal := strings.Split(wrap.String(wordwrap.String(entry[1], valWidth), valWidth), "\n")
		for idx, vLine := range wrappedVal {
			valStr := theme.FrontmatterVal.Render(vLine)
			if idx == 0 {
				rows = append(rows, fmt.Sprintf("%s %s", keyStr, valStr))
			} else {
				rows = append(rows, fmt.Sprintf("%s%s", theme.FrontmatterKey.Render(keyIndent), valStr))
			}
		}
	}

	content := strings.Join(rows, "\n")
	return theme.FrontmatterBox.Render(content)
}
