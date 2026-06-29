# Omarchy Theme Integration Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add a generic theme system so `gitf` follows Omarchy colors by default, supports manual theme files, and refreshes the main TUI and setup wizard while open.

**Architecture:** Add theme configuration to `internal/config`, put parsing/cache/provider logic in a new `internal/theme` package, expose `gitf theme sync-omarchy` through Cobra, and centralize Lipgloss styles in `internal/ui/styles.go`. The UI receives a theme loader and refreshes styles every 2 seconds with Bubbletea ticks.

**Tech Stack:** Go 1.25.5, Go modules, Cobra, Bubbletea, Lipgloss, standard library JSON/TOML-line parsing, test runner `go test ./...`, verification through `go test`, `go vet`, and `go build`.

## Global Constraints

- Do not commit unless the user explicitly asks.
- No new runtime dependencies for theme loading or file watching.
- Theme mode defaults to `auto` for existing configs.
- Invalid theme modes normalize to `auto`.
- Supported theme modes are exactly `auto`, `default`, `omarchy`, and `file`.
- Only `#RRGGBB` colors are valid in theme files and Omarchy colors.
- Invalid or missing colors are ignored and filled from the built-in default theme.
- `auto` silently falls back to the built-in theme when Omarchy is unavailable.
- Explicit `omarchy` and `file` modes fall back to the built-in theme with a discreet warning.
- `gitf theme sync-omarchy` must print a clear stderr error and exit non-zero on failure.
- Main TUI and setup wizard must poll every 2 seconds for theme changes.
- UI code should use semantic theme colors through `internal/ui/styles.go`, not scattered hardcoded Lipgloss colors.
- Do not edit files under `~/.local/share/omarchy/`.

---

## File Structure

- Modify `internal/config/config.go`
  - Add `ThemeConfig`, theme mode constants, and normalization helpers.
  - Add `Theme ThemeConfig` to `Config`.

- Modify `internal/config/config_test.go`
  - Add tests for missing, empty, invalid, and valid theme modes.

- Create `internal/theme/theme.go`
  - Define `Palette`, `Theme`, `Cache`, `Warning`, `Options`, default palette, and merge/validation helpers.

- Create `internal/theme/omarchy.go`
  - Read Omarchy `theme.name` (actual layout: `~/.config/omarchy/current/theme.name`, parent of `theme/`; legacy fallback: `~/.config/omarchy/current/theme/theme.name`) and `colors.toml`.
  - Parse `key = "#RRGGBB"` lines.
  - Map Omarchy colors to semantic palette.

- Create `internal/theme/cache.go`
  - Read/write `~/.config/gitf/theme.json`.
  - Load full and flat JSON theme formats.
  - Validate Omarchy cache by theme name.

- Create `internal/theme/theme_test.go`
  - Cover parser, mapping, cache/file formats, merge behavior, stale cache, and sync.

- Modify `cmd/gitf/main.go`
  - Add `theme sync-omarchy` subcommand.
  - Load theme before main TUI and setup wizard.
  - Pass theme state into `ui.Run` and `ui.RunSetup`.

- Create `internal/ui/styles.go`
  - Convert `theme.Theme` to semantic Lipgloss styles/colors.

- Modify `internal/ui/ui.go`
  - Accept theme state.
  - Replace hardcoded colors with `Styles`.
  - Add 2 second theme polling and warning footer.

- Modify `internal/ui/setup.go`
  - Accept theme state.
  - Replace hardcoded colors with `Styles`.
  - Add 2 second theme polling and warning text.

- Modify `README.md`
  - Document theme config modes, manual file format, sync command, and Omarchy hook line.

---

### Task 1: Config Theme Model

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Produces: `type ThemeMode string`
- Produces constants: `ThemeModeAuto`, `ThemeModeDefault`, `ThemeModeOmarchy`, `ThemeModeFile`
- Produces: `type ThemeConfig struct { Mode string; Path string }`
- Produces: `func NormalizeThemeMode(mode string) ThemeMode`
- Produces: `func (c *Config) GetThemeConfig() ThemeConfig`
- Consumed by Tasks 2, 3, 4, and 5.

- [ ] **Step 1: Write failing config tests**

Append to `internal/config/config_test.go`:

```go
func TestDefaultConfig_ThemeDefaultsToAuto(t *testing.T) {
	cfg, err := DefaultConfig()
	assertNoError(t, err)

	themeCfg := cfg.GetThemeConfig()
	assertEqual(t, string(ThemeModeAuto), themeCfg.Mode, "theme mode")
}

func TestLoad_MissingThemeDefaultsToAuto(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	jsonData := []byte(`{
		"editor": "nvim",
		"search_paths": ["/home/user/dev"]
	}`)
	assertNoError(t, os.WriteFile(configFile, jsonData, 0644))

	cfg, err := load(configFile)
	assertNoError(t, err)

	themeCfg := cfg.GetThemeConfig()
	assertEqual(t, string(ThemeModeAuto), themeCfg.Mode, "missing theme mode")
}

func TestLoad_InvalidThemeModeDefaultsToAuto(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	jsonData := []byte(`{
		"editor": "nvim",
		"search_paths": ["/home/user/dev"],
		"theme": {"mode": "bad-mode"}
	}`)
	assertNoError(t, os.WriteFile(configFile, jsonData, 0644))

	cfg, err := load(configFile)
	assertNoError(t, err)

	themeCfg := cfg.GetThemeConfig()
	assertEqual(t, string(ThemeModeAuto), themeCfg.Mode, "invalid theme mode")
}

func TestLoad_ValidThemeModesPreserved(t *testing.T) {
	tests := []struct {
		name string
		mode string
	}{
		{name: "auto", mode: string(ThemeModeAuto)},
		{name: "default", mode: string(ThemeModeDefault)},
		{name: "omarchy", mode: string(ThemeModeOmarchy)},
		{name: "file", mode: string(ThemeModeFile)},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpDir := t.TempDir()
			configFile := filepath.Join(tmpDir, "config.json")

			jsonData := []byte(fmt.Sprintf(`{
				"editor": "nvim",
				"search_paths": ["/home/user/dev"],
				"theme": {"mode": %q, "path": "~/theme.json"}
			}`, tt.mode))
			assertNoError(t, os.WriteFile(configFile, jsonData, 0644))

			cfg, err := load(configFile)
			assertNoError(t, err)

			themeCfg := cfg.GetThemeConfig()
			assertEqual(t, tt.mode, themeCfg.Mode, "theme mode")
			assertEqual(t, "~/theme.json", themeCfg.Path, "theme path")
		})
	}
}
```

