# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

## Project Overview

**GF (Git Fuzzy)** is a high-performance CLI tool for discovering and opening Git repositories with interactive fuzzy search. Built with Go using Bubbletea (The Elm Architecture) for the TUI, it provides responsive terminal UI with efficient filesystem traversal.

## Common Development Commands

### Build and Run
```bash
make build              # Build the gitf binary
make run                # Build and run gitf
make build-optimized    # Build optimized binary (29% smaller)
make install            # Install to $GOPATH/bin
```

### Testing
```bash
make test               # Run all tests
make test-verbose       # Run tests with verbose output
make test-coverage      # Generate HTML coverage report (opens in browser)
go test -v -run TestName ./internal/package  # Run specific test
```

### Code Quality
```bash
make fmt                # Format code with gofmt
make lint               # Run go vet
make check              # Run fmt, lint, and test
make dev                # Full workflow: clean, fmt, lint, test, build
```

### Cleanup
```bash
make clean              # Remove build artifacts and coverage files
make reset-local        # Remove binary and config for fresh start
```

### Dependencies
```bash
make deps               # Download and tidy dependencies
go mod tidy             # Update module files
```

## Architecture

### High-Level Flow

The application follows a clear initialization and execution flow in `cmd/gitf/main.go`:

1. **Config Loading** → Load from `~/.config/gitf/config.json`; if missing, run setup wizard
2. **Repository Scanning** → Use `scanner.Scan()` to find all Git repos in configured search paths
3. **Recent Reordering** → Load recent history and reorder results with `scanner.ReorderByRecent()`
4. **Interactive UI** → User selects repo via `ui.Run()` using fuzzy search
5. **Editor Launch** → Open selected repo in configured editor
6. **History Update** → Record selected repo path in recent history

### Package Structure

**`internal/config`** - Configuration management
- Handles JSON serialization of `~/.config/gitf/config.json`
- Fields: `Editor` (editor command), `SearchPaths` (directories to scan)
- Provides `Load()` for reading config and `Save()` for writing changes
- `DefaultConfig()` returns sensible defaults (nvim editor, common dev directories)

**`internal/scanner`** - Repository discovery
- `Scan(searchPaths []string)` recursively finds Git repositories using `filepath.Walk()`
- Returns `[]Repository` with `Name` and `Path` fields
- Optimization: skips ignored directories (node_modules, vendor, .git, venv, etc.) via `filepath.SkipDir`
- Deduplicates results using map-based tracking
- `ReorderByRecent(repos, recents)` sorts repos by most recent usage from history

**`internal/history`** - Recent repositories tracking
- Maintains `~/.config/gitf/recent.json` with last 10 opened repositories
- `LoadRecent()` loads recent history; returns empty struct if file doesn't exist
- `Add(repoPath)` prepends to list, maintains MaxRecent=10 limit, deduplicates
- `Save()` persists to JSON file

**`internal/ui`** - Interactive TUI (Bubbletea-based)
- `ui.Model` implements Elm Architecture (Init, Update, View)
- `Run(repos)` starts TUI loop, returns selected repository or nil
- `RunSetup()` runs first-run configuration wizard
- Handles keyboard input: arrow keys/Tab for navigation, typing for fuzzy search, Enter to select, Esc to exit
- Uses lipgloss for terminal styling (rounded borders, colors, padding)
- Uses sahilm/fuzzy for substring matching during search

### Key Design Decisions

**Elm Architecture for UI**: Bubbletea's purely functional Model → Update → View pattern ensures predictable state management and composability. Messages flow through a single Update function.

**Early Directory Skipping**: The scanner uses `filepath.SkipDir` to avoid traversing common dependency/build directories, enabling sub-second discovery even in large codebases.

**Deduplication**: Map-based tracking prevents duplicate repositories when paths are scanned from multiple source directories.

**Graceful Degradation**: History loading is non-fatal (empty history if file doesn't exist). UI and config wizards handle missing state gracefully.

## Code Coverage

Current test coverage focuses on deterministic packages:
- **Config**: 62.9% coverage (6 tests for Load/Save, defaults, path handling)
- **Scanner**: 60.0% coverage (5 tests for Git detection, ignoring, deduplication)
- **History**: Implemented but pending unit tests
- **UI**: Manual testing (TUI interaction is hard to test programmatically)
- **Total**: 11 passing tests

Run `make test-coverage` to generate HTML report.

## Testing Strategy

Tests use table-driven patterns and test temporary directories. When adding tests:
- Create fixtures in temp directories using `os.Mkdir` / `os.WriteFile`
- Use `t.TempDir()` for automatic cleanup
- Test error cases (missing files, invalid JSON, permission issues)
- For UI changes, manual testing in terminal is primary validation method

## Build Configuration

The Makefile uses build-time flags for optimization:
- `-ldflags "-s -w"` strips debug symbols and DWARF info, reducing binary size by ~29%
- Standard build includes symbols for debugging; use `build-optimized` for production

## Dependencies

**Core Libraries**:
- `charmbracelet/bubbletea` - TUI framework (Elm Architecture)
- `charmbracelet/lipgloss` - Terminal styling (borders, colors, layout)
- `sahilm/fuzzy` - Efficient substring matching for search

**Go Version**: 1.25.5 minimum (leverages recent improvements in file handling)

## Important Behavioral Notes

1. **Config Wizard**: Automatically runs on first launch if config doesn't exist. User must provide editor and search paths.

2. **Directory Skipping**: Changes to `ignoredDirs` in `scanner.go` should be tested manually in directories with many files (node_modules, vendor, etc.) to verify performance impact.

3. **Recent History**: Persisted to `~/.config/gitf/recent.json` (separate from config.json). Corrupted history file is handled gracefully by returning empty history.

4. **Editor Integration**: Uses `os.Exec()` to launch the configured editor. On Windows, ensure editor path is correct (may need `.exe` extension).

5. **Search Behavior**: Fuzzy search is case-insensitive substring matching (via sahilm/fuzzy). Pressing characters accumulates in search box; Backspace removes characters.

## Running Single Tests

```bash
# Test a specific function in config package
go test -v -run TestLoad ./internal/config

# Test scanner with verbose output
go test -v ./internal/scanner

# Run all config tests
go test -v ./internal/config
```

## Common Troubleshooting

- **No repositories found**: Check that `search_paths` in config point to directories containing `.git` folders; scanner skips many common dependency directories.
- **Editor doesn't launch**: Verify editor command is in PATH and matches OS (e.g., `nvim` vs `nvim.exe` on Windows).
- **Config not updating**: Config changes require explicit `config.Save()` call; wizard output goes to `~/.config/gitf/config.json`.
