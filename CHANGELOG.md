# Changelog

All notable changes to this project will be documented in this file.

The format is based on [Keep a Changelog](https://keepachangelog.com/en/1.0.0/),
and this project adheres to [Semantic Versioning](https://semver.org/spec/v2.0.0.html).

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

## [Unreleased]

### Planned Features

- CLI flags: `--editor`, `--path`, `--help`, `--version`
- Unit tests for UI package (setup.go, ui.go)
- Unit tests for history package (recent.go)
- Installation script or package manager support (Brew, AUR, etc.)
- Caching for faster startup
- Extended configuration options