Add `fmt` to the test imports.

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/config
```

Expected: FAIL because `ThemeModeAuto`, `ThemeModeDefault`, `ThemeModeOmarchy`, `ThemeModeFile`, and `GetThemeConfig` are undefined.

- [ ] **Step 3: Implement config theme model**

In `internal/config/config.go`, add after tmux constants:

```go
type ThemeMode string

const (
	ThemeModeAuto    ThemeMode = "auto"
	ThemeModeDefault ThemeMode = "default"
	ThemeModeOmarchy ThemeMode = "omarchy"
	ThemeModeFile    ThemeMode = "file"
)

type ThemeConfig struct {
	Mode string `json:"mode,omitempty"`
	Path string `json:"path,omitempty"`
}
```

Update `Config`:

```go
type Config struct {
	Editor                string      `json:"editor"`
	SearchPaths           []string    `json:"search_paths"`
	FileManager           string      `json:"file_manager,omitempty"`
	Terminal              string      `json:"terminal,omitempty"`
	TmuxDefaultOpenAction string      `json:"tmux_default_open_action,omitempty"`
	Theme                 ThemeConfig `json:"theme,omitempty"`
}
```

Add helpers near existing normalization helpers:

```go
func NormalizeThemeMode(mode string) ThemeMode {
	switch ThemeMode(mode) {
	case ThemeModeAuto, ThemeModeDefault, ThemeModeOmarchy, ThemeModeFile:
		return ThemeMode(mode)
	default:
		return ThemeModeAuto
	}
}

func (c *Config) GetThemeConfig() ThemeConfig {
	if c == nil {
		return ThemeConfig{Mode: string(ThemeModeAuto)}
	}
	return ThemeConfig{
		Mode: string(NormalizeThemeMode(c.Theme.Mode)),
		Path: c.Theme.Path,
	}
}
```

Update `defaultConfig()` return value:

```go
		Theme:                 ThemeConfig{Mode: string(ThemeModeAuto)},
```

Update `load(configPath string)` after tmux normalization:

```go
	config.Theme = config.GetThemeConfig()
```

- [ ] **Step 4: Run config tests and verify pass**

Run:

```bash
go test ./internal/config
```

Expected: PASS.

- [ ] **Step 5: Format config package**

Run:

```bash
go fmt ./internal/config
```

Expected: no errors.

---

### Task 2: Theme Package Core, Omarchy Provider, Cache, and File Loader

**Files:**
- Create: `internal/theme/theme.go`
- Create: `internal/theme/omarchy.go`
- Create: `internal/theme/cache.go`
- Create: `internal/theme/theme_test.go`

**Interfaces:**
- Consumes: `config.ThemeConfig`, `config.ThemeModeAuto`, `config.ThemeModeDefault`, `config.ThemeModeOmarchy`, `config.ThemeModeFile`
- Produces: `type Palette struct`
- Produces: `type Theme struct`
- Produces: `type Warning struct`
- Produces: `type Cache struct`
- Produces: `type Options struct`
- Produces: `func Default() Theme`
- Produces: `func Load(cfg config.ThemeConfig) (Theme, *Warning)`
- Produces: `func LoadWithOptions(cfg config.ThemeConfig, opts Options) (Theme, *Warning)`
- Produces: `func LoadOmarchy() (Theme, error)`
- Produces: `func LoadFile(path string) (Theme, error)`
- Produces: `func SyncOmarchy() (Cache, error)`
- Produces: `func CachePath() (string, error)`
- Produces: `func ThemeState(opts Options) string`

- [ ] **Step 1: Write failing theme tests**

Create `internal/theme/theme_test.go`:

```go
package theme

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/theme
```

Expected: FAIL because the package does not exist or symbols are undefined.

- [ ] **Step 3: Implement theme core**

Create `internal/theme/theme.go`:

```go
package theme

import (
	"fmt"
	"regexp"
	"time"

	"github.com/tiagokriok/Git-Fuzzy/internal/config"
)

