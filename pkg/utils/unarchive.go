package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

// safeJoinUnder resolves name relative to destination and rejects paths that escape
// destination after cleaning (zip-slip / tar-slip mitigation).
func safeJoinUnder(destination, name string) (string, error) {
	if name == "" {
		return "", errors.New("archive entry has empty name")
	}
	if filepath.IsAbs(name) {
		return "", fmt.Errorf("archive entry has absolute path: %q", name)
	}

	dest := filepath.Clean(destination)
	target := filepath.Clean(filepath.Join(dest, name))

	if target != dest && !strings.HasPrefix(target, dest+string(os.PathSeparator)) {
		return "", fmt.Errorf("archive entry escapes destination: %q", name)
	}

	return target, nil
}

// Unarchive decompresses and extracts the contents of .tar.gz bundles to the specified destination
func Unarchive(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open tarball '%s': %w", source, err)
	}
	defer func() {
		err = src.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close '%s': %v\n", src.Name(), err)
		}
	}()
	uncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to read the gzip file '%s': %w", source, err)
	}
	defer func() {
		err = uncompressed.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close gzip file '%s': %s", source, err.Error())
		}
	}()
	arc := tar.NewReader(uncompressed)
	var f *tar.Header
	for {
		f, err = arc.Next()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read from archive '%s': %w", source, err)
		}

		switch f.Typeflag {
		case tar.TypeDir:
			dirPath, err := safeJoinUnder(destination, f.Name)
			if err != nil {
				return err
			}
			err = os.MkdirAll(dirPath, f.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create a directory: %w", err)
			}
		case tar.TypeReg:
			// Sometimes tarballs don't include dir entries for their subdirectories
			// (looking at you, gcloud).
			if strings.Contains(f.Name, "/") {
				fileSubDir := filepath.Dir(f.Name)
				fileSubDirPath, err := safeJoinUnder(destination, fileSubDir)
				if err != nil {
					return err
				}
				err = os.MkdirAll(fileSubDirPath, os.FileMode(0o755))
				if err != nil {
					return fmt.Errorf("failed to create parent directory: %w", err)
				}
			}

			err = extractFile(destination, f, arc)
			if err != nil {
				return fmt.Errorf("failed to extract files: %w", err)
			}
		default:
			return fmt.Errorf("unsupported tar entry type %v for %q", f.Typeflag, f.Name)
		}
	}
	return nil
}

// Unzip extracts files from a zip archive to the specified destination directory.
func Unzip(source string, destination string) error {
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func(reader *zip.ReadCloser) {
		err := reader.Close()
		if err != nil {
			fmt.Fprintf(os.Stderr, "possible memory leak: failed to close %s", source)
		}
	}(reader)

	if err := os.MkdirAll(destination, os.ModePerm); err != nil {
		return err
	}

	for _, file := range reader.File {
		filePath, err := safeJoinUnder(destination, file.Name)
		if err != nil {
			return err
		}

		if file.FileInfo().IsDir() {
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		inputFile, err := file.Open()
		if err != nil {
			return err
		}

		outputFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			inputFile.Close()
			return err
		}

		if _, err := io.Copy(outputFile, inputFile); err != nil {
			inputFile.Close()
			outputFile.Close()
			return err
		}

		inputFile.Close()
		outputFile.Close()
	}

	return nil
}

func extractFile(destination string, f *tar.Header, arc io.Reader) error {
	path, err := safeJoinUnder(destination, f.Name)
	if err != nil {
		return err
	}
	return WriteFile(arc, path, os.FileMode(f.Mode))
}
