package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/tiagokriok/Git-Fuzzy/internal/config"
)

func writeFile(t *testing.T, path string, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		t.Fatalf("failed to create dir: %v", err)
	}
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write file: %v", err)
	}
}

func TestParseOmarchyColors(t *testing.T) {
	colors, err := parseOmarchyColors(strings.NewReader(`
# comment
accent = "#7aa2f7"
foreground = "#a9b1d6"
invalid = "blue"
color2 = "#9ece6a"
`))
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	if colors["accent"] != "#7aa2f7" {
		t.Fatalf("expected accent color, got %q", colors["accent"])
	}
	if _, ok := colors["invalid"]; ok {
		t.Fatal("expected invalid color to be ignored")
	}
}

func TestMapOmarchyPalette(t *testing.T) {
	palette := mapOmarchyPalette(map[string]string{
		"accent":               "#7aa2f7",
		"foreground":           "#a9b1d6",
		"selection_background": "#7aa2f7",
		"color1":               "#f7768e",
		"color2":               "#9ece6a",
		"color3":               "#e0af68",
		"color5":               "#ad8ee6",
		"color6":               "#449dab",
		"color7":               "#787c99",
		"color8":               "#444b6a",
	})

	if palette.Accent != "#7aa2f7" {
		t.Fatalf("expected accent, got %q", palette.Accent)
	}
	if palette.Success != "#9ece6a" {
		t.Fatalf("expected success, got %q", palette.Success)
	}
	if palette.Renamed != "#ad8ee6" {
		t.Fatalf("expected renamed, got %q", palette.Renamed)
	}
	if palette.Untracked != "#449dab" {
		t.Fatalf("expected untracked, got %q", palette.Untracked)
	}
}

func TestMergePaletteUsesDefaultsForMissingAndInvalidColors(t *testing.T) {
	base := Default().Palette
	merged := mergePalette(base, Palette{
		Accent:     "#111111",
		Foreground: "not-a-color",
	})

	if merged.Accent != "#111111" {
		t.Fatalf("expected custom accent, got %q", merged.Accent)
	}
	if merged.Foreground != base.Foreground {
		t.Fatalf("expected default foreground, got %q", merged.Foreground)
	}
	if merged.Success != base.Success {
		t.Fatalf("expected default success, got %q", merged.Success)
	}
}

func TestLoadFileAcceptsFullCacheFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "theme.json")
	writeFile(t, path, `{
		"source": "manual",
		"name": "Custom",
		"synced_at": "2026-06-29T00:00:00Z",
		"palette": {"accent": "#111111", "success": "#222222"}
	}`)

	theme, err := LoadFile(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if theme.Name != "Custom" {
		t.Fatalf("expected name Custom, got %q", theme.Name)
	}
	if theme.Palette.Accent != "#111111" {
		t.Fatalf("expected accent, got %q", theme.Palette.Accent)
	}
	if theme.Palette.Success != "#222222" {
		t.Fatalf("expected success, got %q", theme.Palette.Success)
	}
}

func TestLoadFileAcceptsFlatPaletteFormat(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "theme.json")
	writeFile(t, path, `{"accent": "#111111", "foreground": "#222222"}`)

	theme, err := LoadFile(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if theme.Palette.Accent != "#111111" {
		t.Fatalf("expected accent, got %q", theme.Palette.Accent)
	}
	if theme.Palette.Foreground != "#222222" {
		t.Fatalf("expected foreground, got %q", theme.Palette.Foreground)
	}
}

func TestLoadAutoUsesFreshOmarchyCache(t *testing.T) {
	tmpDir := t.TempDir()
	omarchyDir := filepath.Join(tmpDir, "omarchy", "current", "theme")
	cachePath := filepath.Join(tmpDir, "gitf", "theme.json")
	writeFile(t, filepath.Join(omarchyDir, "theme.name"), "Tokyo Night\n")
	writeFile(t, cachePath, `{
		"source": "omarchy",
		"name": "Tokyo Night",
		"synced_at": "2026-06-29T00:00:00Z",
		"palette": {"accent": "#111111"}
	}`)

	theme, warning := LoadWithOptions(config.ThemeConfig{Mode: "auto"}, Options{
		OmarchyThemeDir: omarchyDir,
		CachePath:       cachePath,
	})
	if warning != nil {
		t.Fatalf("expected no warning, got %v", warning)
	}
	if theme.Palette.Accent != "#111111" {
		t.Fatalf("expected cache accent, got %q", theme.Palette.Accent)
	}
}