type Palette struct {
	Accent     string `json:"accent,omitempty"`
	Foreground string `json:"foreground,omitempty"`
	Muted      string `json:"muted,omitempty"`
	Border     string `json:"border,omitempty"`
	Selection  string `json:"selection,omitempty"`
	Success    string `json:"success,omitempty"`
	Warning    string `json:"warning,omitempty"`
	Danger     string `json:"danger,omitempty"`
	Info       string `json:"info,omitempty"`
	Branch     string `json:"branch,omitempty"`
	Stash      string `json:"stash,omitempty"`
	Renamed    string `json:"renamed,omitempty"`
	Copied     string `json:"copied,omitempty"`
	Untracked  string `json:"untracked,omitempty"`
}

type Theme struct {
	Source  string
	Name    string
	Palette Palette
}

type Warning struct {
	Message string
}

type Cache struct {
	Source   string  `json:"source"`
	Name     string  `json:"name"`
	SyncedAt string  `json:"synced_at"`
	Palette  Palette `json:"palette"`
}

type Options struct {
	OmarchyThemeDir string
	CachePath       string
}

var hexColorPattern = regexp.MustCompile(`^#[0-9a-fA-F]{6}$`)

func Default() Theme {
	return Theme{
		Source: "default",
		Name:   "Default",
		Palette: Palette{
			Accent:     "#d75fd7",
			Foreground: "#c0c0c0",
			Muted:      "#808080",
			Border:     "#808080",
			Selection:  "#5fd75f",
			Success:    "#5fd75f",
			Warning:    "#ffd75f",
			Danger:     "#ff5f5f",
			Info:       "#5fafd7",
			Branch:     "#5fd75f",
			Stash:      "#d75fd7",
			Renamed:    "#d75fd7",
			Copied:     "#5fafd7",
			Untracked:  "#5fafd7",
		},
	}
}

func Load(cfg config.ThemeConfig) (Theme, *Warning) {
	return LoadWithOptions(cfg, Options{})
}

func LoadWithOptions(cfg config.ThemeConfig, opts Options) (Theme, *Warning) {
	mode := config.NormalizeThemeMode(cfg.Mode)
	switch mode {
	case config.ThemeModeDefault:
		return Default(), nil
	case config.ThemeModeOmarchy:
		theme, err := LoadOmarchyWithOptions(opts)
		if err != nil {
			return Default(), &Warning{Message: fmt.Sprintf("theme: failed to load Omarchy theme, using default: %v", err)}
		}
		return theme, nil
	case config.ThemeModeFile:
		theme, err := LoadFile(cfg.Path)
		if err != nil {
			return Default(), &Warning{Message: fmt.Sprintf("theme: failed to load theme file, using default: %v", err)}
		}
		return theme, nil
	default:
		theme, err := loadAuto(opts)
		if err != nil {
			return Default(), nil
		}
		return theme, nil
	}
}

func loadAuto(opts Options) (Theme, error) {
	cache, err := readCache(opts)
	if err == nil && cache.Source == "omarchy" {
		currentName, nameErr := readOmarchyThemeName(resolveOmarchyThemeDir(opts))
		if nameErr == nil && cache.Name == currentName {
			return Theme{Source: cache.Source, Name: cache.Name, Palette: mergePalette(Default().Palette, cache.Palette)}, nil
		}
	}

	theme, err := LoadOmarchyWithOptions(opts)
	if err != nil {
		return Theme{}, err
	}
	_ = writeCache(opts, Cache{
		Source:   theme.Source,
		Name:     theme.Name,
		SyncedAt: time.Now().UTC().Format(time.RFC3339),
		Palette:  theme.Palette,
	})
	return theme, nil
}

func isValidHexColor(value string) bool {
	return hexColorPattern.MatchString(value)
}

func mergePalette(base Palette, override Palette) Palette {
	merged := base
	apply := func(value string, set func(string)) {
		if isValidHexColor(value) {
			set(value)
		}
	}
	apply(override.Accent, func(v string) { merged.Accent = v })
	apply(override.Foreground, func(v string) { merged.Foreground = v })
	apply(override.Muted, func(v string) { merged.Muted = v })
	apply(override.Border, func(v string) { merged.Border = v })
	apply(override.Selection, func(v string) { merged.Selection = v })
	apply(override.Success, func(v string) { merged.Success = v })
	apply(override.Warning, func(v string) { merged.Warning = v })
	apply(override.Danger, func(v string) { merged.Danger = v })
	apply(override.Info, func(v string) { merged.Info = v })
	apply(override.Branch, func(v string) { merged.Branch = v })
	apply(override.Stash, func(v string) { merged.Stash = v })
	apply(override.Renamed, func(v string) { merged.Renamed = v })
	apply(override.Copied, func(v string) { merged.Copied = v })
	apply(override.Untracked, func(v string) { merged.Untracked = v })
	return merged
}
```

- [ ] **Step 4: Implement Omarchy provider**

Create `internal/theme/omarchy.go`:

```go
package theme

