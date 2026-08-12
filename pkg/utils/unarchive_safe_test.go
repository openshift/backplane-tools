package utils

import (
	"path/filepath"
	"testing"
)

func TestSafeJoinUnder_ValidPath(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	got, err := safeJoinUnder(dest, "bin/oc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := filepath.Join(dest, "bin", "oc")
	if got != want {
		t.Fatalf("got %q, want %q", got, want)
	}
}

func TestSafeJoinUnder_EscapesDestination(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	_, err := safeJoinUnder(dest, "../../outside.txt")
	if err == nil {
		t.Fatal("expected error for path traversal")
	}
}

func TestSafeJoinUnder_AbsolutePath(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	_, err := safeJoinUnder(dest, "/etc/passwd")
	if err == nil {
		t.Fatal("expected error for absolute archive path")
	}
}

func TestSafeJoinUnder_EmptyName(t *testing.T) {
	t.Parallel()

	dest := t.TempDir()
	_, err := safeJoinUnder(dest, "")
	if err == nil {
		t.Fatal("expected error for empty name")
	}
}
