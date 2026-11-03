package main

import (
	"fmt"
	"io"
	"os"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/discovery"
	"github.com/glowy/cli/internal/render"
	"github.com/glowy/cli/internal/ui"
	"github.com/spf13/cobra"
)

var (
	version = "dev"
	commit  = "none"
	date    = "unknown"
)

var (
	flagPager       bool
	flagRaw         bool
	flagLineNumbers bool
	flagStyle       string
	flagWidth       int
	flagThemes      bool
)

func isStdinPiped() bool {
	stat, err := os.Stdin.Stat()
	if err != nil {
		return false
	}
	// Piped from another command or redirected from file
	return (stat.Mode()&os.ModeNamedPipe != 0) || ((stat.Mode()&os.ModeCharDevice == 0) && stat.Size() > 0)
}

func main() {
	initConsole()

	rootCmd := &cobra.Command{
		Use:     "glowy [file | dir]",
		Short:   "glowy — Fast, local-first Windows & cross-platform Markdown reader",
		Version: fmt.Sprintf("%s (%s) built on %s", version, commit, date),
		Args:    cobra.MaximumNArgs(1),
		RunE:    runApp,
	}
	rootCmd.SetVersionTemplate("glowy {{.Version}}\n")

	rootCmd.Flags().BoolVarP(&flagPager, "pager", "p", false, "Open directly in the built-in interactive TUI pager")
	rootCmd.Flags().BoolVarP(&flagRaw, "raw", "r", false, "Render syntax-highlighted raw Markdown source")
	rootCmd.Flags().BoolVarP(&flagLineNumbers, "line-numbers", "l", false, "Display raw file line numbers in the left gutter")
	rootCmd.Flags().StringVarP(&flagStyle, "style", "s", "", "Override active color scheme (e.g. SlateDark, PaperLight)")
	rootCmd.Flags().IntVarP(&flagWidth, "width", "w", 0, "Override dynamic terminal width wrapping")
	rootCmd.Flags().BoolVar(&flagThemes, "themes", false, "List available color schemes with color swatches")

	themesCmd := &cobra.Command{
		Use:   "themes",
		Short: "List all available color schemes and their color previews",
		RunE:  listThemes,
	}
	rootCmd.AddCommand(themesCmd)

	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func listThemes(cmd *cobra.Command, args []string) error {
	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	configPath, _ := config.GetConfigPath()

	fmt.Println()
	fmt.Println("Available Color Schemes (glowy):")
	if configPath != "" {
		fmt.Printf("Config path: %s\n\n", configPath)
	}

	for _, s := range cfg.Schemes {
		isActive := strings.EqualFold(s.Name, cfg.DefaultScheme)
		prefix := "  "
		activeLabel := ""
		if isActive {
			prefix = "* "
			activeLabel = " (active)"
		}

		bgSwatch := lipgloss.NewStyle().Foreground(lipgloss.Color(s.Background)).Render("██")
		fgSwatch := lipgloss.NewStyle().Foreground(lipgloss.Color(s.Foreground)).Render("██")
		cursorSwatch := lipgloss.NewStyle().Foreground(lipgloss.Color(s.CursorAccent)).Render("██")
		h1Swatch := lipgloss.NewStyle().Foreground(lipgloss.Color(s.Markdown.H1)).Render("██")
		codeSwatch := lipgloss.NewStyle().Foreground(lipgloss.Color(s.Markdown.CodeInlineFg)).Render("██")

		fmt.Printf("%s%-14s%s\n", prefix, s.Name, activeLabel)
		fmt.Printf("    Swatches:  Bg %s %s   Fg %s %s   Accent %s %s   H1 %s %s   Code %s %s\n\n",
			bgSwatch, s.Background,
			fgSwatch, s.Foreground,
			cursorSwatch, s.CursorAccent,
			h1Swatch, s.Markdown.H1,
			codeSwatch, s.Markdown.CodeInlineFg,
		)
	}

	fmt.Println("Tip: Use 'glowy -s <scheme>' to test a theme, or press 't' inside glowy to switch live.")
	fmt.Println("You can add custom schemes by editing settings.json.")
	fmt.Println()
	return nil
}

func runApp(cmd *cobra.Command, args []string) error {
	if flagThemes {
		return listThemes(cmd, args)
	}

	cfg, err := config.LoadConfig()
	if err != nil {
		cfg = config.DefaultConfig()
	}

	// Resolve scheme
	schemeName := cfg.DefaultScheme
	if flagStyle != "" {
		schemeName = flagStyle
	}
	scheme := cfg.GetScheme(schemeName)
	theme := config.NewUITheme(scheme)

	// Resolve mode and line numbers
	mode := cfg.DefaultMode
	if flagRaw {
		mode = "raw"
	}
	lineNumbers := cfg.LineNumbers
	if flagLineNumbers {
		lineNumbers = true
	}

	// Case 1: Argument provided (file or directory)
	if len(args) == 1 {
		target := args[0]
		stat, err := os.Stat(target)
		if err != nil {
			return fmt.Errorf("cannot access '%s': %w", target, err)
		}

		if !stat.IsDir() {
			// Single file
			data, err := os.ReadFile(target)
			if err != nil {
				return fmt.Errorf("cannot read '%s': %w", target, err)
			}
			source := string(data)

			// Direct stdout mode (no pager)
			if !flagPager {
				opts := render.RenderOptions{
					Mode:        mode,
					LineNumbers: lineNumbers,
					Width:       flagWidth,
					Theme:       theme,
				}
				res, err := render.Render(source, opts)
				if err != nil {
					return err
				}
				fmt.Println(res.Content)
				return nil
			}

			// Interactive Pager
			fileMeta, err := discovery.ScanSingleFile(target)
			if err != nil {
				fileMeta = &discovery.MarkdownFile{
					Name: stat.Name(),
					Path: target,
				}
			}

			app, err := ui.NewAppModel(nil, fileMeta, source, true, cfg, theme, mode, lineNumbers)
			if err != nil {
				return err
			}

			// Only WithAltScreen (No mouse capture -> enables native text selection)
			p := tea.NewProgram(app, tea.WithAltScreen())
			_, err = p.Run()
			return err
		}

		// Directory argument provided -> Open Browser on that directory
		return openBrowser(target, cfg, theme, mode, lineNumbers)
	}

	// Case 2: Stdin piped (no arguments provided)
	if isStdinPiped() {
		data, err := io.ReadAll(os.Stdin)
		if err != nil {
			return fmt.Errorf("failed to read stdin: %w", err)
		}
		source := string(data)

		// Direct stdout mode (no pager)
		if !flagPager {
			opts := render.RenderOptions{
				Mode:        mode,
				LineNumbers: lineNumbers,
				Width:       flagWidth,
				Theme:       theme,
			}
			res, err := render.Render(source, opts)
			if err != nil {
				return err
			}
			fmt.Println(res.Content)
			return nil
		}

		// Interactive pager on stdin (Only WithAltScreen, no mouse capture)
		var teaOptions []tea.ProgramOption
		teaOptions = append(teaOptions, tea.WithAltScreen())

		// On Windows/Unix, attach console input when stdin is piped
		if conIn, err := os.Open("CONIN$"); err == nil {
			defer conIn.Close()
			teaOptions = append(teaOptions, tea.WithInput(conIn))
		}

		app, err := ui.NewAppModel(nil, &discovery.MarkdownFile{
			Name: "stdin",
			Path: "-",
		}, source, true, cfg, theme, mode, lineNumbers)
		if err != nil {
			return err
		}

		p := tea.NewProgram(app, teaOptions...)
		_, err = p.Run()
		return err
	}

	// Case 3: No argument -> Open Browser on current directory "."
	return openBrowser(".", cfg, theme, mode, lineNumbers)
}

func openBrowser(dir string, cfg *config.Config, theme *config.UITheme, mode string, lineNumbers bool) error {
	files, err := discovery.ScanDirectory(dir)
	if err != nil {
		return fmt.Errorf("failed to scan directory '%s': %w", dir, err)
	}

	app, err := ui.NewAppModel(files, nil, "", false, cfg, theme, mode, lineNumbers)
	if err != nil {
		return err
	}

	// Only WithAltScreen (No mouse capture -> enables native text selection)
	p := tea.NewProgram(app, tea.WithAltScreen())
	_, err = p.Run()
	return err
}
