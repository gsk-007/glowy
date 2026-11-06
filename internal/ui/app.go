package ui

import (
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/glowy/cli/internal/config"
	"github.com/glowy/cli/internal/discovery"
)

type AppState int

const (
	StateBrowser AppState = iota
	StatePager
)

// AppModel coordinates transitions between the File Browser and Document Pager,
// and manages global overlays (Help Menu and Theme Switcher).
type AppModel struct {
	State        AppState
	Browser      BrowserModel
	Pager        *PagerModel
	Theme        *config.UITheme
	Config       *config.Config
	Width        int
	Height       int
	WrapWidth    int
	DirectPager  bool
	LineNumbers  bool
	DefaultMode  string

	ShowingHelp  bool
	ShowingTheme bool
	ThemeModal   ThemeModal
}

// NewAppModel creates a top-level application model.
func NewAppModel(files []discovery.MarkdownFile, initialFile *discovery.MarkdownFile, initialSource string, directPager bool, cfg *config.Config, theme *config.UITheme, mode string, lineNumbers bool, wrapWidth ...int) (AppModel, error) {
	resolvedWrap := 0
	if len(wrapWidth) > 0 && wrapWidth[0] > 0 {
		resolvedWrap = wrapWidth[0]
	} else if cfg != nil {
		resolvedWrap = cfg.ResolveWrapWidth(0)
	}

	app := AppModel{
		Theme:       theme,
		Config:      cfg,
		WrapWidth:   resolvedWrap,
		DirectPager: directPager,
		LineNumbers: lineNumbers,
		DefaultMode: mode,
	}

	if directPager && initialFile != nil {
		app.State = StatePager
		pager, err := NewPagerModel(initialFile.Path, initialSource, theme, mode, lineNumbers, true)
		if err != nil {
			return app, err
		}
		pager.WrapWidth = resolvedWrap
		app.Pager = &pager
	} else {
		app.State = StateBrowser
		app.Browser = NewBrowserModel(files, theme, cfg.SortBy)
	}

	return app, nil
}

func (m *AppModel) applyTheme(scheme config.Scheme) {
	m.Theme = config.NewUITheme(scheme)
	m.Browser.Theme = m.Theme
	if m.Pager != nil {
		m.Pager.Theme = m.Theme
		_ = m.Pager.reRender()
	}
}

func (m AppModel) Init() tea.Cmd {
	if m.State == StatePager && m.Pager != nil {
		return m.Pager.Init()
	}
	return m.Browser.Init()
}

func (m AppModel) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		m.Width = msg.Width
		m.Height = msg.Height
		if m.Pager != nil {
			*m.Pager, _ = m.Pager.Update(msg)
		}
		m.Browser, _ = m.Browser.Update(msg)
		return m, nil

	case tea.KeyMsg:
		// 1. Help Modal Active
		if m.ShowingHelp {
			switch msg.String() {
			case "ctrl+c":
				if m.Pager != nil && m.Pager.Watcher != nil {
					_ = m.Pager.Watcher.Close()
					m.Pager.Watcher = nil
				}
				return m, tea.Quit
			case "?", "esc", "enter", "q", "backspace":
				m.ShowingHelp = false
				return m, nil
			default:
				return m, nil
			}
		}

		// 2. Theme Modal Active
		if m.ShowingTheme {
			switch msg.String() {
			case "ctrl+c":
				if m.Pager != nil && m.Pager.Watcher != nil {
					_ = m.Pager.Watcher.Close()
					m.Pager.Watcher = nil
				}
				return m, tea.Quit
			case "up", "k":
				scheme := m.ThemeModal.MoveCursor(-1)
				m.applyTheme(scheme)
				return m, nil
			case "down", "j":
				scheme := m.ThemeModal.MoveCursor(1)
				m.applyTheme(scheme)
				return m, nil
			case "enter":
				selected := m.ThemeModal.SelectedScheme()
				m.applyTheme(selected)
				_ = config.SaveDefaultScheme(selected.Name)
				m.ShowingTheme = false
				return m, nil
			case "esc", "q", "backspace", "t":
				orig := m.Config.GetScheme(m.ThemeModal.OriginalScheme)
				m.applyTheme(orig)
				m.ShowingTheme = false
				return m, nil
			default:
				return m, nil
			}
		}

		// 3. Global Shortcuts (when text inputs are not typing)
		isTyping := (m.State == StateBrowser && m.Browser.Filtering) || (m.State == StatePager && m.Pager != nil && m.Pager.Searching)
		if !isTyping {
			switch msg.String() {
			case "?":
				m.ShowingHelp = true
				return m, nil
			case "t":
				m.ShowingTheme = true
				m.ThemeModal = NewThemeModal(m.Config.Schemes, m.Theme.Scheme.Name)
				return m, nil
			}
		}
	}

	// 4. Delegate to Active View
	if m.State == StateBrowser {
		var cmd tea.Cmd
		m.Browser, cmd = m.Browser.Update(msg)

		if m.Browser.Canceled {
			return m, tea.Quit
		}

		if m.Browser.SelectedFile != nil {
			filePath := m.Browser.SelectedFile.Path
			data, err := os.ReadFile(filePath)
			if err != nil {
				m.Browser.SelectedFile = nil
				return m, nil
			}

			vpHeight := m.Height - 2
			if vpHeight < 1 {
				vpHeight = 1
			}

			pager, err := NewPagerModel(filePath, string(data), m.Theme, m.DefaultMode, m.LineNumbers, false)
			if err != nil {
				m.Browser.SelectedFile = nil
				return m, nil
			}

			pager.WrapWidth = m.WrapWidth
			pager.Width = m.Width
			pager.Height = m.Height
			pager.Viewport.Width = m.Width
			pager.Viewport.Height = vpHeight
			_ = pager.reRender()
			pager.Viewport.GotoTop()
			pager.Viewport.YOffset = 0

			m.Pager = &pager
			m.State = StatePager
			return m, m.Pager.Init()
		}

		return m, cmd
	}

	if m.State == StatePager && m.Pager != nil {
		var cmd tea.Cmd
		*m.Pager, cmd = m.Pager.Update(msg)

		if m.Pager.Canceled {
			return m, tea.Quit
		}

		if m.Pager.ReturnToBrowser {
			if m.DirectPager {
				return m, tea.Quit
			}
			m.State = StateBrowser
			m.Browser.SelectedFile = nil
			return m, nil
		}

		return m, cmd
	}

	return m, nil
}

func (m AppModel) View() string {
	var baseView string
	if m.State == StatePager && m.Pager != nil {
		baseView = m.Pager.View()
	} else {
		baseView = m.Browser.View()
	}

	if m.ShowingHelp {
		return Overlay(baseView, RenderHelpModal(m.Theme, m.Width), m.Width, m.Height, m.Theme.Scheme.Background)
	}

	if m.ShowingTheme {
		return Overlay(baseView, m.ThemeModal.Render(m.Theme, m.Width), m.Width, m.Height, m.Theme.Scheme.Background)
	}

	return baseView
}
