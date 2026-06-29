# Tmux Open Actions Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add tmux-aware open actions so selected repositories can open in tmux windows, panes, or named sessions while preserving existing editor behavior outside tmux.

**Architecture:** Keep command execution in a new `internal/tmux` package, keep interactive prompt and key handling in `internal/ui`, and keep `cmd/gitf/main.go` mostly unchanged. Config owns action constants and normalization so old configs safely default to editor behavior.

**Tech Stack:** Go 1.25.5, Go modules, Bubbletea, Lipgloss, Cobra, standard `os/exec`, test runner `go test ./...`, checks through `make check`.

## Global Constraints

- Do not commit unless the user explicitly asks.
- Preserve existing `Enter` behavior outside tmux.
- Existing configs without `tmux_default_open_action` must behave as `editor`.
- Tmux availability requires `TMUX` environment variable and `tmux` binary in `PATH`.
- Tmux actions open the tmux default shell in the selected repository directory via `-c <repo>`.
- `Alt+w`, `Alt+v`, `Alt+h`, and `Alt+s` are the direct tmux actions.
- `Alt+v` means side-by-side pane and must use `tmux split-window -h`.
- `Alt+h` means top/bottom pane and must use `tmux split-window -v`.
- Session creation always prompts for a user-provided name.
- Empty or duplicate session names must keep the prompt open with an error.
- `Esc` cancels the session prompt; `Ctrl+C` exits `gitf`.
- Avoid shell string concatenation for tmux commands. Use `exec.Command` style args.

---

## File Structure

- Modify `internal/config/config.go`
  - Add `TmuxDefaultOpenAction` to `Config`.
  - Add action constants and normalization helpers.
  - Ensure default and loaded config normalize to `editor` when missing or invalid.

- Modify `internal/config/config_test.go`
  - Add tests for default action, missing field, invalid field, and valid field preservation.

- Create `internal/tmux/tmux.go`
  - Detect tmux availability.
  - Execute window, pane, and session commands.
  - Check session existence.
  - Provide package-level injection points for tests.

- Create `internal/tmux/tmux_test.go`
  - Test availability and command args without requiring a real tmux session.

- Modify `internal/ui/ui.go`
  - Add UI mode for session-name prompt.
  - Add status error field.
  - Add key handling for `Alt+w`, `Alt+v`, `Alt+h`, `Alt+s`.
  - Route configured `Enter` action when inside tmux.
  - Quit without returning a selected repository after successful tmux actions.

- Create `internal/ui/tmux_actions_test.go`
  - Test small helper behavior for session input validation and action routing.

- Modify `internal/ui/setup.go`
  - Add setup step 3 for `tmux_default_open_action` selection.

- Modify `README.md`
  - Document tmux shortcuts, config field, and popup binding example.

---

### Task 1: Config Action Model

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

**Interfaces:**
- Produces: `type TmuxOpenAction string`
- Produces constants: `TmuxOpenEditor`, `TmuxOpenWindow`, `TmuxOpenVerticalPane`, `TmuxOpenHorizontalPane`, `TmuxOpenSession`
- Produces: `func NormalizeTmuxOpenAction(action string) TmuxOpenAction`
- Produces: `func (c *Config) GetTmuxDefaultOpenAction() TmuxOpenAction`
- Produces: `func TmuxOpenActionOptions() []TmuxOpenAction`
- Consumed by later tasks: `internal/ui` setup and action routing.

- [ ] **Step 1: Write failing config tests**

Append these tests to `internal/config/config_test.go`:

```go
func TestDefaultConfig_TmuxDefaultOpenAction(t *testing.T) {
	cfg, err := DefaultConfig()
	assertNoError(t, err)

	assertEqual(t, TmuxOpenEditor, cfg.GetTmuxDefaultOpenAction(), "tmux default open action")
}

func TestLoad_MissingTmuxDefaultOpenActionDefaultsToEditor(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	jsonData := []byte(`{
		"editor": "nvim",
		"search_paths": ["/home/user/dev"]
	}`)
	assertNoError(t, os.WriteFile(configFile, jsonData, 0644))

	cfg, err := load(configFile)
	assertNoError(t, err)

	assertEqual(t, TmuxOpenEditor, cfg.GetTmuxDefaultOpenAction(), "missing tmux default open action")
}

func TestLoad_InvalidTmuxDefaultOpenActionDefaultsToEditor(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	jsonData := []byte(`{
		"editor": "nvim",
		"search_paths": ["/home/user/dev"],
		"tmux_default_open_action": "bad-action"
	}`)
	assertNoError(t, os.WriteFile(configFile, jsonData, 0644))

	cfg, err := load(configFile)
	assertNoError(t, err)

	assertEqual(t, TmuxOpenEditor, cfg.GetTmuxDefaultOpenAction(), "invalid tmux default open action")
}

func TestLoad_ValidTmuxDefaultOpenActionPreserved(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "config.json")

	jsonData := []byte(`{
		"editor": "nvim",
		"search_paths": ["/home/user/dev"],
		"tmux_default_open_action": "tmux-window"
	}`)
	assertNoError(t, os.WriteFile(configFile, jsonData, 0644))

	cfg, err := load(configFile)
	assertNoError(t, err)

	assertEqual(t, TmuxOpenWindow, cfg.GetTmuxDefaultOpenAction(), "valid tmux default open action")
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/config
```

