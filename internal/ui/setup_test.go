package ui

import (
	"strings"
	"testing"

	"github.com/tiagokriok/Git-Fuzzy/internal/config"
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
)

func newSetupModelForTest(warning *theme.Warning, themeConfig config.ThemeConfig) SetupModel {
	return newSetupModel(theme.Theme{Name: "Test"}, warning, themeConfig)
}

func TestSetupPathsViewRendersThemeWarning(t *testing.T) {
	m := newSetupModelForTest(&theme.Warning{Message: "theme failed"}, config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
	m.step = 1

	view := m.pathsView()
	if !strings.Contains(view, "theme failed") {
		t.Fatalf("expected paths view to contain theme warning, got %q", view)
	}
}

func TestSetupPathsViewOmitsWarningWhenAbsent(t *testing.T) {
	m := newSetupModelForTest(nil, config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
	m.step = 1

	view := m.pathsView()
	if strings.Contains(view, "theme failed") {
		t.Fatalf("expected paths view without warning to omit theme text, got %q", view)
	}
}

func TestSetupTmuxActionViewRendersThemeWarning(t *testing.T) {
	m := newSetupModelForTest(&theme.Warning{Message: "theme failed"}, config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
	m.step = 2

	view := m.tmuxActionView()
	if !strings.Contains(view, "theme failed") {
		t.Fatalf("expected tmux action view to contain theme warning, got %q", view)
	}
}

func TestSetupEditorViewRendersThemeWarning(t *testing.T) {
	m := newSetupModelForTest(&theme.Warning{Message: "theme failed"}, config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
	m.step = 0

	view := m.editorView()
	if !strings.Contains(view, "theme failed") {
		t.Fatalf("expected editor view to contain theme warning, got %q", view)
	}
}

func TestSetupModelUsesProvidedThemeConfig(t *testing.T) {
	cfg := config.ThemeConfig{Mode: string(config.ThemeModeFile), Path: "/tmp/theme.json"}
	m := newSetupModelForTest(nil, cfg)

	if m.themeConfig.Mode != cfg.Mode {
		t.Fatalf("expected theme mode %q, got %q", cfg.Mode, m.themeConfig.Mode)
	}
	if m.themeConfig.Path != cfg.Path {
		t.Fatalf("expected theme path %q, got %q", cfg.Path, m.themeConfig.Path)
	}
}
