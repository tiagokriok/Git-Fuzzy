package tmux

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"strings"
)

// CmdError wraps a failed tmux invocation together with the combined
// stdout+stderr output. Callers (and the isSessionMissingErr predicate)
// can inspect the output to distinguish expected "not found" messages
// from real failures such as server/permission errors.
type CmdError struct {
	Err    error
	Output string
}

func (e *CmdError) Error() string {
	if e.Output != "" {
		return e.Output
	}
	return e.Err.Error()
}

func (e *CmdError) Unwrap() error { return e.Err }

var (
	ErrUnavailable      = errors.New("tmux is not available")
	ErrEmptySessionName = errors.New("session name cannot be empty")

	lookupPath = exec.LookPath
	getenv     = os.Getenv

	// runCommand invokes a tmux subcommand. It uses CombinedOutput so the
	// returned *CmdError preserves tmux's textual output, which is how we
	// distinguish "can't find session" (expected) from real failures.
	runCommand = func(name string, args ...string) error {
		out, err := exec.Command(name, args...).CombinedOutput()
		if err == nil {
			return nil
		}
		return &CmdError{Err: err, Output: string(out)}
	}

	// isSessionMissingErr reports whether the given error from `has-session`
	// indicates tmux could not find the target session. tmux's standard
	// "can't find session: <name>" message is the only signal we trust;
	// any other failure (server down, permission denied, etc.) is treated
	// as an unexpected error and propagated to the caller.
	isSessionMissingErr = func(err error) bool {
		var c *CmdError
		if !errors.As(err, &c) {
			return false
		}
		return strings.Contains(c.Output, "can't find session")
	}
)

// IsAvailable checks whether the current process is inside a tmux session
// and the tmux binary is reachable.
func IsAvailable() bool {
	if getenv("TMUX") == "" {
		return false
	}
	_, err := lookupPath("tmux")
	return err == nil
}

// OpenWindow creates a new tmux window in the given repo path.
func OpenWindow(repoPath string) error {
	if !IsAvailable() {
		return ErrUnavailable
	}
	return runCommand("tmux", "new-window", "-c", repoPath)
}

// OpenVerticalPane splits the current pane vertically (horizontal split)
// in the given repo path.
func OpenVerticalPane(repoPath string) error {
	if !IsAvailable() {
		return ErrUnavailable
	}
	return runCommand("tmux", "split-window", "-h", "-c", repoPath)
}

// OpenHorizontalPane splits the current pane horizontally (vertical split)
// in the given repo path.
func OpenHorizontalPane(repoPath string) error {
	if !IsAvailable() {
		return ErrUnavailable
	}
	return runCommand("tmux", "split-window", "-v", "-c", repoPath)
}

// SessionExists checks whether a tmux session with the given name exists.
func SessionExists(name string) (bool, error) {
	// Validate the name first so callers get a deterministic error
	// regardless of the surrounding environment, matching OpenSession.
	if strings.TrimSpace(name) == "" {
		return false, ErrEmptySessionName
	}

	if !IsAvailable() {
		return false, ErrUnavailable
	}

	err := runCommand("tmux", "has-session", "-t", name)
	if err == nil {
		return true, nil
	}
	if isSessionMissingErr(err) {
		return false, nil
	}
	return false, err
}

// OpenSession creates a new detached tmux session for the repo and then
// switches the client to it. Returns an error if the session already exists.
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