Expected: FAIL because `TmuxOpenEditor`, `TmuxOpenWindow`, and `GetTmuxDefaultOpenAction` are undefined.

- [ ] **Step 3: Implement config action model**

Edit `internal/config/config.go` so the top-level types include:

```go
type TmuxOpenAction string

const (
	TmuxOpenEditor         TmuxOpenAction = "editor"
	TmuxOpenWindow         TmuxOpenAction = "tmux-window"
	TmuxOpenVerticalPane   TmuxOpenAction = "tmux-vertical-pane"
	TmuxOpenHorizontalPane TmuxOpenAction = "tmux-horizontal-pane"
	TmuxOpenSession        TmuxOpenAction = "tmux-session"
)

type Config struct {
	Editor                string   `json:"editor"`
	SearchPaths           []string `json:"search_paths"`
	FileManager           string   `json:"file_manager,omitempty"`
	Terminal              string   `json:"terminal,omitempty"`
	TmuxDefaultOpenAction string   `json:"tmux_default_open_action,omitempty"`
}

func TmuxOpenActionOptions() []TmuxOpenAction {
	return []TmuxOpenAction{
		TmuxOpenEditor,
		TmuxOpenWindow,
		TmuxOpenVerticalPane,
		TmuxOpenHorizontalPane,
		TmuxOpenSession,
	}
}

func NormalizeTmuxOpenAction(action string) TmuxOpenAction {
	switch TmuxOpenAction(action) {
	case TmuxOpenEditor, TmuxOpenWindow, TmuxOpenVerticalPane, TmuxOpenHorizontalPane, TmuxOpenSession:
		return TmuxOpenAction(action)
	default:
		return TmuxOpenEditor
	}
}

func (c *Config) GetTmuxDefaultOpenAction() TmuxOpenAction {
	if c == nil {
		return TmuxOpenEditor
	}
	return NormalizeTmuxOpenAction(c.TmuxDefaultOpenAction)
}
```

Update `defaultConfig()` return value to include:

```go
		TmuxDefaultOpenAction: string(TmuxOpenEditor),
```

Update `load(configPath string)` after JSON unmarshal:

```go
	config.TmuxDefaultOpenAction = string(config.GetTmuxDefaultOpenAction())
```

- [ ] **Step 4: Run config tests and verify pass**

Run:

```bash
go test ./internal/config
```

Expected: PASS.

- [ ] **Step 5: Run formatting for touched package**

Run:

```bash
go fmt ./internal/config
```

Expected: no errors.

- [ ] **Step 6: Review checkpoint**

Run:

```bash
git diff -- internal/config/config.go internal/config/config_test.go
```

Expected: diff only adds tmux action config support and tests. Do not commit unless the user explicitly asks.

---

### Task 2: Tmux Command Package

**Files:**
- Create: `internal/tmux/tmux.go`
- Create: `internal/tmux/tmux_test.go`

**Interfaces:**
- Consumes: standard `os`, `os/exec`, `errors`, `fmt`, `strings`.
- Produces: `func IsAvailable() bool`
- Produces: `func OpenWindow(repoPath string) error`
- Produces: `func OpenVerticalPane(repoPath string) error`
- Produces: `func OpenHorizontalPane(repoPath string) error`
- Produces: `func SessionExists(name string) (bool, error)`
- Produces: `func OpenSession(name, repoPath string) error`
- Produces errors containing `tmux is not available`, `session name cannot be empty`, and `tmux session already exists`.

- [ ] **Step 1: Write failing tmux package tests**

Create `internal/tmux/tmux_test.go`:

