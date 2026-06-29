package theme

import (
	"fmt"
	"os"
	"path/filepath"
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
		themeDir, dirErr := resolveOmarchyThemeDir(opts)
		if dirErr == nil {
			currentName, nameErr := readOmarchyThemeName(themeDir)
			if nameErr == nil && cache.Name == currentName {
				stale := false
				cachePath, pathErr := resolveCachePath(opts)
				colorsPath := filepath.Join(themeDir, "colors.toml")
				if pathErr == nil {
					if cacheInfo, cacheStatErr := os.Stat(cachePath); cacheStatErr == nil {
						if colorsInfo, colorsStatErr := os.Stat(colorsPath); colorsStatErr == nil {
							if colorsInfo.ModTime().After(cacheInfo.ModTime()) {
								stale = true
							}
						}
					}
				}
				if !stale {
					return Theme{Source: cache.Source, Name: cache.Name, Palette: mergePalette(Default().Palette, cache.Palette)}, nil
				}
			}
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
