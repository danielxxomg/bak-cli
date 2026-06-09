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

	"github.com/danielxxomg/bak-cli/internal/paths"
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
func tarGzDir(dir string, w io.Writer) (retErr error) {
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
		// Normalize to forward slashes and clean for archive portability
		// using paths.CanonicalPath for consistent cross-platform canonical paths.
		rel = paths.CanonicalPath(rel)

		info, err := d.Info()
		if err != nil {
			return fmt.Errorf("stat %s: %w", rel, err)
		}

		hdr, err := tar.FileInfoHeader(info, "")
		if err != nil {
			return fmt.Errorf("tar header for %s: %w", rel, err)
		}
		hdr.Name = rel

		if d.IsDir() {
			if !strings.HasSuffix(hdr.Name, "/") {
				hdr.Name += "/"
			}
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("write tar header for %s: %w", rel, err)
			}
			return nil
		}

		// Handle symlinks.
		if info.Mode()&os.ModeSymlink != 0 {
			link, err := os.Readlink(walkPath)
			if err != nil {
				return fmt.Errorf("readlink %s: %w", rel, err)
			}
			hdr.Linkname = link
			hdr.Typeflag = tar.TypeSymlink
			if err := tw.WriteHeader(hdr); err != nil {
				return fmt.Errorf("write tar header for %s: %w", rel, err)
			}
			return nil
		}

		if err := tw.WriteHeader(hdr); err != nil {
			return fmt.Errorf("write tar header for %s: %w", rel, err)
		}

		//nolint:gosec // G122: Walk callback reads files for backup tar — backup tool must traverse directories
		f, err := os.Open(walkPath)
		if err != nil {
			return fmt.Errorf("open %s: %w", rel, err)
		}

		if _, err := io.Copy(tw, f); err != nil {
			if cerr := f.Close(); cerr != nil {
				return fmt.Errorf("copy %s: %w; close error: %w", rel, err, cerr)
			}
			return fmt.Errorf("copy %s: %w", rel, err)
		}

		if err := f.Close(); err != nil {
			return fmt.Errorf("close %s: %w", rel, err)
		}

		return nil
	})
}

// untarGzDir extracts a tar.gz from reader into targetDir.
func untarGzDir(r io.Reader, targetDir string) (retErr error) {
	gr, err := gzip.NewReader(r)
	if err != nil {
		return fmt.Errorf("gzip reader: %w", err)
	}
	defer func() {
		if cerr := gr.Close(); cerr != nil && retErr == nil {
			retErr = fmt.Errorf("close gzip reader: %w", cerr)
		}
	}()

	if err := os.MkdirAll(targetDir, 0755); err != nil {
		return fmt.Errorf("mkdir: %w", err)
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

		// Security: reject absolute paths in tar entries.
		if filepath.IsAbs(hdr.Name) || path.IsAbs(hdr.Name) {
			return fmt.Errorf("absolute path in tar entry not allowed: %s", hdr.Name)
		}

		target := filepath.Join(targetDir, filepath.FromSlash(hdr.Name))

		// Security: prevent path traversal using canonical path comparison.
		cleanTarget := paths.CanonicalPath(target)
		cleanDir := paths.CanonicalPath(targetDir) + "/"
		if !strings.HasPrefix(cleanTarget, cleanDir) {
			return fmt.Errorf("path traversal detected: %s", hdr.Name)
		}

		switch hdr.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return fmt.Errorf("mkdir %s: %w", hdr.Name, err)
			}

		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return fmt.Errorf("mkdir parent %s: %w", hdr.Name, err)
			}
			mode := hdr.FileInfo().Mode() // safe: fetches os.FileMode directly, no overflow
			//nolint:gosec // G115: hdr.Mode is tar permission bits (max 0777), fits within uint32
			f, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, mode)
			if err != nil {
				return fmt.Errorf("create %s: %w", hdr.Name, err)
			}
			//nolint:gosec // G110: io.Copy within untar — tar entries are bounded by archive size; restore is expected
			if _, err := io.Copy(f, tr); err != nil {
				if cerr := f.Close(); cerr != nil {
					return fmt.Errorf("write %s: %w; close error: %w", hdr.Name, err, cerr)
				}
				return fmt.Errorf("write %s: %w", hdr.Name, err)
			}
			if err := f.Close(); err != nil {
				return fmt.Errorf("close %s: %w", hdr.Name, err)
			}

		case tar.TypeSymlink:
			// Symlinks may fail on some platforms; log warning but continue.
			if err := os.Symlink(hdr.Linkname, target); err != nil {
				fmt.Fprintf(os.Stderr, "warning: symlink %s: %v\n", hdr.Name, err)
			}

		default:
			// Skip unknown entry types (devices, fifos, etc.).
		}
	}

	return nil
}
