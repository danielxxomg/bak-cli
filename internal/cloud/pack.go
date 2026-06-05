package cloud

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
)

// TarGzDirectory creates a base64-encoded tar.gz from the given
// directory. The returned string is suitable for use as Gist file
// content.
func TarGzDirectory(dir string) (string, error) {
	var buf bytes.Buffer
	if err := tarGzDir(dir, &buf); err != nil {
		return "", fmt.Errorf("tar.gz: %w", err)
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

// UntarGz decodes a base64-encoded tar.gz archive and extracts it
// into the target directory, creating it if needed.
func UntarGz(encoded string, targetDir string) error {
	data, err := base64.StdEncoding.DecodeString(encoded)
	if err != nil {
		return fmt.Errorf("untar.gz: decode base64: %w", err)
	}

	reader := bytes.NewReader(data)
	return untarGzDir(reader, targetDir)
}

// tarGzDir writes a gzipped tar of dir to w.
func tarGzDir(dir string, w io.Writer) error {
	gw := gzip.NewWriter(w)
	defer func() { _ = gw.Close() }()

	tw := tar.NewWriter(gw)
	defer func() { _ = tw.Close() }()

	// baseDir uses OS-specific separators for filepath.Rel comparison.
	baseDir := filepath.Clean(dir)

	return filepath.WalkDir(dir, func(walkPath string, d os.DirEntry, err error) error {
		if err != nil {
			return fmt.Errorf("walk: %w", err)
		}

		// Skip the root directory entry itself.
		if walkPath == baseDir {
			return nil
		}

		// Compute relative path for archive member.
		rel, err := filepath.Rel(baseDir, walkPath)
		if err != nil {
			return fmt.Errorf("relative path: %w", err)
		}
		// Normalize to forward slashes and clean for archive portability.
		rel = path.Clean(filepath.ToSlash(rel))

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", walkPath, err)
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("tar header for %s: %w", walkPath, err)
		}
		hdr.Name = rel

		if d.IsDir() {
			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("write tar header for %s: %w", walkPath, err)
			}
			return nil
		}

		// Handle symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(walkPath)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", walkPath, err)
			}
			hdr.Linkname = link
			hdr.Typeflag = tar.TypeSymlink
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("write tar header for %s: %w", walkPath, err)
			}
			return nil
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header for %s: %w", walkPath, err)
		}

		f, err := os.Open(walkPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", walkPath, err)
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("copy %s: %w", walkPath, err)
		}

		return nil
	})
}

// untarGzDir extracts a tar.gz from reader into targetDir.
func untarGzDir(r io.Reader, targetDir string) error {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer gr.Close()

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("mkdir %s: %w", targetDir, err)
	}

	tr := tar.NewReader(gr)
	for {
		hdr, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return fmt.Errorf("tar next: %w", err)
		}

		target := filepath.Join(targetDir, filepath.FromSlash(hdr.Name))

		// Security: prevent path traversal using OS-specific path comparison.
		// filepath.Clean is correct here (not path.Clean) because we compare
		// OS-specific absolute paths for security, not canonical normalization.
		cleanTarget := filepath.Clean(target)
		cleanDir := filepath.Clean(targetDir) + string(filepath.Separator)
		if !strings.HasPrefix(cleanTarget, cleanDir) {
			return fmt.Errorf("path traversal detected: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", target, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", target, err)
			}
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(hdr.Mode))
			if err != nil {
				return fmt.Errorf("create %s: %w", target, err)
			}
			if _, err := io.Copy(f, tr); err != nil {
				f.Close()
				return fmt.Errorf("write %s: %w", target, err)
			}
			f.Close()

		case tar.TypeSymlink:
			// Symlinks may fail on some platforms; skip without error.
			_ = os.Symlink(hdr.Linkname, target)

		default:
			// Skip unknown entry types (devices, fifos, etc.).
		}
	}

	return nil
}
