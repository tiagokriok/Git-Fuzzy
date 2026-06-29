package ui

import (
	"testing"

	"github.com/tiagokriok/Git-Fuzzy/internal/config"
)

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
