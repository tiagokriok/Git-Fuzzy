package tmux

import (
	"errors"
	"os/exec"
	"reflect"
	"strings"
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
	originalIsSessionMissingErr := isSessionMissingErr

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
		// By default, has-session returns an error so SessionExists returns false.
		// We surface tmux's real "can't find session: <name>" message so the
		// production predicate can match it.
		for _, a := range args {
			if a == "has-session" {
				target := ""
				for i, arg := range args {
					if arg == "-t" && i+1 < len(args) {
						target = args[i+1]
					}
				}
				return &CmdError{
					Err:    &exec.ExitError{},
					Output: "can't find session: " + target,
				}
			}
		}
		return nil
	}
	// Default test override matches the production predicate: an error is
	// treated as "session missing" only when the wrapped output contains
	// tmux's standard "can't find session" message.
	isSessionMissingErr = func(err error) bool {
		var c *CmdError
		if !errors.As(err, &c) {
			return false
		}
		return strings.Contains(c.Output, "can't find session")
	}

	t.Cleanup(func() {
		lookupPath = originalLookupPath
		getenv = originalGetenv
		runCommand = originalRunCommand
		isSessionMissingErr = originalIsSessionMissingErr
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

func TestSessionExistsPropagatesUnexpectedHasSessionError(t *testing.T) {
	calls := resetTestHooks(t)

	// Restore the production predicate (output-based matching) so we exercise
	// the real path that distinguishes "session missing" from real failures.
	origIsMiss := isSessionMissingErr
	isSessionMissingErr = func(err error) bool {
		var c *CmdError
		if !errors.As(err, &c) {
			return false
		}
		return strings.Contains(c.Output, "can't find session")
	}
	t.Cleanup(func() { isSessionMissingErr = origIsMiss })

	// runCommand fails with an *exec.ExitError (the same error type tmux uses
	// for "session not found") but the wrapped output does NOT contain
	// "can't find session" — simulating a real failure such as a server or
	// permission error. The production code must NOT treat this as missing.
	runCommand = func(name string, args ...string) error {
		*calls = append(*calls, commandCall{name: name, args: append([]string(nil), args...)})
		return &CmdError{
			Err:    &exec.ExitError{Stderr: []byte("error connecting to /tmp/tmux-1000/default (Permission denied)")},
			Output: "error connecting to /tmp/tmux-1000/default (Permission denied)",
		}
	}

	exists, err := SessionExists("my-session")
	if exists {
		t.Fatal("expected exists=false for failed has-session")
	}
	if err == nil {
		t.Fatal("expected unexpected has-session failure to be propagated")
	}
	if !strings.Contains(err.Error(), "Permission denied") {
		t.Fatalf("expected original error message to surface, got %v", err)
	}
}

func TestSessionExistsRejectsEmptyNameBeforeAvailabilityCheck(t *testing.T) {
	resetTestHooks(t)

	// Force tmux to appear unavailable. If empty-name validation runs after
	// the availability check, we'd get ErrUnavailable instead of
	// ErrEmptySessionName.
	getenv = func(key string) string { return "" }
	lookupPath = func(file string) (string, error) { return "", errors.New("not found") }

	_, err := SessionExists("   ")
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, ErrEmptySessionName) {
		t.Fatalf("expected empty session name error, got %v", err)
	}
}

func TestOpenSessionProceedsOnCantFindSessionOutput(t *testing.T) {
	calls := resetTestHooks(t)

	// Restore the production predicate so we exercise the real
	// "can't find session" detection path.
	origIsMiss := isSessionMissingErr
	isSessionMissingErr = func(err error) bool {
		var c *CmdError
		if !errors.As(err, &c) {
			return false
		}
		return strings.Contains(c.Output, "can't find session")
	}
	t.Cleanup(func() { isSessionMissingErr = origIsMiss })

	// has-session returns tmux's real "can't find session" output, wrapped
	// in *CmdError with an *exec.ExitError — the canonical missing-session
	// error. OpenSession must proceed to new-session and switch-client.
	runCommand = func(name string, args ...string) error {
		*calls = append(*calls, commandCall{name: name, args: append([]string(nil), args...)})
		for _, a := range args {
			if a == "has-session" {
				return &CmdError{
					Err:    &exec.ExitError{},
					Output: "can't find session: work",
				}
			}
		}
		return nil
	}

	if err := OpenSession("work", "/repo"); err != nil {
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