import (
	"bufio"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func LoadOmarchy() (Theme, error) {
	return LoadOmarchyWithOptions(Options{})
}

func LoadOmarchyWithOptions(opts Options) (Theme, error) {
	themeDir := resolveOmarchyThemeDir(opts)
	name, err := readOmarchyThemeName(themeDir)
	if err != nil {
		return Theme{}, err
	}

	file, err := os.Open(filepath.Join(themeDir, "colors.toml"))
	if err != nil {
		return Theme{}, fmt.Errorf("failed to read Omarchy colors: %w", err)
	}
	defer file.Close()

	colors, err := parseOmarchyColors(file)
	if err != nil {
		return Theme{}, err
	}

	return Theme{
		Source:  "omarchy",
		Name:    name,
		Palette: mergePalette(Default().Palette, mapOmarchyPalette(colors)),
	}, nil
}

func resolveOmarchyThemeDir(opts Options) string {
	if opts.OmarchyThemeDir != "" {
		return opts.OmarchyThemeDir
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return ""
	}
	return filepath.Join(home, ".config", "omarchy", "current", "theme")
}

func readOmarchyThemeName(themeDir string) (string, error) {
	data, err := os.ReadFile(filepath.Join(themeDir, "theme.name"))
	if err != nil {
		return "", fmt.Errorf("failed to read Omarchy theme name: %w", err)
	}
	name := strings.TrimSpace(string(data))
	if name == "" {
		return "", fmt.Errorf("Omarchy theme name is empty")
	}
	return name, nil
}

func parseOmarchyColors(r io.Reader) (map[string]string, error) {
	colors := map[string]string{}
	scanner := bufio.NewScanner(r)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, "=", 2)
		if len(parts) != 2 {
			continue
		}
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"")
		if key != "" && isValidHexColor(value) {
			colors[key] = value
		}
	}
	if err := scanner.Err(); err != nil {
		return nil, fmt.Errorf("failed to parse Omarchy colors: %w", err)
	}
	return colors, nil
}

func mapOmarchyPalette(colors map[string]string) Palette {
	accent := firstColor(colors, "accent", "color4")
	return Palette{
		Accent:     accent,
		Foreground: firstColor(colors, "foreground"),
		Muted:      firstColor(colors, "color7"),
		Border:     firstColor(colors, "color8"),
		Selection:  firstColor(colors, "selection_background", "accent", "color4"),
		Success:    firstColor(colors, "color2"),
		Warning:    firstColor(colors, "color3"),
		Danger:     firstColor(colors, "color1"),
		Info:       firstColor(colors, "color6"),
		Branch:     firstColor(colors, "color2"),
		Stash:      firstColor(colors, "color5"),
		Renamed:    firstColor(colors, "color5"),
		Copied:     firstColor(colors, "color6"),
		Untracked:  firstColor(colors, "color6"),
	}
}

func firstColor(colors map[string]string, keys ...string) string {
	for _, key := range keys {
		if isValidHexColor(colors[key]) {
			return colors[key]
		}
	}
	return ""
}
```

- [ ] **Step 5: Implement cache and file loader**

Create `internal/theme/cache.go`:

```go
package theme

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

func CachePath() (string, error) {
	configDir, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get config directory: %w", err)
	}
	return filepath.Join(configDir, "gitf", "theme.json"), nil
}

func LoadFile(path string) (Theme, error) {
	if path == "" {
		return Theme{}, fmt.Errorf("theme file path is empty")
	}
	resolved, err := expandHome(path)
	if err != nil {
		return Theme{}, err
	}
	data, err := os.ReadFile(resolved)
	if err != nil {
		return Theme{}, fmt.Errorf("failed to read theme file: %w", err)
	}
	return decodeThemeJSON(data)
}

func SyncOmarchy() (Cache, error) {
	return SyncOmarchyWithOptions(Options{})
}

func SyncOmarchyWithOptions(opts Options) (Cache, error) {
	theme, err := LoadOmarchyWithOptions(opts)
	if err != nil {
		return Cache{}, err
	}
	cache := Cache{
		Source:   "omarchy",
		Name:     theme.Name,
		SyncedAt: time.Now().UTC().Format(time.RFC3339),
		Palette:  theme.Palette,
	}
	if err := writeCache(opts, cache); err != nil {
		return Cache{}, err
	}
	return cache, nil
}

func readCache(opts Options) (Cache, error) {
	path, err := resolveCachePath(opts)
	if err != nil {
		return Cache{}, err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return Cache{}, err
	}
	var cache Cache
	if err := json.Unmarshal(data, &cache); err != nil {
		return Cache{}, fmt.Errorf("failed to parse theme cache: %w", err)
	}
	cache.Palette = mergePalette(Default().Palette, cache.Palette)
	return cache, nil
}

func writeCache(opts Options, cache Cache) error {
	path, err := resolveCachePath(opts)
	if err != nil {
		return err
	}
	data, err := json.MarshalIndent(cache, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to encode theme cache: %w", err)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("failed to create theme cache directory: %w", err)
	}
	if err := os.WriteFile(path, data, 0644); err != nil {
		return fmt.Errorf("failed to write theme cache: %w", err)
	}
	return nil
}

func resolveCachePath(opts Options) (string, error) {
	if opts.CachePath != "" {
		return opts.CachePath, nil
	}
	return CachePath()
}

func decodeThemeJSON(data []byte) (Theme, error) {
	var cache Cache
	if err := json.Unmarshal(data, &cache); err == nil && cache.Palette != (Palette{}) {
		name := cache.Name
		if name == "" {
			name = "File"
		}
		source := cache.Source
		if source == "" {
			source = "file"
		}
		return Theme{Source: source, Name: name, Palette: mergePalette(Default().Palette, cache.Palette)}, nil
	}

	var palette Palette
	if err := json.Unmarshal(data, &palette); err != nil {
		return Theme{}, fmt.Errorf("failed to parse theme file: %w", err)
	}
	return Theme{Source: "file", Name: "File", Palette: mergePalette(Default().Palette, palette)}, nil
}

func expandHome(path string) (string, error) {
	if path == "~" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return home, nil
	}
	if len(path) > 2 && path[:2] == "~/" {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", fmt.Errorf("failed to get home directory: %w", err)
		}
		return filepath.Join(home, path[2:]), nil
	}
	return path, nil
}

