package utils

import (
	"archive/tar"
	"compress/gzip"
	"errors"
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

func TestUnarchive_HardLink(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "hard-link.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithHardLink(source, "kubectl", "oc"); err != nil {
		t.Fatalf("failed to create hard-link archive: %v", err)
	}

	if err := Unarchive(source, dest); err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	original, err := os.Stat(filepath.Join(dest, "oc"))
	if err != nil {
		t.Fatalf("failed to stat original file: %v", err)
	}
	link, err := os.Stat(filepath.Join(dest, "kubectl"))
	if err != nil {
		t.Fatalf("failed to stat hard link: %v", err)
	}
	if !os.SameFile(original, link) {
		t.Fatal("extracted hard link does not reference the original file")
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

func TestUnarchive_Symlink(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "symlink.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithSymlinkAndTarget(
		source,
		"google-cloud-sdk/platform/bundledpythonunix/bin/idle3",
		"idle3.13",
		"google-cloud-sdk/platform/bundledpythonunix/bin/idle3.13",
	); err != nil {
		t.Fatalf("failed to create symlink archive: %v", err)
	}

	if err := Unarchive(source, dest); err != nil {
		t.Fatalf("Unarchive failed: %v", err)
	}

	linkPath := filepath.Join(dest, "google-cloud-sdk/platform/bundledpythonunix/bin/idle3")
	linkTarget, err := os.Readlink(linkPath)
	if err != nil {
		t.Fatalf("failed to read symlink: %v", err)
	}
	if linkTarget != "idle3.13" {
		t.Fatalf("symlink target is %q, want %q", linkTarget, "idle3.13")
	}
	if data, err := os.ReadFile(linkPath); err != nil || string(data) != "binary contents" {
		t.Fatalf("symlink does not resolve to its target: data=%q, err=%v", data, err)
	}
}

func TestUnarchive_SymlinkTraversalRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "evil-symlink.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithSymlink(source, "subdir/link", "../../outside"); err != nil {
		t.Fatalf("failed to create symlink archive: %v", err)
	}

	err := Unarchive(source, dest)
	if err == nil {
		t.Fatal("expected Unarchive to reject an escaping symlink")
	}
	if !strings.Contains(err.Error(), "escapes destination") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestUnarchive_AbsoluteSymlinkTargetRejected(t *testing.T) {
	t.Parallel()

	source := filepath.Join(t.TempDir(), "absolute-symlink.tar.gz")
	dest := t.TempDir()

	if err := writeTarGzWithSymlink(source, "link", "/etc/passwd"); err != nil {
		t.Fatalf("failed to create symlink archive: %v", err)
	}

	err := Unarchive(source, dest)
	if err == nil {
		t.Fatal("expected Unarchive to reject an absolute symlink target")
	}
	if !strings.Contains(err.Error(), "absolute path") {
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

func writeTarGzWithSymlinkAndTarget(path, name, linkTarget, target string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	gz := gzip.NewWriter(f)
	defer func() {
		err = errors.Join(err, gz.Close())
	}()

	tw := tar.NewWriter(gz)
	defer func() {
		err = errors.Join(err, tw.Close())
	}()

	if err := tw.WriteHeader(&tar.Header{
		Name:     name,
		Typeflag: tar.TypeSymlink,
		Linkname: linkTarget,
	}); err != nil {
		return err
	}

	content := []byte("binary contents")
	if err := tw.WriteHeader(&tar.Header{
		Name:     target,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		return err
	}
	_, err = tw.Write(content)
	return err
}

func writeTarGzWithHardLink(path, name, linkTarget string) (err error) {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer func() {
		err = errors.Join(err, f.Close())
	}()

	gz := gzip.NewWriter(f)
	defer func() {
		err = errors.Join(err, gz.Close())
	}()

	tw := tar.NewWriter(gz)
	defer func() {
		err = errors.Join(err, tw.Close())
	}()

	content := []byte("binary contents")
	if err := tw.WriteHeader(&tar.Header{
		Name:     linkTarget,
		Mode:     0o755,
		Size:     int64(len(content)),
		Typeflag: tar.TypeReg,
	}); err != nil {
		return err
	}
	if _, err := tw.Write(content); err != nil {
		return err
	}

	return tw.WriteHeader(&tar.Header{
		Name:     name,
		Typeflag: tar.TypeLink,
		Linkname: linkTarget,
	})
}
