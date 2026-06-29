# Tmux Open Actions Design

## Summary

Add tmux-aware open actions to `gitf` while preserving the current editor behavior outside tmux. Users can keep `Enter` as the current editor flow, or configure `Enter` to open the selected repository in a tmux window, pane, or session when `gitf` is running inside tmux.

## Goals

- Preserve existing behavior for current users.
- Add direct tmux actions for selected repositories.
- Let users configure the default `Enter` action inside tmux during setup.
- Open tmux targets as shells in the selected repository directory.
- Keep tmux command logic isolated and testable.

## Non-goals

- Do not open the configured editor inside tmux targets.
- Do not add custom shell configuration for tmux actions.
- Do not create automatic tmux session names or templates.
- Do not change repository scanning or git status behavior.

## User behavior

### Enter behavior

`Enter` keeps the current behavior outside tmux. It returns the selected repository to `cmd/gitf/main.go`, which opens the configured editor as it does today.

Inside tmux, `Enter` uses the configured `tmux_default_open_action` value:

- `editor`: current editor behavior.
- `tmux-window`: open a new tmux window.
- `tmux-vertical-pane`: open a vertical split, shown side by side to the right.
- `tmux-horizontal-pane`: open a horizontal split, shown above or below.
- `tmux-session`: prompt for a new session name, create it, then switch to it.

If an existing config does not contain `tmux_default_open_action`, `gitf` treats it as `editor`.

### Direct tmux shortcuts

Add these shortcuts in the main repository list:

- `Alt+w`: open selected repository in a new tmux window.
- `Alt+v`: open selected repository in a vertical tmux pane, side by side.
- `Alt+h`: open selected repository in a horizontal tmux pane, top/bottom.
- `Alt+s`: prompt for a new tmux session name, then open selected repository in that session.

All tmux actions:

- Use the selected repository as the tmux working directory.
- Let tmux start its default shell.
- Focus the created window, pane, or session.
- Close `gitf` after a successful action.
- Show a discreet error and keep `gitf` open if tmux is unavailable or the command fails.

### Tmux session prompt

`Alt+s`, or `Enter` configured as `tmux-session`, opens an in-app prompt for the session name.

Prompt rules:

- The input starts empty.
- `Enter` with an empty name keeps the prompt open and shows a validation error.
- If the session already exists, show an error and ask for another name.
- `Esc` cancels the prompt and returns to the repository list.
- `Ctrl+C` exits `gitf`.

## Tmux availability

Tmux actions are available when both conditions are true:

1. The `TMUX` environment variable is present.
2. The `tmux` executable exists in `PATH`.

This is platform-neutral. It works anywhere those conditions are true, including WSL or MSYS environments.

Outside tmux:

- Direct tmux shortcuts show a discreet footer/status error and keep `gitf` open.
- `Enter` ignores tmux config and uses the current editor behavior.

## Config changes

Add a new optional field:

```go
type Config struct {
    Editor                string   `json:"editor"`
    SearchPaths           []string `json:"search_paths"`
    FileManager           string   `json:"file_manager,omitempty"`
    Terminal              string   `json:"terminal,omitempty"`
    TmuxDefaultOpenAction string   `json:"tmux_default_open_action,omitempty"`
}
```

Supported values:

- `editor`
- `tmux-window`
- `tmux-vertical-pane`
- `tmux-horizontal-pane`
- `tmux-session`

Config helpers should normalize missing or invalid values to `editor` to avoid breaking old configs.

## Setup wizard changes

The setup wizard becomes a 3-step flow:

1. Preferred editor.
2. Search paths.
3. Default `Enter` action inside tmux.

Step 3 should use a navigable list instead of free text. Options:

- Current editor.
- Tmux window.
- Tmux vertical pane.
- Tmux horizontal pane.
- Tmux session.

The saved config includes `tmux_default_open_action`.

## Architecture

### New package: `internal/tmux`

Create a focused package for tmux detection and command execution.

Suggested API:

```go
package tmux

func IsAvailable() bool
func OpenWindow(repoPath string) error
func OpenVerticalPane(repoPath string) error
func OpenHorizontalPane(repoPath string) error
func SessionExists(name string) (bool, error)
func OpenSession(name, repoPath string) error
```

Command behavior:

- Window: `tmux new-window -c <repo>`
- Vertical pane, side by side: `tmux split-window -h -c <repo>`
- Horizontal pane, top/bottom: `tmux split-window -v -c <repo>`
- Session: `tmux new-session -d -s <name> -c <repo>` then `tmux switch-client -t <name>`

The package should avoid shell string concatenation. Use `exec.Command` args to reduce quoting issues.

### UI changes

`internal/ui` should own interactive state:

- Main repository list mode.
- Session-name prompt mode.
- Footer/status error message.

The UI maps keyboard shortcuts to tmux package calls. On successful tmux action, it exits without returning a selected repository, so `main` does not open the editor afterward.

For `Enter`:

- Outside tmux: return selected repository as today.
- Inside tmux and default action is `editor`: return selected repository as today.
- Inside tmux and default action is tmux action: execute that tmux action in the UI and quit on success.

### Main flow

`cmd/gitf/main.go` should remain mostly unchanged:

- Load config.
- Scan repositories.
- Run UI.
- If UI returns a selected repository, open it in the configured editor.
- If UI returns nil, do nothing.

This keeps tmux-specific interactive behavior inside the UI and tmux command behavior inside `internal/tmux`.

## Error handling

- Missing tmux or missing `TMUX`: footer/status error.
- Failed tmux command: footer/status error.
- Empty session name: prompt-level error.
- Existing session name: prompt-level error.
- Failed session creation or switch: prompt-level error while prompt is active, otherwise footer/status error.

Errors should not crash the TUI unless the Bubbletea program itself fails.

## Testing plan

### Config tests

Add coverage for:

- Default config includes `editor` behavior for tmux default action.
- Missing JSON field resolves to `editor`.
- Invalid action resolves to `editor`.
- Valid action values are preserved.

### Tmux package tests

Design `internal/tmux` so command execution and environment lookup can be stubbed in tests.

Cover:

- Availability requires `TMUX` and `tmux` in `PATH`.
- Window command args.
- Vertical pane command args use `split-window -h`.
- Horizontal pane command args use `split-window -v`.
- Session creation checks existing session first.
- Existing session returns a clear error.
- Successful session creation switches client afterward.

### UI tests

Prefer focused tests over full terminal snapshots:

- Session prompt rejects empty input.
- Session prompt handles cancel with `Esc`.
- Tmux action success causes quit state without editor selection.
- Tmux unavailable sets footer/status error.

If these become brittle, keep UI logic small and cover behavior through helper functions.

## Documentation updates

Update README with:

- Tmux direct shortcuts.
- Explanation of inside-tmux versus outside-tmux behavior.
- `tmux_default_open_action` config field.
- Example tmux popup binding similar to:

```tmux
bind -r g display-popup -d '#{pane_current_path}' -w80% -h80% -E "zsh -ic 'exec gitf'"
```

## Acceptance criteria

- Existing `Enter` behavior is unchanged outside tmux.
- Existing configs continue to work without modification.
- Setup wizard lets users choose the default `Enter` action inside tmux.
- `Alt+w`, `Alt+v`, `Alt+h`, and `Alt+s` work inside tmux.
- Tmux actions open a shell in the selected repository directory.
- Tmux actions focus the created target and close `gitf` after success.
- Session creation always asks for a user-provided name.
- Empty or duplicate session names show errors and keep prompting.
- Relevant config and tmux behavior has automated tests.