```go
package tmux

import (
	"errors"
	"reflect"
	"testing"
)

type commandCall struct {
	name string
	args []string
}

func resetTestHooks(t *testing.T) *[]commandCall {
	t.Helper()

	calls := []commandCall{}
	originalLookupPath := lookupPath
	originalGetenv := getenv
	originalRunCommand := runCommand

	lookupPath = func(file string) (string, error) {
		if file == "tmux" {
			return "/usr/bin/tmux", nil
		}
		return "", errors.New("not found")
	}
	getenv = func(key string) string {
		if key == "TMUX" {
			return "/tmp/tmux-1000/default,123,0"
		}
		return ""
	}
	runCommand = func(name string, args ...string) error {
		copiedArgs := append([]string(nil), args...)
		calls = append(calls, commandCall{name: name, args: copiedArgs})
		return nil
	}

	t.Cleanup(func() {
		lookupPath = originalLookupPath
		getenv = originalGetenv
		runCommand = originalRunCommand
	})

	return &calls
}

func TestIsAvailableRequiresTmuxEnvAndBinary(t *testing.T) {
	resetTestHooks(t)

	if !IsAvailable() {
		t.Fatal("expected tmux to be available")
	}

	getenv = func(key string) string { return "" }
	if IsAvailable() {
		t.Fatal("expected tmux to be unavailable without TMUX env")
	}

	getenv = func(key string) string { return "present" }
	lookupPath = func(file string) (string, error) { return "", errors.New("not found") }
	if IsAvailable() {
		t.Fatal("expected tmux to be unavailable without tmux binary")
	}
}

func TestOpenWindowRunsNewWindowInRepoPath(t *testing.T) {
	calls := resetTestHooks(t)

	err := OpenWindow("/home/user/dev/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []commandCall{{name: "tmux", args: []string{"new-window", "-c", "/home/user/dev/repo"}}}
	if !reflect.DeepEqual(expected, *calls) {
		t.Fatalf("expected calls %#v, got %#v", expected, *calls)
	}
}

func TestOpenVerticalPaneUsesHorizontalTmuxSplitFlag(t *testing.T) {
	calls := resetTestHooks(t)

	err := OpenVerticalPane("/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []commandCall{{name: "tmux", args: []string{"split-window", "-h", "-c", "/repo"}}}
	if !reflect.DeepEqual(expected, *calls) {
		t.Fatalf("expected calls %#v, got %#v", expected, *calls)
	}
}

func TestOpenHorizontalPaneUsesVerticalTmuxSplitFlag(t *testing.T) {
	calls := resetTestHooks(t)

	err := OpenHorizontalPane("/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []commandCall{{name: "tmux", args: []string{"split-window", "-v", "-c", "/repo"}}}
	if !reflect.DeepEqual(expected, *calls) {
		t.Fatalf("expected calls %#v, got %#v", expected, *calls)
	}
}

func TestOpenSessionCreatesDetachedSessionAndSwitchesClient(t *testing.T) {
	calls := resetTestHooks(t)

	err := OpenSession("work", "/repo")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}

	expected := []commandCall{
		{name: "tmux", args: []string{"has-session", "-t", "work"}},
		{name: "tmux", args: []string{"new-session", "-d", "-s", "work", "-c", "/repo"}},
		{name: "tmux", args: []string{"switch-client", "-t", "work"}},
	}
	if !reflect.DeepEqual(expected, *calls) {
		t.Fatalf("expected calls %#v, got %#v", expected, *calls)
	}
}

func TestOpenSessionRejectsEmptyName(t *testing.T) {
	resetTestHooks(t)

	err := OpenSession("   ", "/repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "session name cannot be empty" {
		t.Fatalf("expected empty name error, got %v", err)
	}
}

func TestOpenSessionRejectsExistingSession(t *testing.T) {
	resetTestHooks(t)
	runCommand = func(name string, args ...string) error { return nil }

	err := OpenSession("work", "/repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "tmux session already exists: work" {
		t.Fatalf("expected duplicate session error, got %v", err)
	}
}

func TestTmuxActionReturnsUnavailableWhenOutsideTmux(t *testing.T) {
	resetTestHooks(t)
	getenv = func(key string) string { return "" }

	err := OpenWindow("/repo")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "tmux is not available" {
		t.Fatalf("expected unavailable error, got %v", err)
	}
}
```

- [ ] **Step 2: Run tests and verify failure**

Run:

```bash
go test ./internal/tmux
```

Expected: FAIL because package files or symbols are missing.

- [ ] **Step 3: Implement `internal/tmux/tmux.go`**

Create `internal/tmux/tmux.go`:

