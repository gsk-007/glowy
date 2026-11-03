# PRD: glowy — Windows-First CLI & TUI Markdown Reader

## 1. Product Vision
`glowy` is a fast, local-first, Windows-native (yet cross-platform) CLI and TUI Markdown reader written in Go. It solves `glow`'s core architectural issues:
- Eliminates remote/cloud bloat (100% local files + `stdin` only, no `$EDITOR` spawning, no GitHub/GitLab APIs, no clipboard dependencies).
- Fixes silent CLI flag failures (`-l` line numbers and `-p` pager work reliably across both CLI and TUI).
- Replaces external Linux `less -r` dependencies with a unified, built-in Bubble Tea TUI pager (`-p`).
- Maps line numbers (`-l`) accurately to the **raw source file lines**.
- Supports both **Formatted Markdown** and **Syntax-Highlighted Raw Markdown** (`-r` flag and live TUI toggle).
- Renders YAML frontmatter (`---`) as a structured metadata block instead of stripping or mangling it.
- Provides a Windows Terminal-inspired `settings.json` for unified TUI + Markdown color schemes (1 Dark + 1 Light built-in, plus user customization).

---

## 2. CLI Contract & Execution Modes

| Command | Mode | Behavior |
| :--- | :--- | :--- |
| `glowy <file.md>` | Direct CLI (`stdout`) | Renders formatted Markdown to `stdout` and exits. Dynamically wraps to current terminal width. |
| `glowy -r <file.md>` | Direct CLI (`stdout`) | Renders syntax-highlighted raw Markdown source to `stdout` and exits. |
| `glowy -l <file.md>` | Direct CLI (`stdout`) | Renders output to `stdout` with a left gutter showing raw source file line numbers. Combines with `-r` or formatted mode. |
| `glowy -p <file.md>` | Built-in TUI Pager | Opens `<file.md>` directly inside the native Bubble Tea full-screen pager. Never calls external `less`. |
| `glowy` or `glowy <dir>` | Built-in TUI Browser | Recursively discovers `.md` files in the directory, launches the TUI File Browser, and opens selected files in the TUI Pager. |
| `cat a.md \| glowy` | Stdin (`stdout` / `-p`) | Reads piped Markdown from `stdin` and outputs to `stdout` (or opens in TUI Pager if `-p` is passed). |

### CLI Flags
- `-p`, `--pager`: Open directly in the built-in interactive TUI pager.
- `-r`, `--raw`: Render syntax-highlighted raw Markdown source instead of formatted Glamour output.
- `-l`, `--line-numbers`: Display raw file line numbers in the left gutter (works in both CLI `stdout` and TUI pager).
- `-s`, `--style <scheme>`: Override the active color scheme from `settings.json` (e.g., `SlateDark`, `PaperLight`, or custom scheme name).
- `-w`, `--width <int>`: Override dynamic terminal width wrapping with a fixed column width.

---

## 3. Core Component Requirements

### A. Windows Console & Config Bootstrap (`internal/config`)
- Enable Windows Virtual Terminal Processing (`ENABLE_VIRTUAL_TERMINAL_PROCESSING`) and force TrueColor (`termenv.TrueColor`) on startup so PowerShell and Windows Terminal render 24-bit hex colors accurately.
- Load configuration from `%APPDATA%\glowy\settings.json` (`os.UserConfigDir()`). Automatically generate the default `settings.json` if missing.
- Unify styling: The active scheme in `settings.json` styles both the Bubble Tea TUI chrome (borders, status bar, selection, search highlights) and the Glamour/Chroma Markdown rendering.

### B. Local File Discovery Engine (`internal/discovery`)
- **Strict Default Ignores (No `--all` flag):** Automatically ignore `.git`, `node_modules`, `vendor`, `AppData`, `$RECYCLE.BIN`, `System Volume Information`, hidden dot-directories (`.vscode`, `.idea`, etc.), and patterns in `.gitignore`.
- **Direct Path Bypass:** If a user passes an explicit file path (`glowy ignored_dir/doc.md`), bypass directory ignore rules and open the file directly.
- **OS Metadata:** Normalize display paths with forward slashes (`/`). Extract `ModifiedAt` (`mtime`) and `CreatedAt` (using `syscall.Win32FileAttributeData.CreationTime` on Windows, fallback to `mtime` on other OSes).

