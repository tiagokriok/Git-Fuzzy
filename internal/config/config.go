package config

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/tiagokriok/Git-Fuzzy/internal/platform"
)

type TmuxOpenAction string

const (
	TmuxOpenEditor         TmuxOpenAction = "editor"
	TmuxOpenWindow         TmuxOpenAction = "tmux-window"
	TmuxOpenVerticalPane   TmuxOpenAction = "tmux-vertical-pane"
	TmuxOpenHorizontalPane TmuxOpenAction = "tmux-horizontal-pane"
	TmuxOpenSession        TmuxOpenAction = "tmux-session"
)

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

type Config struct {
	Editor                string      `json:"editor"`
	SearchPaths           []string    `json:"search_paths"`
	FileManager           string      `json:"file_manager,omitempty"`
	Terminal              string      `json:"terminal,omitempty"`
	TmuxDefaultOpenAction string      `json:"tmux_default_open_action,omitempty"`
	Theme                 ThemeConfig `json:"theme,omitempty"`
}

func DefaultConfig() (*Config, error) {
	return defaultConfig()
}

func defaultConfig() (*Config, error) {
	homeDir, err := os.UserHomeDir()

	if err != nil {
		return nil, fmt.Errorf("failed to get user home directory: %w", err)
	}

	return &Config{
		Editor: "nvim",
		SearchPaths: []string{
			filepath.Join(homeDir, "dev"), filepath.Join(homeDir, "projects"), filepath.Join(homeDir, "repos"), filepath.Join(homeDir, "workspaces")},
		FileManager:           platform.DetectFileManager(),
		Terminal:              platform.DetectTerminal(),
		TmuxDefaultOpenAction: string(TmuxOpenEditor),
		Theme:                 ThemeConfig{Mode: string(ThemeModeAuto)},
	}, nil
}

func ConfigPath() (string, error) {

	configPath, err := os.UserConfigDir()
	if err != nil {
		return "", fmt.Errorf("failed to get user config directory: %w", err)
	}

	return filepath.Join(configPath, "gitf", "config.json"), nil
}

func Load() (*Config, error) {
	configPath, err := ConfigPath()
	if err != nil {
		return nil, fmt.Errorf("failed to get config path: %w", err)
	}

	return load(configPath)
}

func load(configPath string) (*Config, error) {
	data, err := os.ReadFile(configPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	var config Config
	err = json.Unmarshal(data, &config)
	if err != nil {
		return nil, fmt.Errorf("failed to unmarshal config: %w", err)
	}

	config.TmuxDefaultOpenAction = string(config.GetTmuxDefaultOpenAction())
	config.Theme = config.GetThemeConfig()

	return &config, nil
}

func (c *Config) Save() error {
	configPath, err := ConfigPath()
	if err != nil {
		return fmt.Errorf("failed to get config path: %w", err)
	}

	return save(configPath, c)
}

func save(configPath string, cfg *Config) error {
	cfg.Theme = cfg.GetThemeConfig()
	cfg.TmuxDefaultOpenAction = string(cfg.GetTmuxDefaultOpenAction())
	data, err := json.MarshalIndent(cfg, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal config: %w", err)
	}

	err = os.MkdirAll(filepath.Dir(configPath), 0755)
	if err != nil {
		return fmt.Errorf("failed to create config directory: %w", err)
	}

	err = os.WriteFile(configPath, data, 0644)
	if err != nil {
		return fmt.Errorf("failed to save config: %w", err)
	}

	return nil
}

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

// GetTmuxDefaultOpenAction returns the normalized tmux default open action.
func (c *Config) GetTmuxDefaultOpenAction() TmuxOpenAction {
	if c == nil {
		return TmuxOpenEditor
	}
	return NormalizeTmuxOpenAction(c.TmuxDefaultOpenAction)
}

// TmuxOpenActionOptions returns all valid tmux open actions.
func TmuxOpenActionOptions() []TmuxOpenAction {
	return []TmuxOpenAction{
		TmuxOpenEditor,
		TmuxOpenWindow,
		TmuxOpenVerticalPane,
		TmuxOpenHorizontalPane,
		TmuxOpenSession,
	}
}

// NormalizeTmuxOpenAction normalizes an action string to a valid TmuxOpenAction.
func NormalizeTmuxOpenAction(action string) TmuxOpenAction {
	switch TmuxOpenAction(action) {
	case TmuxOpenEditor, TmuxOpenWindow, TmuxOpenVerticalPane, TmuxOpenHorizontalPane, TmuxOpenSession:
		return TmuxOpenAction(action)
	default:
		return TmuxOpenEditor
	}
}

// GetFileManager returns configured file manager or auto-detects if empty
func (c *Config) GetFileManager() string {
	if c.FileManager != "" {
		return c.FileManager
	}
	return platform.DetectFileManager()
}

// GetTerminal returns configured terminal, detects current terminal, or falls back to system default
func (c *Config) GetTerminal() string {
	if c.Terminal != "" {
		return c.Terminal
	}
	// Try to detect which terminal we're running in
	if current := platform.DetectCurrentTerminal(); current != "" {
		return current
	}
	// Fall back to system default terminal
	return platform.DetectTerminal()
}
