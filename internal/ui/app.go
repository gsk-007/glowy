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

// AppModel coordinates transitions between the File Browser and the Document Pager.
type AppModel struct {
	State        AppState
	Browser      BrowserModel
	Pager        *PagerModel
	Theme        *config.UITheme
	Config       *config.Config
	Width        int
	Height       int
	DirectPager  bool
	LineNumbers  bool
	DefaultMode  string
}

// NewAppModel creates a top-level application model.
func NewAppModel(files []discovery.MarkdownFile, initialFile *discovery.MarkdownFile, initialSource string, directPager bool, cfg *config.Config, theme *config.UITheme, mode string, lineNumbers bool) (AppModel, error) {
	app := AppModel{
		Theme:       theme,
		Config:      cfg,
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
		app.Pager = &pager
	} else {
		app.State = StateBrowser
		app.Browser = NewBrowserModel(files, theme, cfg.SortBy)
	}

	return app, nil
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
	}

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

			pager, err := NewPagerModel(filePath, string(data), m.Theme, m.DefaultMode, m.LineNumbers, false)
			if err != nil {
				m.Browser.SelectedFile = nil
				return m, nil
			}

			vpHeight := m.Height - 2
			if vpHeight < 1 {
				vpHeight = 1
			}
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
	if m.State == StatePager && m.Pager != nil {
		return m.Pager.View()
	}
	return m.Browser.View()
}
