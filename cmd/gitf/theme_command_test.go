package main

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/tiagokriok/Git-Fuzzy/internal/theme"
)

func TestRunThemeSyncOmarchyPrintsSuccess(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runThemeSyncOmarchy(&stdout, &stderr, func() (theme.Cache, error) {
		return theme.Cache{Name: "Tokyo Night"}, nil
	})
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if !strings.Contains(stdout.String(), "Synced Omarchy theme") {
		t.Fatalf("expected success output, got %q", stdout.String())
	}
	if stderr.String() != "" {
		t.Fatalf("expected empty stderr, got %q", stderr.String())
	}
}

func TestRunThemeSyncOmarchyReturnsError(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	err := runThemeSyncOmarchy(&stdout, &stderr, func() (theme.Cache, error) {
		return theme.Cache{}, errors.New("missing colors")
	})
	if err == nil {
		t.Fatal("expected error")
	}
	if stdout.String() != "" {
		t.Fatalf("expected empty stdout, got %q", stdout.String())
	}
	if !strings.Contains(stderr.String(), "missing colors") {
		t.Fatalf("expected stderr error, got %q", stderr.String())
	}
}
