package app

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
)

func ReadPackageManifest(pkgPath string) (*Manifest, error) {
	f, err := os.Open(pkgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return readManifestFromReader(f)
}

func ReadEmbeddedManifest() (*Manifest, error) {
	return readManifestFromReader(bytes.NewReader(embeddedPackage))
}

func readManifestFromReader(r io.Reader) (*Manifest, error) {
	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	header, err := tr.Next()
	if err != nil {
		return nil, fmt.Errorf("read tar: %w", err)
	}

	if header.Name != ManifestName {
		return nil, fmt.Errorf("first entry is not %s, got %s", ManifestName, header.Name)
	}

	data := make([]byte, header.Size)
	if _, err := io.ReadFull(tr, data); err != nil {
		return nil, fmt.Errorf("read manifest data: %w", err)
	}

	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &m, nil
}

func Unpack(pkgPath, versionDir string) (*Manifest, error) {
	f, err := os.Open(pkgPath)
	if err != nil {
		return nil, err
	}
	defer f.Close()
	return unpackFromReader(f, versionDir)
}

func UnpackEmbedded(versionDir string) (*Manifest, error) {
	return unpackFromReader(bytes.NewReader(embeddedPackage), versionDir)
}

func unpackFromReader(r io.Reader, versionDir string) (*Manifest, error) {
	if _, err := os.Stat(versionDir); err == nil {
		fmt.Printf("Removing previous installation: %s\n", versionDir)
		if err := os.RemoveAll(versionDir); err != nil {
			return nil, fmt.Errorf("remove previous: %w", err)
		}
	}

	if err := os.MkdirAll(versionDir, 0755); err != nil {
		return nil, fmt.Errorf("create version dir: %w", err)
	}

	gz, err := gzip.NewReader(r)
	if err != nil {
		return nil, fmt.Errorf("gzip: %w", err)
	}
	defer gz.Close()

	tr := tar.NewReader(gz)

	var manifest *Manifest
	fileCount := 0

	for {
		header, err := tr.Next()
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, fmt.Errorf("read tar entry: %w", err)
		}

		if header.Name == ManifestName {
			data := make([]byte, header.Size)
			if _, err := io.ReadFull(tr, data); err != nil {
				return nil, fmt.Errorf("read manifest: %w", err)
			}
			var m Manifest
			if err := json.Unmarshal(data, &m); err != nil {
				return nil, fmt.Errorf("parse manifest: %w", err)
			}
			manifest = &m
			continue
		}

		target := filepath.Join(versionDir, filepath.FromSlash(header.Name))

		cleanTarget := filepath.Clean(target)
		cleanBase := filepath.Clean(versionDir) + string(os.PathSeparator)
		if !strings.HasPrefix(cleanTarget, cleanBase) && cleanTarget != filepath.Clean(versionDir) {
			return nil, fmt.Errorf("unsafe path in package: %s", header.Name)
		}

		switch header.Typeflag {
		case tar.TypeDir:
			if err := os.MkdirAll(target, 0755); err != nil {
				return nil, fmt.Errorf("mkdir %s: %w", header.Name, err)
			}
		case tar.TypeReg:
			if err := os.MkdirAll(filepath.Dir(target), 0755); err != nil {
				return nil, err
			}

			outFile, err := os.OpenFile(target, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, os.FileMode(header.Mode))
			if err != nil {
				return nil, fmt.Errorf("create %s: %w", header.Name, err)
			}

			written, err := io.Copy(outFile, io.LimitReader(tr, header.Size))
			outFile.Close()
			if err != nil {
				return nil, fmt.Errorf("write %s: %w", header.Name, err)
			}
			if written != header.Size {
				return nil, fmt.Errorf("short write %s: got %d, expected %d", header.Name, written, header.Size)
			}

			if !header.ModTime.IsZero() {
				_ = os.Chtimes(target, header.ModTime, header.ModTime)
			}

			fileCount++
		}
	}

	if manifest == nil {
		os.RemoveAll(versionDir)
		return nil, fmt.Errorf("package missing %s", ManifestName)
	}

	if err := manifest.Save(filepath.Join(versionDir, ManifestName)); err != nil {
		os.RemoveAll(versionDir)
		return nil, fmt.Errorf("save manifest: %w", err)
	}

	mismatches, err := manifest.Verify(versionDir)
	if err != nil {
		os.RemoveAll(versionDir)
		return nil, fmt.Errorf("verification error: %w", err)
	}

	if len(mismatches) > 0 {
		os.RemoveAll(versionDir)
		return nil, fmt.Errorf("hash mismatch for %d file(s): %s", len(mismatches), strings.Join(mismatches[:min(len(mismatches), 5)], ", "))
	}

	fmt.Printf("Extracted %d files, all hashes verified\n", fileCount)
	return manifest, nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