```go
package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

var (
	ErrUnavailable      = errors.New("tmux is not available")
	ErrEmptySessionName = errors.New("session name cannot be empty")

	lookupPath = exec.LookPath
	getenv     = os.Getenv
	runCommand = func(name string, args ...string) error {
		return exec.Command(name, args...).Run()
	}
)

func IsAvailable() bool {
	if getenv("TMUX") == "" {
		return false
	}
	_, err := lookupPath("tmux")
	return err == nil
}

func OpenWindow(repoPath string) error {
	if !IsAvailable() {
		return ErrUnavailable
	}
	return runCommand("tmux", "new-window", "-c", repoPath)
}

func OpenVerticalPane(repoPath string) error {
	if !IsAvailable() {
		return ErrUnavailable
	}
	return runCommand("tmux", "split-window", "-h", "-c", repoPath)
}

func OpenHorizontalPane(repoPath string) error {
	if !IsAvailable() {
		return ErrUnavailable
	}
	return runCommand("tmux", "split-window", "-v", "-c", repoPath)
}

func SessionExists(name string) (bool, error) {
	if !IsAvailable() {
		return false, ErrUnavailable
	}

	if strings.TrimSpace(name) == "" {
		return false, ErrEmptySessionName
	}

	err := runCommand("tmux", "has-session", "-t", name)
	if err == nil {
		return true, nil
	}
	return false, nil
}

func OpenSession(name, repoPath string) error {
	name = strings.TrimSpace(name)
	if name == "" {
		return ErrEmptySessionName
	}
	if !IsAvailable() {
		return ErrUnavailable
	}

	exists, err := SessionExists(name)
	if err != nil {
		return err
	}
	if exists {
		return fmt.Errorf("tmux session already exists: %s", name)
	}

	if err := runCommand("tmux", "new-session", "-d", "-s", name, "-c", repoPath); err != nil {
		return err
	}

	return runCommand("tmux", "switch-client", "-t", name)
}
```

- [ ] **Step 4: Run tmux tests and verify pass**

Run:

```bash
go test ./internal/tmux
```

Expected: PASS.

- [ ] **Step 5: Run formatting for tmux package**

Run:

```bash
go fmt ./internal/tmux
```

Expected: no errors.

- [ ] **Step 6: Review checkpoint**

Run:

```bash
git diff -- internal/tmux/tmux.go internal/tmux/tmux_test.go
```

Expected: new package only contains detection, command wrappers, and tests. Do not commit unless the user explicitly asks.

---

### Task 3: UI Tmux Actions and Session Prompt

**Files:**
- Modify: `internal/ui/ui.go`
- Create: `internal/ui/tmux_actions_test.go`

**Interfaces:**
- Consumes: `config.TmuxOpenAction`, `config.TmuxOpenEditor`, `config.TmuxOpenWindow`, `config.TmuxOpenVerticalPane`, `config.TmuxOpenHorizontalPane`, `config.TmuxOpenSession`
- Consumes: `tmux.IsAvailable`, `tmux.OpenWindow`, `tmux.OpenVerticalPane`, `tmux.OpenHorizontalPane`, `tmux.OpenSession`
- Produces helper: `func validateSessionNameInput(name string) error`
- Produces UI mode constants for repository list and session prompt.

- [ ] **Step 1: Write focused UI tests**

Create `internal/ui/tmux_actions_test.go`:

```go
package ui

import "testing"

func TestValidateSessionNameInputRejectsEmptyName(t *testing.T) {
	err := validateSessionNameInput("   ")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if err.Error() != "session name required" {
		t.Fatalf("expected required error, got %v", err)
	}
}

func TestValidateSessionNameInputAcceptsTrimmedName(t *testing.T) {
	err := validateSessionNameInput(" work ")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
}

func TestModelStartSessionPrompt(t *testing.T) {
	m := Model{}

	m.startSessionPrompt()

	if m.mode != modeSessionPrompt {
		t.Fatalf("expected session prompt mode, got %v", m.mode)
	}
	if m.sessionNameInput != "" {
		t.Fatalf("expected empty session input, got %q", m.sessionNameInput)
	}
	if m.sessionPromptError != "" {
		t.Fatalf("expected empty prompt error, got %q", m.sessionPromptError)
	}
}

func TestModelCancelSessionPrompt(t *testing.T) {
	m := Model{
		mode:               modeSessionPrompt,
		sessionNameInput:   "work",
		sessionPromptError: "session name required",
	}

	m.cancelSessionPrompt()

	if m.mode != modeRepositoryList {
		t.Fatalf("expected repository list mode, got %v", m.mode)
	}
	if m.sessionNameInput != "" {
		t.Fatalf("expected session input to clear, got %q", m.sessionNameInput)
	}
	if m.sessionPromptError != "" {
		t.Fatalf("expected prompt error to clear, got %q", m.sessionPromptError)
	}
}
```

- [ ] **Step 2: Run UI tests and verify failure**

Run:

```bash
go test ./internal/ui
```

Expected: FAIL because helper and mode symbols are undefined.