func TestLoadAutoReloadsStaleOmarchyCacheAndWritesCache(t *testing.T) {
	tmpDir := t.TempDir()
	omarchyDir := filepath.Join(tmpDir, "omarchy", "current", "theme")
	cachePath := filepath.Join(tmpDir, "gitf", "theme.json")
	writeFile(t, filepath.Join(omarchyDir, "theme.name"), "Tokyo Night\n")
	writeFile(t, filepath.Join(omarchyDir, "colors.toml"), `
accent = "#7aa2f7"
foreground = "#a9b1d6"
color1 = "#f7768e"
color2 = "#9ece6a"
color3 = "#e0af68"
color5 = "#ad8ee6"
color6 = "#449dab"
color7 = "#787c99"
color8 = "#444b6a"
`)
	writeFile(t, cachePath, `{
		"source": "omarchy",
		"name": "Old Theme",
		"synced_at": "2026-06-29T00:00:00Z",
		"palette": {"accent": "#111111"}
	}`)

	theme, warning := LoadWithOptions(config.ThemeConfig{Mode: "auto"}, Options{
		OmarchyThemeDir: omarchyDir,
		CachePath:       cachePath,
	})
	if warning != nil {
		t.Fatalf("expected no warning, got %v", warning)
	}
	if theme.Name != "Tokyo Night" {
		t.Fatalf("expected Tokyo Night, got %q", theme.Name)
	}
	if theme.Palette.Accent != "#7aa2f7" {
		t.Fatalf("expected Omarchy accent, got %q", theme.Palette.Accent)
	}

	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("expected cache write, got %v", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("expected valid cache, got %v", err)
	}
	if cache.Name != "Tokyo Night" {
		t.Fatalf("expected updated cache name, got %q", cache.Name)
	}
}

