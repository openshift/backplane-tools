package tools

import (
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"github.com/openshift/backplane-tools/pkg/tools/base"
)

var (
	errTestInstall        = errors.New("test install failure")
	errAnotherTestInstall = errors.New("another test install failure")
)

type installTestTool struct {
	name string
	err  error
}

func (t installTestTool) Name() string {
	return t.name
}

func (t installTestTool) ExecutableName() string {
	return t.name
}

func (t installTestTool) Install() error {
	return t.err
}

func (t installTestTool) Configure() error {
	return nil
}

func (t installTestTool) Remove() error {
	return nil
}

func (t installTestTool) Installed() (bool, error) {
	return false, nil
}

func (t installTestTool) InstalledVersion() (string, error) {
	return "", nil
}

func (t installTestTool) LatestVersion() (string, error) {
	return "", nil
}

func (t installTestTool) Cleanup() (string, error) {
	return "", nil
}

func TestInstallReturnsToolErrors(t *testing.T) {
	originalInstallDir := base.InstallDir
	originalLatestDir := base.LatestDir
	base.InstallDir = t.TempDir()
	base.LatestDir = filepath.Join(base.InstallDir, "latest")
	t.Cleanup(func() {
		base.InstallDir = originalInstallDir
		base.LatestDir = originalLatestDir
	})

	err := Install([]Tool{
		installTestTool{name: "failed-tool", err: errTestInstall},
		installTestTool{name: "another-failed-tool", err: errAnotherTestInstall},
	})
	if err == nil {
		t.Fatal("Install succeeded, want error")
	}
	if !strings.Contains(err.Error(), "failed-tool") || !strings.Contains(err.Error(), "another-failed-tool") {
		t.Fatalf("Install error does not include failed tool names: %v", err)
	}
}