- [ ] **Step 3: Add imports and UI state**

In `internal/ui/ui.go`, add imports:

```go
	"errors"
```

and add project imports:

```go
	"github.com/tiagokriok/Git-Fuzzy/internal/tmux"
```

Add mode definitions near constants:

```go
type uiMode int

const (
	modeRepositoryList uiMode = iota
	modeSessionPrompt
)
```

Extend `Model`:

```go
	mode               uiMode
	statusMessage      string
	sessionNameInput   string
	sessionPromptError string
```

Update `NewModel` to set:

```go
		mode:         modeRepositoryList,
```

- [ ] **Step 4: Add session prompt helper methods**

Add these functions to `internal/ui/ui.go` before `handleKeyPress`:

```go
func validateSessionNameInput(name string) error {
	if strings.TrimSpace(name) == "" {
		return errors.New("session name required")
	}
	return nil
}

func (m *Model) startSessionPrompt() {
	m.mode = modeSessionPrompt
	m.sessionNameInput = ""
	m.sessionPromptError = ""
	m.statusMessage = ""
}

func (m *Model) cancelSessionPrompt() {
	m.mode = modeRepositoryList
	m.sessionNameInput = ""
	m.sessionPromptError = ""
}
```

- [ ] **Step 5: Run focused UI tests and verify helper pass**

Run:

```bash
go test ./internal/ui -run 'TestValidateSessionNameInput|TestModelStartSessionPrompt|TestModelCancelSessionPrompt'
```

Expected: PASS.

- [ ] **Step 6: Render session prompt and footer status**

Modify `View()` near the top after width calculations or before panel rendering:

```go
	if m.mode == modeSessionPrompt {
		return m.renderSessionPrompt()
	}
```

Add this render method:

```go
func (m Model) renderSessionPrompt() string {
	boxWidth := min(max(m.width-8, 40), 80)

	titleStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	inputStyle := lipgloss.NewStyle().Border(lipgloss.RoundedBorder()).BorderForeground(lipgloss.Color("240")).Padding(0, 1).Width(boxWidth - 4)
	errorStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	input := inputStyle.Render(m.sessionNameInput)
	lines := []string{
		titleStyle.Render("New tmux session"),
		"Session name:",
		input,
	}

	if m.sessionPromptError != "" {
		lines = append(lines, errorStyle.Render(m.sessionPromptError))
	}

	lines = append(lines, footerStyle.Render("Enter: create | Esc: cancel | Ctrl+C: exit"))

	panel := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color("240")).
		Padding(1, 2).
		Width(boxWidth).
		Render(lipgloss.JoinVertical(lipgloss.Left, lines...))

	return lipgloss.Place(m.width, m.height, lipgloss.Center, lipgloss.Center, panel)
}
```

Modify `renderFooter()`:

```go
func (m Model) renderFooter() string {
	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Align(lipgloss.Center)
	footerText := "↑/↓: nav repos | Shift+↑/↓: scroll status | Enter: open | Alt+w/v/h/s: tmux | ^O: files | ^T: term | ^B: remote | ^G: refresh | Esc: exit"
	if m.statusMessage != "" {
		footerText = m.statusMessage + " | " + footerText
	}
	return footerStyle.Render(footerText)
}
```

- [ ] **Step 7: Implement tmux action dispatch helpers**

Add these methods before `handleKeyPress`:

```go
func (m *Model) selectedRepositoryPath() (string, bool) {
	if len(m.filtered) == 0 || m.selectedIdx >= len(m.filtered) {
		return "", false
	}
	return m.filtered[m.selectedIdx].Path, true
}

func (m *Model) executeTmuxAction(action config.TmuxOpenAction) (tea.Model, tea.Cmd) {
	repoPath, ok := m.selectedRepositoryPath()
	if !ok {
		return m, nil
	}

	if action != config.TmuxOpenEditor && !tmux.IsAvailable() {
		m.statusMessage = "tmux is not available"
		return m, nil
	}

	var err error
	switch action {
	case config.TmuxOpenWindow:
		err = tmux.OpenWindow(repoPath)
	case config.TmuxOpenVerticalPane:
		err = tmux.OpenVerticalPane(repoPath)
	case config.TmuxOpenHorizontalPane:
		err = tmux.OpenHorizontalPane(repoPath)
	case config.TmuxOpenSession:
		m.startSessionPrompt()
		return m, nil
	case config.TmuxOpenEditor:
		selected := m.filtered[m.selectedIdx]
		selectedRepository = &selected
		return m, tea.Quit
	default:
		selected := m.filtered[m.selectedIdx]
		selectedRepository = &selected
		return m, tea.Quit
	}

	if err != nil {
		m.statusMessage = err.Error()
		return m, nil
	}

	selectedRepository = nil
	return m, tea.Quit
}

func (m *Model) submitSessionPrompt() (tea.Model, tea.Cmd) {
	if err := validateSessionNameInput(m.sessionNameInput); err != nil {
		m.sessionPromptError = err.Error()
		return m, nil
	}

	repoPath, ok := m.selectedRepositoryPath()
	if !ok {
		m.cancelSessionPrompt()
		return m, nil
	}

	if err := tmux.OpenSession(strings.TrimSpace(m.sessionNameInput), repoPath); err != nil {
		m.sessionPromptError = err.Error()
		return m, nil
	}

	selectedRepository = nil
	return m, tea.Quit
}
```