func TestReadOmarchyThemeNamePrefersParentLayout(t *testing.T) {
	tmpDir := t.TempDir()
	// Actual Omarchy layout: theme.name is a sibling of the theme/ directory.
	omarchyCurrent := filepath.Join(tmpDir, "omarchy", "current")
	themeDir := filepath.Join(omarchyCurrent, "theme")
	writeFile(t, filepath.Join(omarchyCurrent, "theme.name"), "Tokyo Night\n")

	name, err := readOmarchyThemeName(themeDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if name != "Tokyo Night" {
		t.Fatalf("expected Tokyo Night from parent layout, got %q", name)
	}
}

func TestReadOmarchyThemeNameFallsBackToLegacyLayout(t *testing.T) {
	tmpDir := t.TempDir()
	// Legacy/test layout: theme.name lives inside the theme/ directory.
	themeDir := filepath.Join(tmpDir, "omarchy", "current", "theme")
	writeFile(t, filepath.Join(themeDir, "theme.name"), "Tokyo Night\n")

	name, err := readOmarchyThemeName(themeDir)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if name != "Tokyo Night" {
		t.Fatalf("expected Tokyo Night from legacy layout, got %q", name)
	}
}

func TestReadOmarchyThemeNameReturnsErrorWhenMissing(t *testing.T) {
	themeDir := filepath.Join(t.TempDir(), "omarchy", "current", "theme")
	if err := os.MkdirAll(themeDir, 0755); err != nil {
		t.Fatalf("failed to create theme dir: %v", err)
	}
	if _, err := readOmarchyThemeName(themeDir); err == nil {
		t.Fatal("expected error when theme.name is missing")
	}
}

func TestLoadOmarchyWithParentLayout(t *testing.T) {
	tmpDir := t.TempDir()
	omarchyCurrent := filepath.Join(tmpDir, "omarchy", "current")
	themeDir := filepath.Join(omarchyCurrent, "theme")
	writeFile(t, filepath.Join(omarchyCurrent, "theme.name"), "Tokyo Night\n")
	writeFile(t, filepath.Join(themeDir, "colors.toml"), `
accent = "#7aa2f7"
foreground = "#a9b1d6"
color1 = "#f7768e"
color2 = "#9ece6a"
color3 = "#e0af68"
color5 = "#ad8ee6"
color6 = "#449dab"
color7 = "#787c99"
color8 = "#444b6a"
`)

	theme, err := LoadOmarchyWithOptions(Options{OmarchyThemeDir: themeDir})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if theme.Name != "Tokyo Night" {
		t.Fatalf("expected Tokyo Night, got %q", theme.Name)
	}
	if theme.Palette.Accent != "#7aa2f7" {
		t.Fatalf("expected Omarchy accent, got %q", theme.Palette.Accent)
	}
}

func TestSyncOmarchyWithParentLayoutWritesCache(t *testing.T) {
	tmpDir := t.TempDir()
	omarchyCurrent := filepath.Join(tmpDir, "omarchy", "current")
	themeDir := filepath.Join(omarchyCurrent, "theme")
	cachePath := filepath.Join(tmpDir, "gitf", "theme.json")
	writeFile(t, filepath.Join(omarchyCurrent, "theme.name"), "Tokyo Night\n")
	writeFile(t, filepath.Join(themeDir, "colors.toml"), `
accent = "#7aa2f7"
foreground = "#a9b1d6"
color1 = "#f7768e"
color2 = "#9ece6a"
color3 = "#e0af68"
color5 = "#ad8ee6"
color6 = "#449dab"
color7 = "#787c99"
color8 = "#444b6a"
`)

	cache, err := SyncOmarchyWithOptions(Options{OmarchyThemeDir: themeDir, CachePath: cachePath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cache.Name != "Tokyo Night" {
		t.Fatalf("expected cache name, got %q", cache.Name)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file, got %v", err)
	}
}

func TestExplicitOmarchyFailureReturnsDefaultWithWarning(t *testing.T) {
	theme, warning := LoadWithOptions(config.ThemeConfig{Mode: "omarchy"}, Options{
		OmarchyThemeDir: filepath.Join(t.TempDir(), "missing"),
		CachePath:       filepath.Join(t.TempDir(), "theme.json"),
	})
	if warning == nil {
		t.Fatal("expected warning")
	}
	if theme.Name != Default().Name {
		t.Fatalf("expected default theme, got %q", theme.Name)
	}
}

func TestLoadFileAcceptsFullCacheWithEmptyPalette(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "theme.json")
	writeFile(t, path, `{
		"source": "manual",
		"name": "Custom",
		"synced_at": "2026-06-29T00:00:00Z",
		"palette": {}
	}`)

	theme, err := LoadFile(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if theme.Name != "Custom" {
		t.Fatalf("expected name Custom, got %q", theme.Name)
	}
	if theme.Source != "manual" {
		t.Fatalf("expected source manual, got %q", theme.Source)
	}
	def := Default()
	if theme.Palette.Accent != def.Palette.Accent {
		t.Fatalf("expected default accent %q, got %q", def.Palette.Accent, theme.Palette.Accent)
	}
	if theme.Palette.Foreground != def.Palette.Foreground {
		t.Fatalf("expected default foreground %q, got %q", def.Palette.Foreground, theme.Palette.Foreground)
	}
	if theme.Palette.Success != def.Palette.Success {
		t.Fatalf("expected default success %q, got %q", def.Palette.Success, theme.Palette.Success)
	}
}

func TestLoadFileAcceptsFullCacheWithoutPaletteField(t *testing.T) {
	tmpDir := t.TempDir()
	path := filepath.Join(tmpDir, "theme.json")
	writeFile(t, path, `{
		"source": "omarchy",
		"name": "Tokyo Night",
		"synced_at": "2026-06-29T00:00:00Z"
	}`)

	theme, err := LoadFile(path)
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if theme.Name != "Tokyo Night" {
		t.Fatalf("expected name Tokyo Night, got %q", theme.Name)
	}
	if theme.Source != "omarchy" {
		t.Fatalf("expected source omarchy, got %q", theme.Source)
	}
	def := Default()
	if theme.Palette.Accent != def.Palette.Accent {
		t.Fatalf("expected default accent %q, got %q", def.Palette.Accent, theme.Palette.Accent)
	}
}

func TestLoadAutoDetectsStaleCacheByMtime(t *testing.T) {
	tmpDir := t.TempDir()
	omarchyDir := filepath.Join(tmpDir, "omarchy", "current", "theme")
	cachePath := filepath.Join(tmpDir, "gitf", "theme.json")
	writeFile(t, filepath.Join(omarchyDir, "theme.name"), "Tokyo Night\n")
	writeFile(t, filepath.Join(omarchyDir, "colors.toml"), `
accent = "#222222"
foreground = "#a9b1d6"
color1 = "#f7768e"
color2 = "#9ece6a"
color3 = "#e0af68"
color5 = "#ad8ee6"
color6 = "#449dab"
color7 = "#787c99"
color8 = "#444b6a"
`)
	writeFile(t, cachePath, `{
		"source": "omarchy",
		"name": "Tokyo Night",
		"synced_at": "2026-06-28T00:00:00Z",
		"palette": {"accent": "#111111"}
	}`)
	// Set cache mtime older than colors.toml mtime to simulate staleness
	if err := os.Chtimes(cachePath, time.Now(), time.Now().Add(-time.Hour)); err != nil {
		t.Fatalf("failed to set cache mtime: %v", err)
	}
	// Ensure colors.toml has current mtime (fresh)
	if err := os.Chtimes(filepath.Join(omarchyDir, "colors.toml"), time.Now(), time.Now()); err != nil {
		t.Fatalf("failed to set colors mtime: %v", err)
	}

	theme, warning := LoadWithOptions(config.ThemeConfig{Mode: "auto"}, Options{
		OmarchyThemeDir: omarchyDir,
		CachePath:       cachePath,
	})
	if warning != nil {
		t.Fatalf("expected no warning, got %v", warning)
	}
	if theme.Palette.Accent != "#222222" {
		t.Fatalf("expected new accent from colors.toml, got %q", theme.Palette.Accent)
	}
	// Verify cache was rewritten with new values
	data, err := os.ReadFile(cachePath)
	if err != nil {
		t.Fatalf("expected cache update, got %v", err)
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		t.Fatalf("expected valid cache, got %v", err)
	}
	if cache.Palette.Accent != "#222222" {
		t.Fatalf("expected updated cache accent, got %q", cache.Palette.Accent)
	}
}

func TestSyncOmarchyWritesCache(t *testing.T) {
	tmpDir := t.TempDir()
	omarchyDir := filepath.Join(tmpDir, "omarchy", "current", "theme")
	cachePath := filepath.Join(tmpDir, "gitf", "theme.json")
	writeFile(t, filepath.Join(omarchyDir, "theme.name"), "Tokyo Night\n")
	writeFile(t, filepath.Join(omarchyDir, "colors.toml"), `
accent = "#7aa2f7"
foreground = "#a9b1d6"
color1 = "#f7768e"
color2 = "#9ece6a"
color3 = "#e0af68"
color5 = "#ad8ee6"
color6 = "#449dab"
color7 = "#787c99"
color8 = "#444b6a"
`)

	cache, err := SyncOmarchyWithOptions(Options{OmarchyThemeDir: omarchyDir, CachePath: cachePath})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if cache.Name != "Tokyo Night" {
		t.Fatalf("expected cache name, got %q", cache.Name)
	}
	if _, err := os.Stat(cachePath); err != nil {
		t.Fatalf("expected cache file, got %v", err)
	}
}
