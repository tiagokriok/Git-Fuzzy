# GF - Git Repository Finder

A high-performance CLI tool for discovering and opening Git repositories with interactive fuzzy search. Built with Go, leveraging modern terminal UI patterns and efficient filesystem traversal.

## Key Features

- **Real-time Fuzzy Search**: Instant repository filtering with substring matching
- **Interactive TUI**: Responsive terminal UI using The Elm Architecture (Bubbletea)
- **Intelligent Scanning**: Recursive filesystem traversal with automatic exclusion of non-essential directories
- **Editor Integration**: Seamless handoff to configured editor (Neovim, VS Code, Vim, etc.)
- **Zero-Configuration Setup**: Interactive wizard creates sensible defaults on first run
- **Performance Optimized**: Efficient directory traversal with early termination and deduplication

## Quick Start

### Installation

```bash
git clone https://github.com/tiagokriok/gf.git
cd gf
go build -o gf ./cmd/gf
sudo mv gf /usr/local/bin/  # Optional: add to PATH
```

### Basic Usage

```bash
gf
```

On first run, an interactive wizard creates `~/.config/gf/config.json`:

```json
{
  "editor": "nvim",
  "search_paths": [
    "~/dev",
    "~/projects",
    "~/repos",
    "~/workspaces"
  ]
}
```

Customize by editing the configuration file directly or re-running the setup wizard.

## Keyboard Shortcuts

- `↑` / `↓` or `Tab` / `Shift+Tab`: Navigate repositories
- `Type`: Filter by repository name (fuzzy search)
- `Enter`: Open selected repository in editor
- `Backspace`: Delete character from search
- `Esc` / `Ctrl+C`: Exit without selection

## Technical Architecture

### Project Structure

```
gf/
├── cmd/gf/
│   └── main.go              # Application entry point & editor integration
├── internal/
│   ├── config/
│   │   ├── config.go        # Configuration management (load/save/defaults)
│   │   └── config_test.go   # Unit tests (62.8% coverage)
│   ├── scanner/
│   │   ├── scanner.go       # Repository discovery with optimized traversal
│   │   └── scanner_test.go  # Unit tests (85.7% coverage)
│   └── ui/
│       ├── ui.go            # Interactive TUI & result rendering
│       └── setup.go         # Setup wizard for first-run configuration
├── go.mod
└── go.sum
```

### Design Patterns

- **Separation of Concerns**: Modular architecture with config, scanner, and UI packages
- **The Elm Architecture**: TUI implementation using Bubbletea for predictable state management
- **Efficient Traversal**: `filepath.Walk()` with early `SkipDir` for large directory trees
- **Deduplication**: Map-based tracking prevents duplicate repository entries
- **Error Handling**: Comprehensive error wrapping with context using `fmt.Errorf("%w")`

## Development

### Build & Test

```bash
# Build
go build -o gf ./cmd/gf

# Run tests with verbose output
go test -v ./...

# Check test coverage
go test -cover ./...

# Run specific test
go test -v -run TestNameHere ./internal/config
```

### Code Quality

```bash
# Format code (Go standard)
gofmt -l -w .
go fmt ./...

# Static analysis
go vet ./...

# Update dependencies
go mod tidy
go mod download
```

### Test Coverage Summary

- **Config Package**: 62.8% coverage (6 tests)
- **Scanner Package**: 85.7% coverage (5 tests)
- **UI Package**: Manual testing (TUI interaction testing)
- **Total**: 11 passing tests

## Technology Stack

### Core Dependencies

| Package | Purpose | Version |
|---------|---------|---------|
| [charmbracelet/bubbletea](https://github.com/charmbracelet/bubbletea) | TUI framework implementing Elm Architecture | Latest |
| [charmbracelet/lipgloss](https://github.com/charmbracelet/lipgloss) | Terminal styling and layout | Latest |
| [sahilm/fuzzy](https://github.com/sahilm/fuzzy) | Efficient fuzzy string matching | Latest |

### Go Version
- **Go 1.25.5** or higher required

## Configuration

Configuration is stored in `~/.config/gf/config.json` and created automatically on first run.

### Configuration Options

| Option | Type | Description | Example |
|--------|------|-------------|---------|
| `editor` | string | Command to launch when opening repository | `"nvim"`, `"code"`, `"vim"` |
| `search_paths` | array | Directories to recursively scan | `["/home/user/dev", "/work"]` |

### Configuration Example

```json
{
  "editor": "nvim",
  "search_paths": [
    "/home/user/dev",
    "/home/user/projects",
    "/home/user/work",
    "/opt/services"
  ]
}
```

### Performance Optimization

The scanner automatically skips these directories to improve traversal speed:

```
node_modules, vendor, target        # Dependency/build directories
.git, .config, .vscode, .idea       # Hidden configuration directories
.cache, venv, .venv, venv3, .venv3  # Cache and virtual environments
```

This intelligent filtering enables sub-second repository discovery even in large directory trees.

## License

MIT
