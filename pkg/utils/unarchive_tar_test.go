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

func TestUnarchive_SymlinkRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "symlink.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithSymlink(source, "link", "../outside"); err != nil {
		t.Fatalf("failed to create symlink archive: %v", err)
	}

	err := Unarchive(source, dest)
	if err == nil {
		t.Fatal("expected Unarchive to reject symlink entries")
	}
	if !strings.Contains(err.Error(), "unsupported tar entry type") {
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
