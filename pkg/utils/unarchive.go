package utils

import (
	"archive/tar"
	"archive/zip"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

// Unarchive decompresses and extracts the contents of .tar.gz bundles to the specified destination
func Unarchive(source string, destination string) error {
	src, err := os.Open(source)
	if err != nil {
		return fmt.Errorf("failed to open tarball '%s': %v", source, err)
	}
	defer func() {
		err = src.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close '%s': %v\n", src.Name(), err)
		}
	}()
	uncompressed, err := gzip.NewReader(src)
	if err != nil {
		return fmt.Errorf("failed to read the gzip file '%s': %v", source, err)
	}
	defer func() {
		err = uncompressed.Close()
		if err != nil {
			fmt.Printf("WARNING: failed to close gzip file '%s': %v", source, err)
		}
	}()
	arc := tar.NewReader(uncompressed)
	var f *tar.Header
	for {
		f, err = arc.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("failed to read from archive '%s': %v", source, err)
		}
		if f.FileInfo().IsDir() {
			err = os.MkdirAll(filepath.Join(destination, f.Name), f.FileInfo().Mode())
			if err != nil {
				return fmt.Errorf("failed to create a directory : %v", err)
			}
		} else {
			err = extractFile(destination, f, arc)
			if err != nil {
				return fmt.Errorf("failed to extract files: %v", err)
			}
		}
	}
	return nil
}

// Unzip extracts files from a zip archive to the specified destination directory.
func Unzip(source string, destination string) error {
	// Open the zip archive for reading
	reader, err := zip.OpenReader(source)
	if err != nil {
		return err
	}
	defer func(reader *zip.ReadCloser) {
		err := reader.Close()
		if err != nil {

		}
	}(reader)

	// Create the destination directory if it doesn't exist
	if err := os.MkdirAll(destination, os.ModePerm); err != nil {
		return err
	}

	// Extract each file from the zip archive
	for _, file := range reader.File {
		filePath := filepath.Join(destination, file.Name)
		if file.FileInfo().IsDir() {
			// Create the directory if it doesn't exist
			err := os.MkdirAll(filePath, os.ModePerm)
			if err != nil {
				return err
			}
			continue
		}

		// Create the parent directory of the file if it doesn't exist
		if err := os.MkdirAll(filepath.Dir(filePath), os.ModePerm); err != nil {
			return err
		}

		// Open the file inside the zip archive
		inputFile, err := file.Open()
		if err != nil {
			return err
		}

		// Create the output file
		outputFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		// Copy the contents from the input file to the output file
		if _, err := io.Copy(outputFile, inputFile); err != nil {
			return err
		}
	}

	return nil
}

func extractFile(destination string, f *tar.Header, arc io.Reader) error {
	dst, err := os.Create(filepath.Join(destination, f.Name))
	if err != nil {
		return fmt.Errorf("failed to create file: %v", err)
	}
	defer func() {
		err = dst.Close()
		if err != nil {
			fmt.Printf("warning: failed to close '%s': %v\n", dst.Name(), err)
		}
	}()

	err = dst.Chmod(os.FileMode(f.Mode))
	if err != nil {
		return fmt.Errorf("failed to set permission on '%s': %v", dst.Name(), err)
	}
	_, err = dst.ReadFrom(arc)
	if err != nil {
		return fmt.Errorf("failed to read from archive  %v", err)
	}
	return nil
}
