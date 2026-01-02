# GF - Git Fuzzy

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

#### Using `go install` (Fastest) ⭐

```bash
go install github.com/tiagokriok/Git-Fuzzy/cmd/gitf@latest
gitf
```

This installs the latest version directly to `$GOPATH/bin`. Works on Linux, macOS, and Windows.

**Requirements**: Go 1.25.5 or higher

#### Using Makefile (Recommended for Development)

```bash
git clone https://github.com/tiagokriok/Git-Fuzzy.git
cd Git-Fuzzy
make build          # Build the binary
make install        # Install to $GOPATH/bin
```

#### Manual Build from Source

```bash
git clone https://github.com/tiagokriok/Git-Fuzzy.git
cd Git-Fuzzy
go build -o gitf ./cmd/gitf
sudo mv gitf /usr/local/bin/  # Optional: add to PATH
```

### Basic Usage

```bash
gitf
```

On first run, an interactive wizard creates `~/.config/gitf/config.json`:

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

### Makefile Commands

```bash
make help              # Show all available commands
make build             # Build the gitf binary
make build-optimized   # Build optimized binary (29% smaller)
make run               # Run gitf directly
make test              # Run all tests
make test-verbose      # Run tests with verbose output
make test-coverage     # Generate coverage report (opens HTML)
make fmt               # Format code
make lint              # Run go vet
make clean             # Remove build artifacts
make install           # Install to $GOPATH/bin
make deps              # Download and manage dependencies
make reset-local       # Reset config and binary for fresh start
make check             # Run fmt, lint, and test
make dev               # Full development workflow (clean, fmt, lint, test, build)
```

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
├── cmd/gitf/
│   └── main.go                      # Application entry point & editor integration
├── internal/
│   ├── config/
│   │   ├── config.go                # Configuration management (load/save/defaults)
│   │   └── config_test.go           # Unit tests (62.9% coverage)
│   ├── history/
│   │   └── recent.go                # Recent repositories tracking & persistence
│   ├── scanner/
│   │   ├── scanner.go               # Repository discovery with optimized traversal
│   │   │                             # Includes ReorderByRecent() for intelligent sorting
│   │   └── scanner_test.go          # Unit tests (60.0% coverage)
│   └── ui/
│       ├── ui.go                    # Interactive TUI & result rendering
│       └── setup.go                 # Setup wizard for first-run configuration
├── Makefile                         # Development workflow automation
├── .gitignore                       # Git ignore patterns
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
go build -o gitf ./cmd/gitf

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

- **Config Package**: 62.9% coverage (6 tests)
- **Scanner Package**: 60.0% coverage (5 tests)
- **History Package**: Implemented (pending unit tests)
- **UI Package**: Manual testing (TUI interaction testing)
- **Total**: 11 passing tests ✅

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

Configuration is stored in `~/.config/gitf/config.json` and created automatically on first run.

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

## Recent Updates

### Version 0.1.0

#### ✨ New Features
- **Makefile**: Complete development workflow automation with 15+ targets
- **Binary Optimization**: Build-time flag optimization reduces binary size by 29% (5.1M → 3.6M)
- **Recent Repositories**: Track and prioritize last 10 opened repositories
- **Intelligent Sorting**: Automatically reorder search results by most recent usage

#### 🔧 Improvements
- Test Coverage: Now at 11 passing tests across config and scanner packages
- Code Quality: Full `go fmt`, `go vet` compliance with error wrapping

#### 📊 Project Stats
- **Lines of Code**: ~990 (production: 700, tests: 290)
- **Binary Size**: 5.1M (standard), 3.6M (optimized)
- **Test Pass Rate**: 100% (11/11 tests passing)
- **Supported Go Version**: 1.25.5+

## License

MIT