- [ ] **Step 8: Route key handling for session prompt and tmux shortcuts**

At the top of `handleKeyPress`, before the existing switch, add:

```go
	if m.mode == modeSessionPrompt {
		switch msg.String() {
		case "ctrl+c":
			selectedRepository = nil
			return m, tea.Quit
		case "esc":
			m.cancelSessionPrompt()
			return m, nil
		case "enter":
			return m.submitSessionPrompt()
		case "backspace":
			if len(m.sessionNameInput) > 0 {
				m.sessionNameInput = m.sessionNameInput[:len(m.sessionNameInput)-1]
			}
			m.sessionPromptError = ""
			return m, nil
		default:
			m.sessionNameInput += msg.String()
			m.sessionPromptError = ""
			return m, nil
		}
	}
```

Add cases to the main switch:

```go
	case "alt+w":
		return m.executeTmuxAction(config.TmuxOpenWindow)

	case "alt+v":
		return m.executeTmuxAction(config.TmuxOpenVerticalPane)

	case "alt+h":
		return m.executeTmuxAction(config.TmuxOpenHorizontalPane)

	case "alt+s":
		return m.executeTmuxAction(config.TmuxOpenSession)
```

Replace the existing `enter` case with:

```go
	case "enter":
		if len(m.filtered) > 0 {
			if tmux.IsAvailable() {
				return m.executeTmuxAction(m.config.GetTmuxDefaultOpenAction())
			}

			selected := m.filtered[m.selectedIdx]
			selectedRepository = &selected
			return m, tea.Quit
		}
		return m, nil
```

- [ ] **Step 9: Run UI tests**

Run:

```bash
go test ./internal/ui
```

Expected: PASS.

- [ ] **Step 10: Run all current tests**

Run:

```bash
go test ./...
```

Expected: PASS.

- [ ] **Step 11: Run formatting**

Run:

```bash
go fmt ./internal/ui
```

Expected: no errors.

- [ ] **Step 12: Review checkpoint**

Run:

```bash
git diff -- internal/ui/ui.go internal/ui/tmux_actions_test.go
```

Expected: diff adds session prompt mode and tmux shortcuts without changing scanner, git status, or main editor launch flow. Do not commit unless the user explicitly asks.

---

### Task 4: Setup Wizard Tmux Default Action

**Files:**
- Modify: `internal/ui/setup.go`

**Interfaces:**
- Consumes: `config.TmuxOpenActionOptions()`
- Consumes: `config.TmuxOpenAction` constants
- Produces: saved `Config.TmuxDefaultOpenAction`

- [ ] **Step 1: Write setup action-label helper test**

Create or append to `internal/ui/tmux_actions_test.go`:

```go
func TestTmuxActionLabel(t *testing.T) {
	tests := []struct {
		name   string
		action config.TmuxOpenAction
		want   string
	}{
		{name: "editor", action: config.TmuxOpenEditor, want: "Current editor"},
		{name: "window", action: config.TmuxOpenWindow, want: "Tmux window"},
		{name: "vertical", action: config.TmuxOpenVerticalPane, want: "Tmux vertical pane"},
		{name: "horizontal", action: config.TmuxOpenHorizontalPane, want: "Tmux horizontal pane"},
		{name: "session", action: config.TmuxOpenSession, want: "Tmux session"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := tmuxActionLabel(tt.action); got != tt.want {
				t.Fatalf("expected %q, got %q", tt.want, got)
			}
		})
	}
}
```

Add this import to `internal/ui/tmux_actions_test.go`:

```go
import (
	"testing"

	"github.com/tiagokriok/Git-Fuzzy/internal/config"
)
```

- [ ] **Step 2: Run test and verify failure**

Run:

```bash
go test ./internal/ui -run TestTmuxActionLabel
```

Expected: FAIL because `tmuxActionLabel` is undefined.

- [ ] **Step 3: Extend setup model**

In `internal/ui/setup.go`, extend `SetupModel`:

