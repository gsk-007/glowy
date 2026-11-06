package config

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strconv"
	"strings"
)

type MarkdownColors struct {
	H1           string `json:"h1"`
	H2           string `json:"h2"`
	H3           string `json:"h3"`
	CodeInlineFg string `json:"codeInlineFg"`
	CodeBlockBg  string `json:"codeBlockBg"`
	QuoteFg      string `json:"quoteFg"`
	LinkFg       string `json:"linkFg"`
}

type Scheme struct {
	Name                string         `json:"name"`
	Background          string         `json:"background"`
	Foreground          string         `json:"foreground"`
	CursorAccent        string         `json:"cursorAccent"`
	SelectionBackground string         `json:"selectionBackground"`
	Muted               string         `json:"muted"`
	Border              string         `json:"border"`
	StatusBarBg         string         `json:"statusBarBg"`
	StatusBarFg         string         `json:"statusBarFg"`
	SearchHighlightBg   string         `json:"searchHighlightBg"`
	SearchHighlightFg   string         `json:"searchHighlightFg"`
	Markdown            MarkdownColors `json:"markdown"`
}

type Config struct {
	DefaultScheme string   `json:"defaultScheme"`
	LineNumbers   bool     `json:"lineNumbers"`
	DefaultMode   string   `json:"defaultMode"`
	SortBy        string   `json:"sortBy"`
	WordWrap      string   `json:"wordWrap"`
	Schemes       []Scheme `json:"schemes"`
}

// DefaultSlateDark returns the default SlateDark scheme.
func DefaultSlateDark() Scheme {
	return Scheme{
		Name:                "SlateDark",
		Background:          "#121417",
		Foreground:          "#E2E8F0",
		CursorAccent:        "#2E8B57",
		SelectionBackground: "#264F78",
		Muted:               "#64748B",
		Border:              "#1E293B",
		StatusBarBg:         "#0F1115",
		StatusBarFg:         "#94A3B8",
		SearchHighlightBg:   "#EAB308",
		SearchHighlightFg:   "#000000",
		Markdown: MarkdownColors{
			H1:           "#38BDF8",
			H2:           "#34D399",
			H3:           "#A78BFA",
			CodeInlineFg: "#F472B6",
			CodeBlockBg:  "#1A1D24",
			QuoteFg:      "#94A3B8",
			LinkFg:       "#60A5FA",
		},
	}
}

// DefaultPaperLight returns the default PaperLight scheme.
func DefaultPaperLight() Scheme {
	return Scheme{
		Name:                "PaperLight",
		Background:          "#FAFAFA",
		Foreground:          "#1E293B",
		CursorAccent:        "#059669",
		SelectionBackground: "#BFDBFE",
		Muted:               "#64748B",
		Border:              "#E2E8F0",
		StatusBarBg:         "#F1F5F9",
		StatusBarFg:         "#475569",
		SearchHighlightBg:   "#FDE047",
		SearchHighlightFg:   "#000000",
		Markdown: MarkdownColors{
			H1:           "#0284C7",
			H2:           "#059669",
			H3:           "#7C3AED",
			CodeInlineFg: "#DB2777",
			CodeBlockBg:  "#F1F5F9",
			QuoteFg:      "#64748B",
			LinkFg:       "#2563EB",
		},
	}
}

// DefaultConfig returns the default application configuration.
func DefaultConfig() *Config {
	return &Config{
		DefaultScheme: "SlateDark",
		LineNumbers:   false,
		DefaultMode:   "formatted",
		SortBy:        "modified",
		WordWrap:      "dynamic",
		Schemes: []Scheme{
			DefaultSlateDark(),
			DefaultPaperLight(),
		},
	}
}

// GetConfigPath returns the canonical path to settings.json in %APPDATA%\glowy or ~/.config/glowy.
func GetConfigPath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		home, homeErr := os.UserHomeDir()
		if homeErr != nil {
			return "", err
		}
		configDir = filepath.Join(home, ".config")
	}
	return filepath.Join(configDir, "glowy", "settings.json"), nil
}

// LoadConfig loads the settings from the user config directory, bootstrapping default settings if missing.
func LoadConfig() (*Config, error) {
	targetPath, pathErr := GetConfigPath()
	var data []byte

	// If user config exists, load it
	if pathErr == nil {
		if content, err := os.ReadFile(targetPath); err == nil {
			data = content
		}
	}

	// Fallback to local settings.json in current directory if config directory file doesn't exist
	if len(data) == 0 {
		if localContent, err := os.ReadFile("settings.json"); err == nil {
			data = localContent
		}
	}

	// If still not found, write and bootstrap the default settings
	if len(data) == 0 {
		cfg := DefaultConfig()
		if pathErr == nil {
			_ = os.MkdirAll(filepath.Dir(targetPath), 0755)
			if encoded, err := json.MarshalIndent(cfg, "", "  "); err == nil {
				_ = os.WriteFile(targetPath, encoded, 0644)
			}
		}
		return cfg, nil
	}

	var cfg Config
	if err := json.Unmarshal(data, &cfg); err != nil {
		return DefaultConfig(), err
	}

	// Ensure fallback schemes exist
	if len(cfg.Schemes) == 0 {
		cfg.Schemes = []Scheme{DefaultSlateDark(), DefaultPaperLight()}
	}
	if cfg.DefaultScheme == "" {
		cfg.DefaultScheme = "SlateDark"
	}
	if cfg.DefaultMode == "" {
		cfg.DefaultMode = "formatted"
	}
	if cfg.SortBy == "" {
		cfg.SortBy = "modified"
	}

	return &cfg, nil
}

// GetScheme returns the scheme matching name (case-insensitive) or the default scheme.
func (c *Config) GetScheme(name string) Scheme {
	target := strings.TrimSpace(name)
	if target != "" {
		for _, s := range c.Schemes {
			if strings.EqualFold(s.Name, target) {
				return s
			}
		}
	}

	// Try default scheme
	for _, s := range c.Schemes {
		if strings.EqualFold(s.Name, c.DefaultScheme) {
			return s
		}
	}

	// Fallback to first scheme or hardcoded default
	if len(c.Schemes) > 0 {
		return c.Schemes[0]
	}

	return DefaultSlateDark()
}

// ResolveWrapWidth returns flagWidth if > 0, or a numeric WordWrap from settings.json if configured, else 0 (dynamic).
func (c *Config) ResolveWrapWidth(flagWidth int) int {
	if flagWidth > 0 {
		return flagWidth
	}
	if c == nil {
		return 0
	}
	w, err := strconv.Atoi(strings.TrimSpace(c.WordWrap))
	if err == nil && w > 0 {
		return w
	}
	return 0
}

// SaveDefaultScheme updates the default scheme in settings.json and persists it to disk.
func SaveDefaultScheme(schemeName string) error {
	path, err := GetConfigPath()
	if err != nil {
		return err
	}

	cfg, err := LoadConfig()
	if err != nil {
		cfg = DefaultConfig()
	}
	cfg.DefaultScheme = schemeName

	_ = os.MkdirAll(filepath.Dir(path), 0755)
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}