func ThemeState(opts Options) string {
	var parts []string
	themeDir := resolveOmarchyThemeDir(opts)
	if name, err := readOmarchyThemeName(themeDir); err == nil {
		parts = append(parts, "omarchy-name="+name)
	}
	if info, err := os.Stat(filepath.Join(themeDir, "colors.toml")); err == nil {
		parts = append(parts, fmt.Sprintf("omarchy-colors=%d", info.ModTime().UnixNano()))
	}
	if cachePath, err := resolveCachePath(opts); err == nil {
		if info, statErr := os.Stat(cachePath); statErr == nil {
			parts = append(parts, fmt.Sprintf("cache=%d", info.ModTime().UnixNano()))
		}
	}
	return strings.Join(parts, "|")
}
```

- [ ] **Step 6: Run theme tests and verify pass**

Run:

```bash
go test ./internal/theme
```

Expected: PASS.

- [ ] **Step 7: Format theme package**

Run:

```bash
go fmt ./internal/theme
```

Expected: no errors.

---

### Task 3: CLI Command and Theme Loading Integration

**Files:**
- Modify: `cmd/gitf/main.go`
- Modify: `internal/ui/ui.go`
- Modify: `internal/ui/setup.go`

**Interfaces:**
- Consumes: `theme.Load`, `theme.SyncOmarchy`, `theme.Theme`, `theme.Warning`
- Produces: `gitf theme sync-omarchy`
- Produces temporary UI signatures for later style tasks:
  - `ui.Run(repos []scanner.Repository, cfg *config.Config, appTheme theme.Theme, warning *theme.Warning) (*scanner.Repository, error)`
  - `ui.RunSetup(appTheme theme.Theme, warning *theme.Warning) (*config.Config, error)`

- [ ] **Step 1: Add CLI sync command test through helper**

Create `cmd/gitf/theme_command_test.go`:

```go
package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
)

func TestRunThemeSyncOmarchyPrintsSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runThemeSyncOmarchy(&stdout, &stderr, func() (theme.Cache, error) {
		return theme.Cache{Name: "Tokyo Night"}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Synced Omarchy theme") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunThemeSyncOmarchyReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runThemeSyncOmarchy(&stdout, &stderr, func() (theme.Cache, error) {
		return theme.Cache{}, errors.New("missing colors")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing colors") {
		t.Fatalf("expected stderr error, got %q", stderr.String())
	}
}
```

- [ ] **Step 2: Run command tests and verify failure**

Run:

```bash
go test ./cmd/gitf
```

Expected: FAIL because `runThemeSyncOmarchy` is undefined and UI signatures are not updated yet.

- [ ] **Step 3: Add CLI command helper and Cobra subcommand**

In `cmd/gitf/main.go`, add imports:

```go
	"io"

	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
```

In `main()`, before `rootCmd.Execute()`, add:

```go
	themeCmd := &cobra.Command{
		Use:   "theme",
		Short: "Theme utilities",
	}
	syncOmarchyCmd := &cobra.Command{
		Use:   "sync-omarchy",
		Short: "Sync the current Omarchy theme into gitf cache",
		RunE: func(cmd *cobra.Command, args []string) error {
			return runThemeSyncOmarchy(cmd.OutOrStdout(), cmd.ErrOrStderr(), theme.SyncOmarchy)
		},
	}
	themeCmd.AddCommand(syncOmarchyCmd)
	rootCmd.AddCommand(themeCmd)
