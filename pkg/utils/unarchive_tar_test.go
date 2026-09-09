package utils

import (
	"archive/tar"
	"compress/gzip"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestUnarchive_ValidTarGz(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "valid.tar.gz")
	dest := t.TempDir()

	if err := writeTarGz(source, map[string]string{
		"subdir/tool": "binary contents",
	}); err != nil {
		t.Fatalf("failed to create test archive: %v", err)
	}

	if err := Unarchive(source, dest); err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	extracted := filepath.Join(dest, "subdir", "tool")
	data, err := os.ReadFile(extracted)
	if err != nil {
		t.Fatalf("failed to read extracted file: %v", err)
	}
	if string(data) != "binary contents" {
		t.Fatalf("got %q, want %q", string(data), "binary contents")
	}
}

func TestUnarchive_PathTraversalRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "evil.tar.gz")
	dest := t.TempDir()
	outside := filepath.Join(filepath.Dir(dest), "zip-slip-outside.txt")

	if err := writeTarGz(source, map[string]string{
		"../../zip-slip-outside.txt": "pwned",
	}); err != nil {
		t.Fatalf("failed to create malicious archive: %v", err)
	}

	err := Unarchive(source, dest)
	if err == nil {
		t.Fatal("expected Unarchive to reject path traversal")
	}
	if _, err := os.Stat(outside); !os.IsNotExist(err) {
		os.Remove(outside)
		t.Fatal("path traversal wrote outside destination directory")
	}
}

func TestUnarchive_EscapingSymlinkRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "symlink.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithSymlink(source, "link", "../outside"); err != nil {
		t.Fatalf("failed to create symlink archive: %v", err)
	}

	err := Unarchive(source, dest)
	if err == nil {
		t.Fatal("expected Unarchive to reject symlink escaping destination")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, statErr := os.Lstat(filepath.Join(dest, "link")); !os.IsNotExist(statErr) {
		t.Fatal("escaping symlink should not have been created")
	}
}

func TestUnarchive_Symlink(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "symlink.tar.gz")
	dest := t.TempDir()

	// A symlink whose target stays within the destination is allowed.
	if err := writeTarGzWithEntries(source, []tarEntry{
		{name: "oc", content: "binary contents"},
		{name: "kubectl", typeflag: tar.TypeSymlink, linkname: "oc"},
	}); err != nil {
		t.Fatalf("failed to create symlink archive: %v", err)
	}

	if err := Unarchive(source, dest); err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	linkPath := filepath.Join(dest, "kubectl")
	target, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("expected symlink at %s: %v", linkPath, err)
	}
	if target != "oc" {
		t.Fatalf("got symlink target %q, want %q", target, "oc")
	}
	data, err := os.ReadFile(linkPath)
	if err != nil {
		t.Fatalf("failed to read through symlink: %v", err)
	}
	if string(data) != "binary contents" {
		t.Fatalf("got %q through symlink, want %q", string(data), "binary contents")
	}
}

func TestUnarchive_HardLink(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "hardlink.tar.gz")
	dest := t.TempDir()

	// Mirrors openshift-client 4.22.x: kubectl is a hard link to oc.
	if err := writeTarGzWithEntries(source, []tarEntry{
		{name: "oc", content: "binary contents"},
		{name: "kubectl", typeflag: tar.TypeLink, linkname: "oc"},
	}); err != nil {
		t.Fatalf("failed to create hard link archive: %v", err)
	}

	if err := Unarchive(source, dest); err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	data, err := os.ReadFile(filepath.Join(dest, "kubectl"))
	if err != nil {
		t.Fatalf("failed to read hard-linked file: %v", err)
	}
	if string(data) != "binary contents" {
		t.Fatalf("got %q, want %q", string(data), "binary contents")
	}

	// Confirm oc and kubectl share the same inode (i.e. it's a real hard link).
	ocInfo, err := os.Stat(filepath.Join(dest, "oc"))
	if err != nil {
		t.Fatalf("failed to stat oc: %v", err)
	}
	kubectlInfo, err := os.Stat(filepath.Join(dest, "kubectl"))
	if err != nil {
		t.Fatalf("failed to stat kubectl: %v", err)
	}
	if !os.SameFile(ocInfo, kubectlInfo) {
		t.Fatal("expected oc and kubectl to be the same file (hard link)")
	}
}

func TestUnarchive_EscapingHardLinkRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "hardlink.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithEntries(source, []tarEntry{
		{name: "link", typeflag: tar.TypeLink, linkname: "../../etc/passwd"},
	}); err != nil {
		t.Fatalf("failed to create malicious hard link archive: %v", err)
	}

	err := Unarchive(source, dest)
	if err == nil {
		t.Fatal("expected Unarchive to reject hard link escaping destination")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func writeTarGz(path string, files map[string]string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	for name, content := range files {
		hdr := &tar.Header{
			Name:     name,
			Mode:     0o644,
			Size:     int64(len(content)),
			Typeflag: tar.TypeReg,
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if _, err := tw.Write([]byte(content)); err != nil {
			return err
		}
	}

	return nil
}

type tarEntry struct {
	name     string
	content  string
	typeflag byte
	linkname string
}

// writeTarGzWithEntries writes a gzipped tar archive containing the given entries,
// in order. Entries with a zero typeflag default to a regular file.
func writeTarGzWithEntries(path string, entries []tarEntry) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	for _, e := range entries {
		typeflag := e.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		hdr := &tar.Header{
			Name:     e.name,
			Mode:     0o644,
			Typeflag: typeflag,
			Linkname: e.linkname,
		}
		if typeflag == tar.TypeReg {
			hdr.Size = int64(len(e.content))
		}
		if err := tw.WriteHeader(hdr); err != nil {
			return err
		}
		if typeflag == tar.TypeReg {
			if _, err := tw.Write([]byte(e.content)); err != nil {
				return err
			}
		}
	}

	return nil
}

func writeTarGzWithSymlink(path, name, linkTarget string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()

	gz := gzip.NewWriter(f)
	defer gz.Close()

	tw := tar.NewWriter(gz)
	defer tw.Close()

	hdr := &tar.Header{
		Name:     name,
		Mode:     0o644,
		Typeflag: tar.TypeSymlink,
		Linkname: linkTarget,
	}
	return tw.WriteHeader(hdr)
}
