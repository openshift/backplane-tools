/*
test defines various utilities to aid in testing packages
*/
package test

import (
	"io"
	"os"
	"path/filepath"

	"github.com/openshift/backplane-tools/pkg/source/github"
)

// dir is a temporary directory whose structure matches what's
// expected in ~/.local/bin/backplane/. (Essentially, it is a tempdir
// with a "latest" subdir contained within).
//
// After creating a new dir, it is the user's responsibility to
// call cleanup() to dispose of it's contents
type dir struct {
	Root string
	Latest string
}

// newDir builds a new temporary environment for testing.
//
// After creating a new dir, it is the user's responsibility to
// call cleanup() to dispose of the test artifacts
func newDir() (dir, error) {
	root, err := newDirRoot()
	if err != nil {
		return dir{}, err
	}

	latest, err := newLatestDir(root)
	e := dir {
		Root: root,
		Latest: latest,
	}
	return e, nil
}

// cleanup disposes of the test environment and it's contents
func (d dir) cleanup() error {
	return os.RemoveAll(d.Root)
}

// Empty returns true if the dir's .Root dir is empty (excluding the always-present
// 'latest' dir), false otherwise
func (d dir) Empty() (bool, error) {
	entries, err := os.ReadDir(d.Root)
	if err != nil {
		return false, err
	}
	// Always expect a 'latest' subdir
	if len(entries) != 1 {
		return false, nil
	}
	entryPath := filepath.Join(d.Root, entries[0].Name())
	if entryPath != d.Latest {
		return false, nil
	}
	return true, nil
}

// LatestEmpty returns true if the dir's .Latest dir is empty, false otherwise
func (d dir) LatestEmpty() (bool, error) {
	entries, err := os.ReadDir(d.Latest)
	if err != nil {
		return false, err
	}
	if len(entries) == 0 {
		return true, nil
	}
	return false, nil
}

// HasSubdir returns true if the provided subdir is present
func (d dir) HasSubdir(subdir string) (bool, error) {
	_, err := os.ReadDir(filepath.Join(d.Root, subdir))
	// If full path doesn't exist, this isn't a subdirectory contained in the test dir
	if err == os.ErrNotExist {
		return false, nil
	}
	// If there's an error unrelated to the existence of the subdir, return it
	if err != nil {
		return false, err
	}
	// Otherwise, the subdirectory must exist in the dir's testing environment
	return true, nil
}

// GetFile returns the file matching the provided path
func (d dir) Open(path string) (*os.File, error) {
	fullPath := filepath.Join(d.Root, path)
	return os.Open(fullPath)
}

// Mkdir creates a directory with the given permissions relative to the Root
func (d dir) MkdirAll(path string, perm os.FileMode) error {
	fullPath := filepath.Join(d.Root, path)
	return os.MkdirAll(fullPath, perm)
}

// RemoveAll removes the file at the provided path, as well as any children it contains
func (d dir) RemoveAll(path string) error {
	fullPath := filepath.Join(d.Root, path)
	return os.RemoveAll(fullPath)
}

// Create creates or truncates a file at the provided path. If the path contains subdirectories
// relative to the Root, Mkdir should be called first.
//
// As a part of the creation process, the file pointer is reset to the beginning of the file for simplified
// testing. This means, however, that subsequent writes to this file will overwrite the provided contents
func (d dir) Create(path string, perm os.FileMode, contents string) (*os.File, error) {
	fullPath := filepath.Join(d.Root, path)
	file, err := os.Create(fullPath)
	if err != nil {
		return file, err
	}
	err = file.Chmod(perm)
	if err != nil {
		return file, err
	}
	_, err = file.WriteString(contents)
	if err != nil {
		return file, err
	}
	err = file.Sync()
	if err != nil {
		return file, err
	}
	// Reset file pointer so that its contents can be read for testing
	_, err = file.Seek(0, io.SeekStart)
	if err != nil {
		return file, err
	}

	return file, nil
}

// CreateLink establishes a symlink between to files in the test environment
func (d dir) CreateLink(from, to string) error {
	fromPath := filepath.Join(d.Root, from)
	toPath := filepath.Join(d.Root, to)
	return os.Link(fromPath, toPath)
}

// IsLink determines whether the supplied path points to a valid symlink
func (d dir) IsLink(link string) (bool, error) {
	fullPath := filepath.Join(d.Root, link)
	linkInfo, err := os.Lstat(fullPath)
	if err != nil {
		return false, err
	}
	return linkInfo.Mode()&os.ModeSymlink == os.ModeSymlink, nil
}

func (d dir) EvalSymlink(path string) (string, error) {
	fullPath := filepath.Join(d.Root, path)
	return filepath.EvalSymlinks(fullPath)
}

// newTestDir builds the root dir of a dir
func newDirRoot() (string, error) {
	return os.MkdirTemp(os.TempDir(), "backplane-tools-")
}

// newLatestDir creates the directory "latest" in the provided root,
// in order to conform to the applications standard directory structure
func newLatestDir(root string) (string, error) {
	latestDirPath := filepath.Join(root, "latest")
	err := os.Mkdir(latestDirPath, os.FileMode(0755))
	return latestDirPath, err
}

// GithubEnv is a temporary environment containing both a
// pkg/source/github.TestSource and a directory whose structure
// matches that expected in ~/.local/bin/backplane/.
//
// After creating a new GithubEnv, it is the user's responsibility
// to call Cleanup() to dispose of it's contents
type GithubEnv struct {
	// TestSource provides a normal pkg/source/github.Source paired with
	// an httptest.Server for testing against
	*github.TestSource

	// dir contains references to the temporary directory owned
	// by this GithubEnv
	dir
}

// NewGithubEnv constructs a new GithubEnv to test against
//
// After creating a new GithubEnv, it is the user's responsibility
// to call Cleanup() to dispose of it's contents
func NewGithubEnv(owner, repo string) (*GithubEnv, error) {
	src, err := github.NewTestSource(owner, repo)
	if err != nil {
		return &GithubEnv{}, err
	}
	dir, err := newDir()
	if err != nil {
		return &GithubEnv{}, err
	}
	env := &GithubEnv{
		TestSource: src,
		dir: dir,
	}
	return env, nil
}

// Cleanup disposes of the test environment's components and their contents
func (e *GithubEnv) Cleanup() error {
	e.TestSource.Cleanup()
	return e.dir.cleanup()
}
