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
	themeDir, err := resolveOmarchyThemeDir(opts)
	if err != nil {
		return Theme{}, err
	}
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

func resolveOmarchyThemeDir(opts Options) (string, error) {
	if opts.OmarchyThemeDir != "" {
		return opts.OmarchyThemeDir, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", fmt.Errorf("failed to get home directory: %w", err)
	}
	return filepath.Join(home, ".config", "omarchy", "current", "theme"), nil
}

func readOmarchyThemeName(themeDir string) (string, error) {
	// Actual Omarchy layout: `theme.name` is a sibling of the `theme/`
	// directory (e.g. `~/.config/omarchy/current/theme.name`). Some layouts
	// (legacy and tests) keep it inside the theme directory. Try the parent
	// first, then fall back to the theme directory itself.
	candidates := []string{
		filepath.Join(filepath.Dir(themeDir), "theme.name"),
		filepath.Join(themeDir, "theme.name"),
	}
	var data []byte
	var lastErr error
	for _, path := range candidates {
		d, err := os.ReadFile(path)
		if err == nil {
			data = d
			lastErr = nil
			break
		}
		lastErr = err
	}
	if lastErr != nil {
		return "", fmt.Errorf("failed to read Omarchy theme name: %w", lastErr)
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