### C. Rendering Engine (`internal/render`)
- **Dynamic Width:** Detect terminal width dynamically on launch and on `tea.WindowSizeMsg`.
- **YAML Frontmatter Block:** Detect top `--- ... ---` YAML blocks, parse key-value pairs, and render them as a bordered metadata card/table at the top of formatted output while tracking the line offset.
- **Formatted Mode:** Render Markdown via `glamour` using a custom `ansi.StyleConfig` built dynamically from the active `settings.json` color scheme.
- **Raw Mode:** Render raw Markdown text using `chroma/v2` (Markdown lexer + terminal256/truecolor formatter matching the active scheme).
- **Raw Line Number Mapping (`-l`):**
  - In Raw mode: 1:1 line numbers (`1..N`) in a dimmed left gutter (`  1 │ `).
  - In Formatted mode: Parse the Markdown AST via `goldmark` to map block start lines (accounting for frontmatter line count) to their corresponding raw file line numbers in the left gutter.

### D. TUI View 1: File Browser (`internal/ui/browser.go`)
- **Flat List View (Default):** Shows all discovered `.md` files with filename, relative folder path, and relative timestamp.
- **Folder View Toggle (`Tab` or `v`):** Groups `.md` files under collapsible/grouped parent directory headers.
- **Sorting (`s` key cycles):**
  1. `Modified At` (Newest first — Default)
  2. `File Name` (Alphabetical A–Z)
  3. `Created At` (Newest first)
- **Fuzzy Search (`/`):** Case-insensitive filtering across file names and relative paths.
- **Navigation:** `Up`/`Down` or `j`/`k`, `Enter` to open in TUI Pager, `q` or `Esc` to quit.

### E. TUI View 2: Native Pager (`internal/ui/pager.go`)
- **Mouse Support:** Mouse wheel scrolling (`tea.WithMouseCellMotion()`) enabled by default.
- **Familiar Navigation Keys:**
  - `Up`/`Down` or `j`/`k`: Scroll 1 line
  - `PgUp`/`PgDn` or `u`/`d`: Scroll half/full page
  - `Home`/`End` or `g`/`G`: Jump to top/bottom
  - `r`: Toggle between Formatted Markdown and Syntax-Highlighted Raw Markdown
  - `l`: Toggle raw line number gutter on/off
  - `/`: Activate case-insensitive in-document search bar
  - `n` / `N`: Jump viewport to next / previous search match
  - `Esc` / `Backspace`: Clear search (if active) or return to File Browser (if launched from directory) / quit (if launched via `glowy -p file.md`)
  - `q`: Quit application
- **In-Document Search (`/`):** Case-insensitive substring matching across the rendered lines. Highlights all occurrences using `searchHighlightBg` / `searchHighlightFg` from `settings.json` and scrolls the active match into view.
- **Live File Reload (`fsnotify`):** Watch the currently open local file for write events and re-render automatically while preserving the current scroll offset.
- **Bottom Status Bar:**
  - Displays: ` filename.md │ Formatted (or Raw) │ Ln <raw_line> (<scroll>%) │ <current>/<total> matches `

---

## 4. Target Directory Structure
```text
glowy/
├── cmd/
│   └── glowy/
│       ├── main.go              # Cobra CLI flags, stdin detection, CLI vs TUI router
│       ├── console_windows.go   # Windows VT100 + TrueColor initialization
│       └── console_other.go     # No-op for non-Windows platforms
├── internal/
│   ├── config/
│   │   ├── settings.go          # settings.json loader, auto-bootstrapper, structs
│   │   └── theme.go             # Converts Scheme struct -> Lipgloss styles & Glamour StyleConfig
│   ├── discovery/
│   │   ├── scanner.go           # Recursive .md walker with .gitignore & system ignores
│   │   ├── metadata_windows.go  # Windows CreationTime + ModTime extractor
│   │   └── metadata_other.go    # Fallback timestamp extractor
│   ├── render/
│   │   ├── engine.go            # Unified renderer (Formatted Glamour + Raw Chroma)
│   │   ├── frontmatter.go       # YAML frontmatter parser & bordered box renderer
│   │   └── linenumbers.go       # Goldmark AST raw line mapper & left gutter compositor
│   └── ui/
│       ├── app.go               # Root Bubble Tea model coordinating Browser & Pager views
│       ├── browser.go           # Flat list + Folder view, sorting (Modified/Name/Created), filter
│       ├── pager.go             # Viewport, mouse wheel, live fsnotify reload, status bar
│       └── search.go            # Case-insensitive in-doc search & ANSI-safe match highlighter
├── settings.default.json
├── go.mod
└── PRD.md
