package actions

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
)

// RunExport creates a gzipped tar archive of the specified backup and writes
// a confirmation message to out.
func RunExport(homeDir, backupID, outputPath string, out io.Writer) error {
	// Validate backup ID format.
	if !isValidBackupID(backupID) {
		return fmt.Errorf("invalid backup ID %q (expected format: YYYYMMDD-HHMMSS, e.g. 20260604-150405)", backupID)
	}

	sourceDir := filepath.Join(homeDir, ".bak", "backups", backupID)

	// Verify the backup directory exists.
	info, err := os.Stat(sourceDir)
	if err != nil {
		if os.IsNotExist(err) {
			return fmt.Errorf("backup %q not found", backupID)
		}
		return fmt.Errorf("access backup dir: %w", err)
	}
	if !info.IsDir() {
		return fmt.Errorf("%q is not a directory", sourceDir)
	}

	// Create output file.
	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output file: %w", err)
	}

	// Write tar.gz.
	if err := CreateTarGz(sourceDir, outFile); err != nil {
		// Clean up partial output on error.
		closeErr := outFile.Close()
		removeErr := os.Remove(outputPath)
		if closeErr != nil {
			return fmt.Errorf("create archive: %w (also failed to close output: %v)", err, closeErr)
		}
		if removeErr != nil {
			return fmt.Errorf("create archive: %w (also failed to remove partial output: %v)", err, removeErr)
		}
		return fmt.Errorf("create archive: %w", err)
	}

	if err := outFile.Close(); err != nil {
		return fmt.Errorf("close output file: %w", err)
	}

	fmt.Fprintf(out, "Exported backup %q to %s\n", backupID, outputPath)
	return nil
}

// CreateTarGz creates a gzipped tar archive of the given directory,
// writing to the provided writer.
func CreateTarGz(srcDir string, w io.Writer) (retErr error) {
	gw := gzip.NewWriter(w)
	defer func() {
		if err := gw.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("close gzip writer: %w", err)
		}
	}()

	tw := tar.NewWriter(gw)
	defer func() {
		if err := tw.Close(); err != nil && retErr == nil {
			retErr = fmt.Errorf("close tar writer: %w", err)
		}
	}()

	return filepath.WalkDir(srcDir, func(walkPath string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}

		// Archive-relative path using path.Clean for canonical normalization.
		rel, err := filepath.Rel(filepath.Dir(srcDir), walkPath)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}
		rel = path.Clean(filepath.ToSlash(rel))

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %q: %w", walkPath, err)
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("tar header: %w", err)
		}
		hdr.Name = rel

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header: %w", err)
		}

		// Directories have no content.
		if d.IsDir() {
			return nil
		}

		// Copy file contents into the tar stream.
		src, err := os.Open(walkPath)
		if err != nil {
			return fmt.Errorf("open %q: %w", walkPath, err)
		}
		defer src.Close()

		if _, err := io.Copy(tw, src); err != nil {
			return fmt.Errorf("copy %q to tar: %w", walkPath, err)
		}

		return nil
	})
}

// isValidBackupID checks the format YYYYMMDD-HHMMSS.
func isValidBackupID(id string) bool {
	if len(id) != 15 || id[8] != '-' {
		return false
	}
	for i, c := range id {
		if i == 8 {
			continue
		}
		if c < '0' || c > '9' {
			return false
		}
	}
	return true
}
