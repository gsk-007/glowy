# glowy

[![Go Version](https://img.shields.io/badge/Go-1.22+-00ADD8?style=flat&logo=go)](https://golang.org)
[![License: MIT](https://img.shields.io/badge/License-MIT-blue.svg)](LICENSE)
[![Release](https://img.shields.io/badge/release-v0.1.1-emerald.svg)](https://github.com/glowy/cli/releases)

`glowy` is a fast, Windows-first (and cross-platform) CLI and TUI Markdown reader built in Go, designed to eliminate terminal rendering, paging, and line-numbering pitfalls. It renders local Markdown files and standard input with zero external dependencies, native mouse text selection, and dynamic 24-bit TrueColor themes.

---

## Why glowy?

- **Built-in Native TUI Pager (`-p`):** Completely eliminates dependencies on external Linux utilities like `less -r`. Works smoothly in Windows Terminal, PowerShell, CMD, Linux, and macOS with live `fsnotify` file-watch and instant auto-reload on save while preserving your scroll position.
- **Accurate Raw Source Line Numbers (`-l`):** Unlike tools that number rendered output lines sequentially, `glowy` parses the Markdown AST via Goldmark to map rendered blocks and YAML frontmatter offsets back to the exact 1-indexed line numbers of your raw source file.
- **Instant Formatted vs. Raw Toggle (`-r` / `r`):** Toggle between styled Glamour Markdown formatting and Chroma syntax-highlighted raw Markdown source on the fly without exiting the pager.
- **Case-Insensitive In-Document Search (`/`):** Search within documents with ANSI-safe match highlighting using scheme colors (`searchHighlightBg` / `searchHighlightFg`) and navigate matches using `n` (next) and `N` (previous).
- **Native Mouse Text Selection:** Leaves terminal mouse tracking unhijacked (`tea.WithAltScreen()` only), allowing native click-and-drag mouse text selection across all views without holding `Shift`.
- **Vertical Card File Browser:** Sleek 2-line card feed displaying filenames, relative timestamps, and dimmed directory subtexts with live `/` fuzzy search and `s` sort cycling (`Modified ↓`, `Name A-Z`, `Created ↓`).
- **Windows Terminal-Style `settings.json` Themes:** Unified TrueColor color schemes styling both the TUI chrome (status bar, borders, selection, highlights) and Markdown elements. Live preview schemes with `t` in the TUI or run `glowy themes` in the CLI.

---

## Installation

### Via `go install`
```bash
go install github.com/glowy/cli/cmd/glowy@latest
```

### Building From Source
Requirements: Go 1.22 or higher.

```bash
git clone https://github.com/glowy/cli.git
cd cli
go build -ldflags="-s -w -X main.version=v0.1.1" -o glowy.exe ./cmd/glowy
```

---

## Quick Start & CLI Usage

### 1. Direct CLI Rendering (stdout)
Print formatted Markdown directly to standard output:
```bash
glowy README.md
```

Display raw source file line numbers in the left gutter:
```bash
glowy -l README.md
```

Display syntax-highlighted raw Markdown with 1:1 line numbers:
```bash
glowy -r -l README.md
```

Override active color scheme:
```bash
glowy -s PaperLight README.md
```

Override terminal wrap width:
```bash
glowy -w 100 README.md
```

### 2. Built-in TUI Pager
Open a document directly inside the full-screen interactive pager:
```bash
glowy -p README.md
```

### 3. File Browser
Launch the interactive file browser in the current directory:
```bash
glowy
```

Launch the file browser in a specific directory:
```bash
glowy docs/
```

### 4. Piped Standard Input
Pipe Markdown from any command directly to `stdout` or the interactive pager:
```bash
# Direct stdout
cat README.md | glowy

# Interactive TUI pager
cat README.md | glowy -p
```

### 5. Color Scheme Utilities
Inspect available color schemes, view TrueColor ANSI swatches, and check the configuration path:
```bash
glowy themes
```

---

## CLI Flags Reference

| Flag | Shorthand | Type | Description |
| :--- | :--- | :--- | :--- |
| `--pager` | `-p` | `bool` | Open document directly in the built-in interactive TUI pager. |
| `--raw` | `-r` | `bool` | Render syntax-highlighted raw Markdown source instead of formatted output. |
| `--line-numbers` | `-l` | `bool` | Display raw file line numbers in the left gutter (works in stdout and pager). |
| `--style` | `-s` | `string` | Override active color scheme (e.g. `SlateDark`, `PaperLight`). |
| `--width` | `-w` | `int` | Override dynamic terminal width wrapping with a fixed column count. |
| `--themes` | | `bool` | List available color schemes with color swatches and config path. |
| `--version` | `-v` | `bool` | Print version, commit hash, and build date. |
| `--help` | `-h` | `bool` | Print command-line help. |

---

## TUI Keyboard Shortcuts

### File Browser (Card Feed View)
| Key | Action |
| :--- | :--- |
| `↑` / `k` | Move cursor to previous file card |
| `↓` / `j` | Move cursor to next file card |
| `PgUp` / `u` | Move cursor up by half page |
| `PgDn` / `d` | Move cursor down by half page |
| `Home` / `g` | Jump to first file card |
| `End` / `G` | Jump to last file card |
| `Enter` | Open selected file in Document Pager |
| `/` | Activate live fuzzy search filter across filenames and directory paths |
| `s` | Cycle sort order: `Modified ↓` → `Name A-Z` → `Created ↓` |
| `t` | Open interactive Theme Switcher modal |
| `?` | Toggle Help Menu overlay |
| `Esc` | Clear active search filter, or quit application |
| `q` / `Ctrl+C` | Quit application |

### Document Pager
| Key | Action |
| :--- | :--- |
| `↑` / `k` | Scroll up 1 line |
| `↓` / `j` | Scroll down 1 line |
| `PgUp` / `u` | Scroll up half page |
| `PgDn` / `d` | Scroll down half page |
| `Home` / `g` | Jump to top of document (`YOffset = 0`) |
| `End` / `G` | Jump to bottom of document |
| `/` | Activate in-document case-insensitive search bar |
| `n` | Jump to next search match |
| `N` | Jump to previous search match |
| `r` | Toggle between Formatted Markdown and Syntax-Highlighted Raw Markdown |
| `l` | Toggle raw source line numbers gutter on/off |
| `t` | Open interactive Theme Switcher modal |
| `?` | Toggle Help Menu overlay |
| `Esc` / `Backspace` | Clear search / Close modal / Return to File Browser (or exit if opened via `-p`) |
| `q` / `Ctrl+C` | Quit application |

---

## Configuration & Custom Themes

`glowy` automatically creates a default `settings.json` on its first launch if one does not exist.

### Configuration File Locations
- **Windows:** `%APPDATA%\glowy\settings.json` (`C:\Users\<user>\AppData\Roaming\glowy\settings.json`)
- **Linux & macOS:** `~/.config/glowy/settings.json`

### `settings.json` Schema
```json
{
  "defaultScheme": "SlateDark",
  "lineNumbers": false,
  "defaultMode": "formatted",
  "sortBy": "modified",
  "wordWrap": "dynamic",
  "schemes": [
    {
      "name": "SlateDark",
      "background": "#121417",
      "foreground": "#E2E8F0",
      "cursorAccent": "#2E8B57",
      "selectionBackground": "#264F78",
      "muted": "#64748B",
      "border": "#1E293B",
      "statusBarBg": "#0F1115",
      "statusBarFg": "#94A3B8",
      "searchHighlightBg": "#EAB308",
      "searchHighlightFg": "#000000",
      "markdown": {
        "h1": "#38BDF8",
        "h2": "#34D399",
        "h3": "#A78BFA",
        "codeInlineFg": "#F472B6",
        "codeBlockBg": "#1A1D24",
        "quoteFg": "#94A3B8",
        "linkFg": "#60A5FA"
      }
    },
    {
      "name": "PaperLight",
      "background": "#FAFAFA",
      "foreground": "#1E293B",
      "cursorAccent": "#059669",
      "selectionBackground": "#BFDBFE",
      "muted": "#64748B",
      "border": "#E2E8F0",
      "statusBarBg": "#F1F5F9",
      "statusBarFg": "#475569",
      "searchHighlightBg": "#FDE047",
      "searchHighlightFg": "#000000",
      "markdown": {
        "h1": "#0284C7",
        "h2": "#059669",
        "h3": "#7C3AED",
        "codeInlineFg": "#DB2777",
        "codeBlockBg": "#F1F5F9",
        "quoteFg": "#64748B",
        "linkFg": "#2563EB"
      }
    }
  ]
}
```

### Adding Custom Schemes
To add a custom color scheme, append a new object to the `"schemes"` array in your `settings.json` and set `"defaultScheme"` to its name (or pass `-s <name>` on the CLI). All hex colors are rendered using 24-bit TrueColor.

---

## License

MIT License. See [LICENSE](LICENSE) for details.
