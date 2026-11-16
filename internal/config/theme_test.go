package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultScheme != "ObsidianGlow" {
		t.Fatalf("expected ObsidianGlow, got %s", cfg.DefaultScheme)
	}
	if len(cfg.Schemes) != 4 {
		t.Fatalf("expected 4 schemes, got %d", len(cfg.Schemes))
	}

	expectedOrder := []string{"ObsidianGlow", "SlateDark", "CrystalLight", "PaperLight"}
	for i, name := range expectedOrder {
		if cfg.Schemes[i].Name != name {
			t.Fatalf("expected scheme %d to be %s, got %s", i, name, cfg.Schemes[i].Name)
		}
		got := cfg.GetScheme(name)
		if got.Name != name {
			t.Fatalf("expected %s, got %s", name, got.Name)
		}
	}

	unknown := cfg.GetScheme("nonexistent")
	if unknown.Name != "ObsidianGlow" {
		t.Fatalf("expected fallback to ObsidianGlow, got %s", unknown.Name)
	}
}

func TestUIThemeAndGlamour(t *testing.T) {
	cfg := DefaultConfig()
	scheme := cfg.GetScheme("SlateDark")

	theme := NewUITheme(scheme)
	if theme == nil {
		t.Fatal("theme should not be nil")
	}

	glamourStyle := scheme.ToGlamourStyle()
	if glamourStyle.H1.StylePrimitive.Color == nil || *glamourStyle.H1.StylePrimitive.Color != scheme.Markdown.H1 {
		t.Fatalf("H1 color mismatch: %v vs %v", glamourStyle.H1.StylePrimitive.Color, scheme.Markdown.H1)
	}
	if glamourStyle.H1.StylePrimitive.BackgroundColor != nil {
		t.Fatalf("expected H1 BackgroundColor to be nil, got %v", *glamourStyle.H1.StylePrimitive.BackgroundColor)
	}
	if glamourStyle.H2.StylePrimitive.Prefix != "" || glamourStyle.H3.StylePrimitive.Prefix != "" {
		t.Fatalf("expected empty heading prefixes, got H2=%q H3=%q", glamourStyle.H2.StylePrimitive.Prefix, glamourStyle.H3.StylePrimitive.Prefix)
	}
}

func TestLoadConfigBootstrap(t *testing.T) {
	tempDir, err := os.MkdirTemp("", "glowy-test-*")
	if err != nil {
		t.Fatal(err)
	}
	defer os.RemoveAll(tempDir)

	// Set temporary config dir via APPDATA or HOME
	origAppData := os.Getenv("APPDATA")
	defer os.Setenv("APPDATA", origAppData)
	_ = os.Setenv("APPDATA", tempDir)

	cfg, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig failed: %v", err)
	}
	if cfg.DefaultScheme != "ObsidianGlow" {
		t.Fatalf("expected ObsidianGlow, got %s", cfg.DefaultScheme)
	}

	// Verify file was written
	writtenPath := filepath.Join(tempDir, "glowy", "settings.json")
	if _, err := os.Stat(writtenPath); err != nil {
		t.Fatalf("settings.json was not created at %s: %v", writtenPath, err)
	}

	// Verify legacy 2-scheme settings.json gets new built-in schemes appended
	legacy := `{"defaultScheme":"PaperLight","schemes":[{"name":"SlateDark","background":"#121417"},{"name":"PaperLight","background":"#FAFAFA"}]}`
	if err := os.WriteFile(writtenPath, []byte(legacy), 0644); err != nil {
		t.Fatalf("failed to write legacy settings.json: %v", err)
	}
	migrated, err := LoadConfig()
	if err != nil {
		t.Fatalf("LoadConfig on legacy settings failed: %v", err)
	}
	if len(migrated.Schemes) != 4 {
		t.Fatalf("expected 4 schemes after merging built-ins, got %d", len(migrated.Schemes))
	}
	expectedOrder := []string{"ObsidianGlow", "SlateDark", "CrystalLight", "PaperLight"}
	for i, name := range expectedOrder {
		if migrated.Schemes[i].Name != name {
			t.Fatalf("expected migrated scheme %d to be %s, got %s", i, name, migrated.Schemes[i].Name)
		}
	}
}
