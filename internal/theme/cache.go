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
	if err := json.Unmarshal(data, &cache); err == nil && (cache.Source != "" || cache.Name != "" || cache.Palette != (Palette{})) {
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
	themeDir, err := resolveOmarchyThemeDir(opts)
	if err != nil {
		return ""
	}
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