```go
type SetupModel struct {
	step                int
	editor              textinput.Model
	paths               textinput.Model
	tmuxActionIdx       int
	tmuxDefaultActions  []config.TmuxOpenAction
	completed           *config.Config
	err                 error
}
```

Update `RunSetup()` model initialization:

```go
	model := SetupModel{
		step:               0,
		editor:             editorInput,
		paths:              pathsInput,
		tmuxDefaultActions: config.TmuxOpenActionOptions(),
	}
```

- [ ] **Step 4: Add setup action labels**

Add to `internal/ui/setup.go`:

```go
func tmuxActionLabel(action config.TmuxOpenAction) string {
	switch action {
	case config.TmuxOpenEditor:
		return "Current editor"
	case config.TmuxOpenWindow:
		return "Tmux window"
	case config.TmuxOpenVerticalPane:
		return "Tmux vertical pane"
	case config.TmuxOpenHorizontalPane:
		return "Tmux horizontal pane"
	case config.TmuxOpenSession:
		return "Tmux session"
	default:
		return "Current editor"
	}
}
```

- [ ] **Step 5: Update setup key handling**

In `SetupModel.Update`, replace the `enter` handling with a 3-step flow:

```go
		case "enter":
			if m.step == 0 {
				if m.editor.Value() == "" {
					m.err = fmt.Errorf("editor cannot be empty")
					return m, tea.Quit
				}
				m.step = 1
				m.paths.Focus()
				m.editor.Blur()
				return m, nil
			}

			if m.step == 1 {
				if m.paths.Value() == "" {
					m.err = fmt.Errorf("paths cannot be empty")
					return m, tea.Quit
				}
				m.step = 2
				m.paths.Blur()
				return m, nil
			}

			homeDir, _ := os.UserHomeDir()
			var searchPaths []string
			for path := range strings.SplitSeq(m.paths.Value(), ",") {
				path = strings.TrimSpace(path)
				if path != "" {
					if strings.HasPrefix(path, "~") {
						path = strings.Replace(path, "~", homeDir, 1)
					}
					searchPaths = append(searchPaths, path)
				}
			}

			selectedAction := config.TmuxOpenEditor
			if len(m.tmuxDefaultActions) > 0 && m.tmuxActionIdx < len(m.tmuxDefaultActions) {
				selectedAction = m.tmuxDefaultActions[m.tmuxActionIdx]
			}

			m.completed = &config.Config{
				Editor:                m.editor.Value(),
				SearchPaths:           searchPaths,
				TmuxDefaultOpenAction: string(selectedAction),
			}

			return m, tea.Quit
```

Extend `shift+tab` handling:

```go
		case "shift+tab":
			if m.step == 1 {
				m.step = 0
				m.editor.Focus()
				m.paths.Blur()
				return m, nil
			}
			if m.step == 2 {
				m.step = 1
				m.paths.Focus()
				return m, nil
			}
```

Add navigation for step 2:

```go
		case "up":
			if m.step == 2 && m.tmuxActionIdx > 0 {
				m.tmuxActionIdx--
			}
			return m, nil

		case "down":
			if m.step == 2 && m.tmuxActionIdx < len(m.tmuxDefaultActions)-1 {
				m.tmuxActionIdx++
			}
			return m, nil
```

At the end of `Update`, ensure only text inputs update:

```go
	if m.step == 0 {
		var cmd tea.Cmd
		m.editor, cmd = m.editor.Update(msg)
		return m, cmd
	}
	if m.step == 1 {
		var cmd tea.Cmd
		m.paths, cmd = m.paths.Update(msg)
		return m, cmd
	}
	return m, nil
```

- [ ] **Step 6: Add setup step 3 view**

Change `View()`:

```go
func (m SetupModel) View() string {
	if m.step == 0 {
		return m.editorView()
	}
	if m.step == 1 {
		return m.pathsView()
	}
	return m.tmuxActionView()
}
```

Add:

```go
func (m SetupModel) tmuxActionView() string {
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("205"))
	selectedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("46")).Bold(true)
	mutedStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240"))

	title := headerStyle.Render("🎯 Git Fuzzy Setup")
	subtitle := mutedStyle.Render("Step 3 of 3: Tmux Enter Action")

	var lines []string
	for i, action := range m.tmuxDefaultActions {
		label := tmuxActionLabel(action)
		if i == m.tmuxActionIdx {
			lines = append(lines, selectedStyle.Render("▶ "+label))
		} else {
			lines = append(lines, "  "+label)
		}
	}

	footerStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("240")).Padding(1, 0)
	footer := footerStyle.Render("↑/↓: choose | Enter: save | Shift+Tab: back | Ctrl+C: cancel")

	return fmt.Sprintf("%s\n%s\n\nDefault action for Enter while inside tmux:\n%s\n\n%s", title, subtitle, strings.Join(lines, "\n"), footer)
}
```

