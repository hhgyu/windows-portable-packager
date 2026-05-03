package app

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"
)

func Pack(srcDir, outputPath, appName, version, arch, exeName string) error {
	if info, err := os.Stat(srcDir); err != nil {
		return fmt.Errorf("source directory: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", srcDir)
	}

	exePath := filepath.Join(srcDir, exeName)
	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("exe not found: %s: %w", exeName, err)
	}

	manifest, err := GenerateManifest(srcDir, appName, version, arch, exeName)
	if err != nil {
		return fmt.Errorf("generate manifest: %w", err)
	}

	fmt.Printf("Packaging %d files (%s %s, arch %s)...\n", len(manifest.Files), appName, version, arch)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	gw := gzip.NewWriter(outFile)
	defer gw.Close()

	tw := tar.NewWriter(gw)
	defer tw.Close()

	manifestData, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return err
	}

	if err := tw.WriteHeader(&tar.Header{
		Name:    ManifestName,
		Size:    int64(len(manifestData)),
		Mode:    0644,
		ModTime: time.Now(),
	}); err != nil {
		return fmt.Errorf("write manifest header: %w", err)
	}
	if _, err := tw.Write(manifestData); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	err = filepath.Walk(srcDir, func(path string, info os.FileInfo, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}

		rel, err := filepath.Rel(srcDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if rel == "." || rel == ManifestName {
			return nil
		}

		if info.IsDir() {
			return tw.WriteHeader(&tar.Header{
				Typeflag: tar.TypeDir,
				Name:     rel + "/",
				Mode:     0755,
				ModTime:  info.ModTime(),
			})
		}

		if err := tw.WriteHeader(&tar.Header{
			Name:    rel,
			Size:    info.Size(),
			Mode:    0644,
			ModTime: info.ModTime(),
		}); err != nil {
			return fmt.Errorf("header %s: %w", rel, err)
		}

		f, err := os.Open(path)
		if err != nil {
			return err
		}
		defer f.Close()

		if _, err := io.Copy(tw, f); err != nil {
			return fmt.Errorf("write %s: %w", rel, err)
		}
		f.Close()
		return nil
	})

	if err != nil {
		return fmt.Errorf("walk source: %w", err)
	}

	if err := tw.Close(); err != nil {
		return err
	}
	if err := gw.Close(); err != nil {
		return err
	}

	outInfo, _ := os.Stat(outputPath)
	sizeMB := float64(outInfo.Size()) / 1024 / 1024
	fmt.Printf("Package created: %s (%.1f MB)\n", outputPath, sizeMB)

	return nil
}
