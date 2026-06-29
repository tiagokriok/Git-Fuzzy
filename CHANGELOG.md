# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

## [0.5.0] - 2026-06-29

### Added

- Omarchy theme integration with automatic theme detection.
- Theme config modes: `auto`, `default`, `omarchy`, and `file`.
- `gitf theme sync-omarchy` command to generate `~/.config/gitf/theme.json`.
- Semantic UI styling that follows Omarchy colors in the main TUI and setup wizard.
- Live theme refresh while the TUI is open.
- Manual theme file support with partial palette fallback.

## [0.4.0] - 2026-06-29

### Added

- Tmux open actions for selected repositories:
  - `Alt+w` opens a new tmux window.
  - `Alt+v` opens a vertical split, side by side.
  - `Alt+h` opens a horizontal split, top/bottom.
  - `Alt+s` prompts for a new tmux session name.
- Configurable `tmux_default_open_action` for `Enter` behavior inside tmux.
- Setup wizard step for choosing the default tmux Enter action.
- Documentation for tmux popup usage and tmux shortcuts.

## [0.1.0] - 2026-01-01

### ✨ Added

- **Interactive TUI**: Full terminal user interface using Bubbletea (Elm Architecture)
- **Fuzzy Search**: Real-time repository filtering with sahilm/fuzzy
- **Setup Wizard**: Interactive first-run configuration with editor and search paths
- **Recent Repositories**: Track and prioritize last 10 opened repositories
- **Smart Sorting**: Automatically reorder results by most recent usage
- **Configuration Management**: JSON-based config at `~/.config/gf/config.json`
- **Keyboard Navigation**: ↑/↓ arrows, Tab/Shift+Tab, Enter/Esc controls
- **Makefile**: Complete development workflow with 15+ automation targets
- **Optimized Builds**: Binary optimization with -s -w flags (29% smaller)
- **Comprehensive Tests**: 11 unit tests with coverage for config and scanner packages

### 🔧 Changed

- Improved error handling with context-aware error wrapping
- Enhanced repository display with cleaned path formatting
- Optimized directory scanning with intelligent directory filtering

### 🐛 Fixed

- Fixed `.gitignore` to allow `cmd/gf/` directory to be tracked
- Resolved duplicate repository entries in results
- Corrected file not found error handling in config loading

### 📊 Features Summary

**Core Functionality:**
- Discover Git repositories in specified directories
- Interactive selection with real-time fuzzy search
- Automatic opening in configured editor
- Recent repository tracking and prioritization

**User Experience:**
- Non-intrusive setup wizard
- Responsive terminal interface
- Clear keyboard shortcuts
- Pagination for large result sets
- Home directory path normalization

**Development:**
- 100% test pass rate (11/11 tests)
- 62.9% coverage (config), 60.0% (scanner)
- Full `go fmt` and `go vet` compliance
- Modular architecture for easy testing

### 📦 Binaries

- **Standard**: 5.1M
- **Optimized**: 3.6M (29% reduction)

### 🔗 Dependencies

- Go 1.25.5+
- charmbracelet/bubbletea (TUI framework)
- charmbracelet/bubbles (UI components)
- charmbracelet/lipgloss (terminal styling)
- sahilm/fuzzy (fuzzy matching)

---

## [0.1.4] - 2026-01-02

### ⚠️ BREAKING CHANGES

- **Configuration folder renamed**: `~/.config/gf/` → `~/.config/gitf/`
  - If upgrading from v0.1.2 or earlier, manually move your config:
    ```bash
    mkdir -p ~/.config/gitf
    mv ~/.config/gf/config.json ~/.config/gitf/config.json
    rm -rf ~/.config/gf
    ```

### 🔧 Changed

- Binary directory renamed from `cmd/gf/` to `cmd/gitf/`
- Config path changed from `~/.config/gf/config.json` to `~/.config/gitf/config.json`
- Updated all build scripts and documentation

### 📦 Installation

- **go install**: `go install github.com/tiagokriok/Git-Fuzzy/cmd/gitf@v0.1.4`
- Pre-built binaries available for all platforms

---

## [0.1.3] - 2026-01-02

### 🔧 Changed

- Fixed `go install` command to produce `gitf` binary instead of `gf`
- Renamed `cmd/gf` directory to `cmd/gitf` for proper naming convention
- Updated build scripts and documentation

### 📦 Note

- This version still uses `~/.config/gf/config.json` path

---

## [Unreleased]

### Planned Features

- CLI flags: `--editor`, `--path`, `--help`, `--version`
- Unit tests for UI package (setup.go, ui.go)
- Unit tests for history package (recent.go)
- Installation script or package manager support (Brew, AUR, etc.)
- Caching for faster startup
- Extended configuration options