```

Add helper near `handleSetup`:

```go
func runThemeSyncOmarchy(stdout io.Writer, stderr io.Writer, syncFn func() (theme.Cache, error)) error {
	cache, err := syncFn()
	if err != nil {
		fmt.Fprintf(stderr, "failed to sync Omarchy theme: %v\n", err)
		return err
	}
	cachePath, pathErr := theme.CachePath()
	if pathErr != nil {
		fmt.Fprintf(stdout, "Synced Omarchy theme %q\n", cache.Name)
		return nil
	}
	fmt.Fprintf(stdout, "Synced Omarchy theme %q to %s\n", cache.Name, cachePath)
	return nil
}
```

- [ ] **Step 4: Load theme in TUI and setup flows**

In `runTUI`, when config is missing, replace `ui.RunSetup()` with:

```go
			setupTheme, setupWarning := theme.Load(config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
			cfg, err = ui.RunSetup(setupTheme, setupWarning)
```

After config is loaded and before scanning repos, add:

```go
	appTheme, themeWarning := theme.Load(cfg.GetThemeConfig())
```

Replace `ui.Run(repos, cfg)` with:

```go
	selected, err := ui.Run(repos, cfg, appTheme, themeWarning)
```

In `handleSetup`, before `ui.RunSetup`, add:

```go
	setupTheme, themeWarning := theme.Load(config.ThemeConfig{Mode: string(config.ThemeModeAuto)})
```

Replace `ui.RunSetup()` with:

```go
	newCfg, err := ui.RunSetup(setupTheme, themeWarning)
```

- [ ] **Step 5: Temporarily update UI and setup signatures**

In `internal/ui/ui.go`, add import:

```go
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
```

Extend `Model`:

```go
	theme        theme.Theme
	themeWarning *theme.Warning
```

Change `NewModel` signature:

```go
func NewModel(repos []scanner.Repository, cfg *config.Config, appTheme theme.Theme, warning *theme.Warning) Model
```

Set fields in the return:

```go
		theme:        appTheme,
		themeWarning: warning,
```

Change `Run` signature and model creation:

```go
func Run(repos []scanner.Repository, cfg *config.Config, appTheme theme.Theme, warning *theme.Warning) (*scanner.Repository, error) {
	selectedRepository = nil

	model := NewModel(repos, cfg, appTheme, warning)
```

In `internal/ui/setup.go`, add import:

```go
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
```

Extend `SetupModel`:

```go
	theme        theme.Theme
	themeWarning *theme.Warning
```

Change `RunSetup` signature:

```go
func RunSetup(appTheme theme.Theme, warning *theme.Warning) (*config.Config, error)
```

Set model fields:

```go
		theme:              appTheme,
		themeWarning:       warning,
```

These fields are used by later tasks.

- [ ] **Step 6: Run command and full tests**

Run:

```bash
go test ./cmd/gitf ./internal/ui
```

Expected: PASS.

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 7: Format changed packages**

Run:

```bash
go fmt ./cmd/gitf ./internal/ui
```

Expected: no errors.

---

### Task 4: UI Semantic Styles and Main TUI Theme Refresh

**Files:**
- Create: `internal/ui/styles.go`
- Create: `internal/ui/styles_test.go`
- Modify: `internal/ui/ui.go`

**Interfaces:**
- Consumes: `theme.Theme`, `theme.Load`, `config.ThemeConfig`
- Produces: `type Styles struct`
- Produces: `func NewStyles(appTheme theme.Theme) Styles`
- Produces: `type themeRefreshMsg struct { theme theme.Theme; warning *theme.Warning; state string }`
- Produces: `func scheduleThemeRefresh(cfg config.ThemeConfig, state string) tea.Cmd`

- [ ] **Step 1: Write styles tests**

Create `internal/ui/styles_test.go`:

```go
package ui

import (
	"testing"

	"github.com/charmbracelet/lipgloss"
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
```

- [ ] **Step 2: Run UI tests and verify failure**

Run:

```bash
go test ./internal/ui -run 'TestNewStylesUsesThemeAccent|TestThemeWarningText'
```

Expected: FAIL because `NewStyles` and `themeWarningText` are undefined.

- [ ] **Step 3: Implement styles**

Create `internal/ui/styles.go`:

```go
package ui

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
)

type Styles struct {
	Title        lipgloss.Style
	Subtitle     lipgloss.Style
	Muted        lipgloss.Style
	MutedItalic  lipgloss.Style
	Selected     lipgloss.Style
	Border       lipgloss.Color
	Accent       lipgloss.Color
	Foreground   lipgloss.Color
	Success      lipgloss.Color
	Warning      lipgloss.Color
	Danger       lipgloss.Color
	Info         lipgloss.Color
	Branch       lipgloss.Color
	Stash        lipgloss.Color
	Renamed      lipgloss.Color
	Copied       lipgloss.Color
	Untracked    lipgloss.Color
	SearchBox    lipgloss.Style
	Panel        lipgloss.Style
	Error        lipgloss.Style
	Footer       lipgloss.Style
	FooterPadded lipgloss.Style
}

func NewStyles(appTheme theme.Theme) Styles {
	palette := theme.Default().Palette
	if appTheme.Palette != (theme.Palette{}) {
		palette = appTheme.Palette
	}
	accent := lipgloss.Color(palette.Accent)
	border := lipgloss.Color(palette.Border)
	muted := lipgloss.Color(palette.Muted)
	danger := lipgloss.Color(palette.Danger)
	return Styles{
		Title:        lipgloss.NewStyle().Bold(true).Foreground(accent),
		Subtitle:     lipgloss.NewStyle().Foreground(muted),
		Muted:        lipgloss.NewStyle().Foreground(muted),
		MutedItalic:  lipgloss.NewStyle().Foreground(muted).Italic(true),
		Selected:     lipgloss.NewStyle().Foreground(lipgloss.Color(palette.Selection)).Bold(true),
		Border:       border,
		Accent:       accent,
		Foreground:   lipgloss.Color(palette.Foreground),
		Success:      lipgloss.Color(palette.Success),
		Warning:      lipgloss.Color(palette.Warning),
		Danger:       danger,
		Info:         lipgloss.Color(palette.Info),
		Branch:       lipgloss.Color(palette.Branch),
		Stash:        lipgloss.Color(palette.Stash),
		Renamed:      lipgloss.Color(palette.Renamed),
		Copied:       lipgloss.Color(palette.Copied),
		Untracked:    lipgloss.Color(palette.Untracked),
		SearchBox:    lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(accent).Padding(0, 1).Align(lipgloss.Left),
		Panel:        lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(border).Padding(1),
		Error:        lipgloss.NewStyle().Foreground(danger),
		Footer:       lipgloss.NewStyle().Foreground(muted).Align(lipgloss.Center),
		FooterPadded: lipgloss.NewStyle().Foreground(muted).Padding(1, 0),
	}
}

func themeWarningText(warning *theme.Warning) string {
	if warning == nil {
		return ""
	}
	return warning.Message
}
```

- [ ] **Step 4: Add styles and theme refresh state to main model**

In `internal/ui/ui.go`, extend `Model`:

```go
	styles          Styles
	themeState      string
	themeConfig     config.ThemeConfig
```

Add message type near other message types:

```go
type themeRefreshMsg struct {
	theme   theme.Theme
	warning *theme.Warning
	state   string
}
```

Update `NewModel` return:

```go
		styles:       NewStyles(appTheme),
		themeConfig:  cfg.GetThemeConfig(),
		themeState:   theme.ThemeState(theme.Options{}),
		statusMessage: themeWarningText(warning),
```

Update `Init()` to batch git status and theme refresh:

```go
func (m Model) Init() tea.Cmd {
	var cmds []tea.Cmd
	cmds = append(cmds, scheduleThemeRefresh(m.themeConfig, m.themeState))
	if len(m.repositories) > 0 {
		cmds = append(cmds, m.fetchGitStatusAsync(m.repositories[0].Path))
	}
	return tea.Batch(cmds...)
}
```

Add to `Update` switch:

```go
	case themeRefreshMsg:
		if msg.state != "" && msg.state != m.themeState {
			m.theme = msg.theme
			m.themeWarning = msg.warning
			m.styles = NewStyles(msg.theme)
			m.themeState = msg.state
			if warningText := themeWarningText(msg.warning); warningText != "" {
				m.statusMessage = warningText
			}
		}
		return m, scheduleThemeRefresh(m.themeConfig, m.themeState)
```

Add function:

```go
func scheduleThemeRefresh(cfg config.ThemeConfig, previousState string) tea.Cmd {
	return tea.Tick(2*time.Second, func(t time.Time) tea.Msg {
		loadedTheme, warning := theme.Load(cfg)
		return themeRefreshMsg{
			theme:   loadedTheme,
			warning: warning,
			state:   theme.ThemeState(theme.Options{}),
		}
	})
}
```

- [ ] **Step 5: Replace main TUI hardcoded styles**

In `internal/ui/ui.go`, use `m.styles` in render methods. Required replacements:

- Search box style: use `m.styles.SearchBox.Width(searchBoxWidth)`.
- Selected repo style: use `m.styles.Selected`.
- Muted text: use `m.styles.Muted` or `m.styles.MutedItalic`.
- Panels: use `m.styles.Panel.Width(width).Height(...)`.
- Titles: use `m.styles.Title`.
- Errors: use `m.styles.Error`.
- Branch: use `lipgloss.NewStyle().Foreground(m.styles.Branch).Bold(true)`.
- Stats:
  - ahead and added use `m.styles.Success`
  - behind and deleted use `m.styles.Danger`
  - modified uses `m.styles.Warning`
  - untracked uses `m.styles.Untracked`
- File status colors map:

```go
		statusColors := map[string]lipgloss.Color{
			"M":  m.styles.Warning,
			"A":  m.styles.Success,
			"D":  m.styles.Danger,
			"R":  m.styles.Renamed,
			"C":  m.styles.Copied,
			"??": m.styles.Untracked,
		}
```

- Footer style: use `m.styles.Footer`.
- Session prompt styles: use `m.styles.Title`, `m.styles.SearchBox`, `m.styles.Error`, `m.styles.Muted`, `m.styles.Panel`.

- [ ] **Step 6: Run UI tests**

Run:

```bash
go test ./internal/ui
```

Expected: PASS.

- [ ] **Step 7: Format UI package**

Run:

```bash
go fmt ./internal/ui
```

Expected: no errors.

---

### Task 5: Setup Wizard Theme Styles and Refresh

**Files:**
- Modify: `internal/ui/setup.go`
- Modify: `internal/ui/styles_test.go`

**Interfaces:**
- Consumes: `Styles`, `NewStyles`, `theme.Load`, `theme.ThemeState`, `themeRefreshMsg`, `scheduleThemeRefresh`
- Produces setup wizard with themed styles and 2 second refresh.

- [ ] **Step 1: Write setup model theme test**

Append to `internal/ui/styles_test.go`:

```go
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
	model := newSetupModel(appTheme, &theme.Warning{Message: "theme warning"})
	if model.styles.Title.GetBold() != true {
		t.Fatal("expected bold title style")
	}
	if model.themeWarning == nil || model.themeWarning.Message != "theme warning" {
		t.Fatalf("expected theme warning")
	}
}
```

- [ ] **Step 2: Run focused test and verify failure**

Run:

```bash
go test ./internal/ui -run TestNewSetupModelUsesStyles
```

Expected: FAIL because `newSetupModel` and setup style fields are not implemented.

- [ ] **Step 3: Extract setup model constructor**

In `internal/ui/setup.go`, add fields to `SetupModel`:

```go
	styles      Styles
	themeState  string
	themeConfig config.ThemeConfig
```

Create constructor:

```go
func newSetupModel(appTheme theme.Theme, warning *theme.Warning) SetupModel {
	editorInput := textinput.New()
	editorInput.Placeholder = "e.g., vim, nvim, code, zed"
	editorInput.SetValue("nvim")
	editorInput.Focus()

	pathsInput := textinput.New()
	pathsInput.Placeholder = "e.g., ~/dev, ~/projects"

	return SetupModel{
		step:               0,
		editor:             editorInput,
		paths:              pathsInput,
		tmuxDefaultActions: config.TmuxOpenActionOptions(),
		theme:              appTheme,
		themeWarning:       warning,
		styles:             NewStyles(appTheme),
		themeConfig:        config.ThemeConfig{Mode: string(config.ThemeModeAuto)},
		themeState:         theme.ThemeState(theme.Options{}),
	}
}
```

Update `RunSetup` to use constructor:

```go
func RunSetup(appTheme theme.Theme, warning *theme.Warning) (*config.Config, error) {
	model := newSetupModel(appTheme, warning)
```

- [ ] **Step 4: Add setup theme refresh**

Update `SetupModel.Init()`:

```go
func (m SetupModel) Init() tea.Cmd {
	return scheduleThemeRefresh(m.themeConfig, m.themeState)
}
```

Add `themeRefreshMsg` case in `SetupModel.Update`:

```go
	case themeRefreshMsg:
		if msg.state != "" && msg.state != m.themeState {
			m.theme = msg.theme
			m.themeWarning = msg.warning
			m.styles = NewStyles(msg.theme)
			m.themeState = msg.state
		}
		return m, scheduleThemeRefresh(m.themeConfig, m.themeState)
```

- [ ] **Step 5: Replace setup hardcoded styles**

In `editorView`, replace local styles with:

```go
	title := m.styles.Title.Render("Git Fuzzy Setup")
	subtitle := m.styles.Subtitle.Render("Step 1 of 3: Editor")
	input := m.styles.SearchBox.Width(50).Render(m.editor.View())
	footer := m.styles.FooterPadded.Render("Enter: next | Ctrl+C: cancel")
```

Append warning text if present:

```go
	warning := themeWarningText(m.themeWarning)
	if warning != "" {
		return fmt.Sprintf("%s\n\n%s\n\n%s\n\nWhat's your preferred editor?\n%s\n\n%s", title, subtitle, m.styles.Error.Render(warning), input, footer)
	}
```

Apply equivalent style replacements to `pathsView` and `tmuxActionView`:

- `Title` for title.
- `Subtitle` for subtitle.
- `SearchBox.Width(50)` for input.
- `FooterPadded` for footer.
- `Selected` for selected tmux action.
- Plain unselected lines stay plain.

- [ ] **Step 6: Run setup/UI tests**

Run:

```bash
go test ./internal/ui
```

Expected: PASS.

- [ ] **Step 7: Format UI package**

Run:

```bash
go fmt ./internal/ui
```

Expected: no errors.

---

### Task 6: Documentation and Verification

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes complete behavior from Tasks 1 through 5.
- Produces user-facing documentation for theme config and Omarchy sync.

- [ ] **Step 1: Add README theme section**

Add a section after configuration examples or after Tmux Actions:

```markdown
### Theme Integration

`gitf` can follow your Omarchy theme automatically. Existing configs default to automatic theme detection:

```json
{
  "theme": {
    "mode": "auto"
  }
}
```

Supported modes:

```text
auto      Use Omarchy when available, otherwise use the built-in theme
default   Always use the built-in theme
omarchy   Require Omarchy colors, falling back to default with a warning if unavailable
file      Load a manual theme file from theme.path
```

Manual theme example:

```json
{
  "theme": {
    "mode": "file",
    "path": "~/.config/gitf/my-theme.json"
  }
}
```

Manual theme files can use a flat palette:

```json
{
  "accent": "#7aa2f7",
  "foreground": "#a9b1d6",
  "success": "#9ece6a"
}
```

Only `#RRGGBB` colors are accepted. Missing colors use built-in defaults.
```

- [ ] **Step 2: Add Omarchy sync command docs**

Add:

```markdown
#### Omarchy hook

Sync the current Omarchy theme into the `gitf` cache:

```bash
gitf theme sync-omarchy
```

To sync whenever Omarchy changes theme, add this to `~/.config/omarchy/hooks/theme-set`:

```bash
gitf theme sync-omarchy >/dev/null
```

`gitf` also checks for theme changes while open and refreshes within 2 seconds.
```

- [ ] **Step 3: Run full verification**

Run:

```bash
gofmt -l internal/config/config.go internal/config/config_test.go internal/theme/theme.go internal/theme/omarchy.go internal/theme/cache.go internal/theme/theme_test.go cmd/gitf/main.go cmd/gitf/theme_command_test.go internal/ui/styles.go internal/ui/styles_test.go internal/ui/ui.go internal/ui/setup.go
```

Expected: no output.

Run:

```bash
go test ./... -count=1
```

Expected: PASS.

Run:

```bash
go vet ./...
```

Expected: PASS.

Run:

```bash
go build ./...
```

Expected: PASS.

- [ ] **Step 4: Manual Omarchy smoke test**

Run:

```bash
gitf theme sync-omarchy
```

Expected output contains:

```text
Synced Omarchy theme
```

Then run:

```bash
gitf
```

Expected:

- TUI colors match active Omarchy theme.
- Change Omarchy theme in another terminal.
- TUI refreshes within 2 seconds.

- [ ] **Step 5: Final status check**

Run:

```bash
git status --short
git diff --stat
```

Expected: changes limited to config, theme package, command integration, UI styles, README, spec, and plan docs.

---

## Self-Review

- Spec coverage: Task 1 covers config. Task 2 covers default theme, Omarchy parsing, cache, file mode, auto fallback, stale cache, and sync. Task 3 covers Cobra command and theme loading into TUI/setup. Task 4 covers main TUI styles and polling. Task 5 covers setup wizard styles and polling. Task 6 covers docs and verification.
- Placeholder scan: no placeholder markers are intentionally present.
- Type consistency: `config.ThemeConfig`, `theme.Theme`, `theme.Warning`, `theme.Options`, `Styles`, `NewStyles`, and `scheduleThemeRefresh` are introduced before dependent tasks use them.
- Scope check: this is one cohesive feature because config, theme provider, CLI sync, and UI refresh are all required for the accepted Omarchy-following behavior.
