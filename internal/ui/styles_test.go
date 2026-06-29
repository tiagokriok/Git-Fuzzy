package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/tiagokriok/Git-Fuzzy/internal/config"
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
)

func TestNewStylesUsesThemeAccent(t *testing.T) {
	styles := NewStyles(theme.Theme{
		Name: "Test",
		Palette: theme.Palette{
			Accent:     "#111111",
			Foreground: "#222222",
			Muted:      "#333333",
			Border:     "#444444",
			Selection:  "#555555",
			Success:    "#666666",
			Warning:    "#777777",
			Danger:     "#888888",
			Info:       "#999999",
			Branch:     "#aaaaaa",
			Stash:      "#bbbbbb",
			Renamed:    "#cccccc",
			Copied:     "#dddddd",
			Untracked:  "#eeeeee",
		},
	})

	if styles.Accent != lipgloss.Color("#111111") {
		t.Fatalf("expected accent #111111, got %q", styles.Accent)
	}
}

func TestThemeWarningText(t *testing.T) {
	warning := theme.Warning{Message: "theme failed"}
	if got := themeWarningText(&warning); got != "theme failed" {
		t.Fatalf("expected warning text, got %q", got)
	}
	if got := themeWarningText(nil); got != "" {
		t.Fatalf("expected empty warning, got %q", got)
	}
}

func TestNewSetupModelUsesStyles(t *testing.T) {
	appTheme := theme.Theme{
		Name: "Test",
		Palette: theme.Palette{
			Accent:     "#111111",
			Foreground: "#222222",
			Muted:      "#333333",
			Border:     "#444444",
			Selection:  "#555555",
			Success:    "#666666",
			Warning:    "#777777",
			Danger:     "#888888",
			Info:       "#999999",
			Branch:     "#aaaaaa",
			Stash:      "#bbbbbb",
			Renamed:    "#cccccc",
			Copied:     "#dddddd",
			Untracked:  "#eeeeee",
		},
	}
	model := newSetupModel(appTheme, &theme.Warning{Message: "theme warning"}, config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
	if model.styles.Title.GetBold() != true {
		t.Fatal("expected bold title style")
	}
	if model.themeWarning == nil || model.themeWarning.Message != "theme warning" {
		t.Fatalf("expected theme warning")
	}
	if model.themeConfig.Mode != string(config.ThemeModeAuto) {
		t.Fatalf("expected theme config mode auto, got %q", model.themeConfig.Mode)
	}
}
