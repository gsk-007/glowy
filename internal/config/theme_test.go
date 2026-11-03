package config

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()
	if cfg.DefaultScheme != "SlateDark" {
		t.Fatalf("expected SlateDark, got %s", cfg.DefaultScheme)
	}
	if len(cfg.Schemes) != 2 {
		t.Fatalf("expected 2 schemes, got %d", len(cfg.Schemes))
	}

	slate := cfg.GetScheme("SlateDark")
	if slate.Name != "SlateDark" {
		t.Fatalf("expected SlateDark, got %s", slate.Name)
	}

	paper := cfg.GetScheme("PaperLight")
	if paper.Name != "PaperLight" {
		t.Fatalf("expected PaperLight, got %s", paper.Name)
	}

	unknown := cfg.GetScheme("nonexistent")
	if unknown.Name != "SlateDark" {
		t.Fatalf("expected fallback to SlateDark, got %s", unknown.Name)
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
	if cfg.DefaultScheme != "SlateDark" {
		t.Fatalf("expected SlateDark, got %s", cfg.DefaultScheme)
	}

	// Verify file was written
	writtenPath := filepath.Join(tempDir, "glowy", "settings.json")
	if _, err := os.Stat(writtenPath); err != nil {
		t.Fatalf("settings.json was not created at %s: %v", writtenPath, err)
	}
}
