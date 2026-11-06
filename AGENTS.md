# Project Agent Guidelines (`glowy`)

## Git & Version Control Rules
- **Never use `git add -A` or `git add .`**: Always stage specific, explicitly named file paths only (e.g., `git add cmd/glowy/main.go internal/render/engine.go`).
- **Never commit compiled binaries**: Never stage or commit `glowy.exe`, `*.exe`, or other build artifacts.
- **Git Author**: Use `gsk-007 <gsingh16472@gmail.com>` when committing changes.

## TUI & Rendering Invariants
- **Native Mouse Selection**: Only pass `tea.WithAltScreen()` to Bubble Tea (`tea.NewProgram`) — never enable mouse tracking so native terminal click-and-drag text selection remains functional.
- **Full-Canvas Background & Zero Layout Shift**: Maintain strict viewport height (`m.Height - 2`) and full-width (`m.Width`) 24-bit TrueColor background painting across all views and modals.