- [ ] **Step 7: Run setup/UI tests**

Run:

```bash
go test ./internal/ui
```

Expected: PASS.

- [ ] **Step 8: Format setup file**

Run:

```bash
go fmt ./internal/ui
```

Expected: no errors.

- [ ] **Step 9: Review checkpoint**

Run:

```bash
git diff -- internal/ui/setup.go internal/ui/tmux_actions_test.go
```

Expected: setup has a third step and tests cover action labels. Do not commit unless the user explicitly asks.

---

### Task 5: Documentation and Full Verification

**Files:**
- Modify: `README.md`

**Interfaces:**
- Consumes behavior implemented in Tasks 1 through 4.
- Produces user docs for tmux popup binding, shortcuts, and config field.

- [ ] **Step 1: Update README feature list**

In `README.md`, add this bullet under Key Features:

```markdown
- **Tmux Actions**: Open selected repositories in tmux windows, panes, or named sessions when running inside tmux
```

- [ ] **Step 2: Add tmux usage section**

Add this section after Basic Usage:

```markdown
### Tmux Actions

When `gitf` runs inside tmux, it can open the selected repository directly in tmux targets.

Shortcuts:

```text
Alt+w  Open in a new tmux window
Alt+v  Open in a vertical pane, side by side
Alt+h  Open in a horizontal pane, top/bottom
Alt+s  Prompt for a new tmux session name and switch to it
```

Tmux actions open the default tmux shell in the selected repository directory. They do not run the configured editor.

`Enter` keeps the normal editor behavior outside tmux. Inside tmux, `Enter` uses `tmux_default_open_action` from the config. Existing configs default to `editor`.

Example tmux popup binding:

```tmux
bind -r g display-popup -d '#{pane_current_path}' -w80% -h80% -E "zsh -ic 'exec gitf'"
```
```

- [ ] **Step 3: Update config example**

In the Linux/macOS config example, include:

```json
{
  "editor": "nvim",
  "search_paths": [
    "/home/user/dev",
    "/home/user/projects",
    "/home/user/work",
    "/opt/repositories"
  ],
  "tmux_default_open_action": "editor"
}
```

Add this explanation near the config examples:

```markdown
`tmux_default_open_action` controls what `Enter` does while `gitf` is running inside tmux. Supported values are `editor`, `tmux-window`, `tmux-vertical-pane`, `tmux-horizontal-pane`, and `tmux-session`.
```

- [ ] **Step 4: Run full formatting**

Run:

```bash
make fmt
```

Expected: PASS and no formatting errors.

- [ ] **Step 5: Run full verification**

Run:

```bash
make check
```

Expected: PASS for fmt, vet, and tests.

- [ ] **Step 6: Build binary**

Run:

```bash
make build
```

Expected: PASS and `./gitf` exists.

- [ ] **Step 7: Manual tmux smoke test**

Inside a tmux session, run:

```bash
./gitf
```

Expected:

- `Alt+w` creates and focuses a new tmux window in the selected repo.
- `Alt+v` creates and focuses a side-by-side pane in the selected repo.
- `Alt+h` creates and focuses a top/bottom pane in the selected repo.
- `Alt+s` opens the session prompt.
- Empty session name shows `session name required`.
- Existing session name shows `tmux session already exists: <name>`.
- Valid session name creates and switches to the session.

- [ ] **Step 8: Manual outside-tmux smoke test**

Outside tmux, run:

```bash
./gitf
```

Expected:

- `Enter` opens the selected repo in the configured editor.
- `Alt+w`, `Alt+v`, `Alt+h`, and `Alt+s` show `tmux is not available` without exiting.

- [ ] **Step 9: Final review checkpoint**

Run:

```bash
git status --short
git diff --stat
```

Expected: changes are limited to config, tmux package, UI, setup, tests, docs, and the design/plan docs. Do not commit unless the user explicitly asks.

---

## Self-Review

- Spec coverage: Task 1 covers config defaults and migration. Task 2 covers tmux availability and commands. Task 3 covers shortcuts, prompt behavior, errors, and UI exit behavior. Task 4 covers setup wizard. Task 5 covers README and verification.
- Placeholder scan: no placeholder markers or vague implementation placeholders are intentionally present.
- Type consistency: `config.TmuxOpenAction` constants are produced in Task 1 and consumed with the same names in Tasks 3 and 4. `tmux.OpenSession` signature is consistently `func OpenSession(name, repoPath string) error`.
