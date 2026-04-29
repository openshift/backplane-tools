package base

import (
	"debug/buildinfo"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/openshift/backplane-tools/pkg/utils"
)

type GoBin struct {
	Default
	Module string
	Branch string
}

func (g *GoBin) LatestVersion() (string, error) {
	return g.Branch, nil
}

func (g *GoBin) Install() error {
	goBin, err := exec.LookPath("go")
	if err != nil {
		return fmt.Errorf("failed to locate 'go' on $PATH: the Go toolchain is required to install %s: %w", g.Name(), err)
	}

	ref := g.Module + "@" + g.Branch
	cmd := exec.Command(goBin, "install", ref)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		return fmt.Errorf("failed to run 'go install %s': %w", ref, err)
	}

	gobin, err := resolveGOBIN(goBin)
	if err != nil {
		return err
	}
	builtBinary := filepath.Join(gobin, g.ExecutableName())

	if _, err := os.Stat(builtBinary); err != nil {
		return fmt.Errorf("expected binary at '%s' after 'go install', but it was not found: %w", builtBinary, err)
	}

	version, err := extractModuleVersion(builtBinary, g.Module)
	if err != nil {
		return err
	}

	toolDir := g.ToolDir()
	versionedDir := filepath.Join(toolDir, version)
	if err := os.MkdirAll(versionedDir, os.FileMode(0o755)); err != nil {
		return fmt.Errorf("failed to create version-specific directory '%s': %w", versionedDir, err)
	}

	src, err := os.Open(builtBinary)
	if err != nil {
		return fmt.Errorf("failed to open built binary '%s': %w", builtBinary, err)
	}
	defer src.Close()

	destPath := filepath.Join(versionedDir, g.ExecutableName())
	if err := utils.WriteFile(src, destPath, 0o755); err != nil {
		return fmt.Errorf("failed to copy binary to '%s': %w", destPath, err)
	}

	latestFilePath := g.SymlinkPath()
	if err := os.Remove(latestFilePath); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove existing '%s' symlink at '%s': %w", g.ExecutableName(), LatestDir, err)
	}

	if err := os.Symlink(destPath, latestFilePath); err != nil {
		return fmt.Errorf("failed to link new '%s' binary to '%s': %w", g.ExecutableName(), LatestDir, err)
	}

	return nil
}

func resolveGOBIN(goBin string) (string, error) {
	out, err := exec.Command(goBin, "env", "GOBIN").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'go env GOBIN': %w", err)
	}
	gobin := strings.TrimSpace(string(out))
	if gobin != "" {
		return gobin, nil
	}

	out, err = exec.Command(goBin, "env", "GOPATH").Output()
	if err != nil {
		return "", fmt.Errorf("failed to run 'go env GOPATH': %w", err)
	}
	gopath := strings.TrimSpace(string(out))
	if gopath == "" {
		return "", errors.New("both GOBIN and GOPATH are empty; cannot determine where 'go install' placed the binary")
	}
	return filepath.Join(gopath, "bin"), nil
}

func extractModuleVersion(binaryPath, module string) (string, error) {
	info, err := buildinfo.ReadFile(binaryPath)
	if err != nil {
		return "", fmt.Errorf("failed to read build info from '%s': %w", binaryPath, err)
	}
	if info.Main.Path == module {
		return info.Main.Version, nil
	}
	return "", fmt.Errorf("module '%s' not found in build info of '%s'", module, binaryPath)
}
