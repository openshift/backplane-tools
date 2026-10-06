package utils

import (
	"runtime"
	"strings"
	"testing"
)

func TestGetLineInReaderMatchingKey(t *testing.T) {
	t.Parallel()

	data := "abc123  openshift-client-linux-4.22.8.tar.gz\n"
	line, err := GetLineInReaderMatchingKey(strings.NewReader(data), "openshift-client-linux-4.22.8.tar.gz")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(line, "abc123") {
		t.Fatalf("got line %q", line)
	}
}

func TestGetLineInReaderMatchingKey_NoMatch(t *testing.T) {
	t.Parallel()

	_, err := GetLineInReaderMatchingKey(strings.NewReader("foo bar\n"), "missing")
	if err == nil {
		t.Fatal("expected error for no match")
	}
}

func TestArchAliasesFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		arch string
		want []string
	}{
		{"amd64", []string{"amd64", "x86_64"}},
		// arm64 must alias to 'aarch64' as well: projects using Rust-style target
		// triples (eg - coreos/butane) publish arm64 assets under that name instead
		// of 'arm64' or 'arm'.
		{"arm64", []string{"arm64", "arm", "aarch64"}},
		{"ppc64le", []string{"ppc64le"}},
	}

	for _, tt := range tests {
		got := archAliasesFor(tt.arch)
		if len(got) != len(tt.want) {
			t.Fatalf("archAliasesFor(%q) = %v, want %v", tt.arch, got, tt.want)
		}
		for i := range got {
			if got[i] != tt.want[i] {
				t.Fatalf("archAliasesFor(%q) = %v, want %v", tt.arch, got, tt.want)
			}
		}
	}
}

func TestGetArchAliases_MatchesCurrentRuntimeArch(t *testing.T) {
	t.Parallel()

	// GetArchAliases() is a thin wrapper around archAliasesFor(runtime.GOARCH);
	// archAliasesFor itself is covered exhaustively above, so this just exercises
	// the public entry point to confirm it delegates correctly.
	got := GetArchAliases()
	want := archAliasesFor(runtime.GOARCH)
	if len(got) != len(want) {
		t.Fatalf("GetArchAliases() = %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Fatalf("GetArchAliases() = %v, want %v", got, want)
		}
	}
}

func TestGetArchAliases_ContainsAarch64ForArm64Assets(t *testing.T) {
	t.Parallel()

	// Regression test for ROSAENG-68831: coreos/butane publishes its arm64
	// release asset as 'butane-aarch64-unknown-linux-gnu', which previously
	// matched none of the arm64 aliases ('arm64', 'arm').
	aliases := archAliasesFor("arm64")
	if !ContainsAny("butane-aarch64-unknown-linux-gnu", aliases) {
		t.Fatalf("expected arm64 aliases %v to match butane's aarch64 asset name", aliases)
	}
}
