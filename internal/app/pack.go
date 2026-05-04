package app

import (
	"archive/tar"
	"compress/gzip"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/klauspost/compress/zstd"
	"github.com/ulikunitz/xz"
)

type Compression int

const (
	CompressionZstd Compression = iota
	CompressionGzip
	CompressionXZ
)

type PackOptions struct {
	Compression       Compression
	Level             int
	SplashMinMs       int
	LenientLockDetect bool
}

const SplashName = "_splash"

func Pack(srcDir, outputPath, appName, version, arch, exeName, splashPath string, opts ...PackOptions) error {
	opt := PackOptions{Compression: CompressionZstd, Level: 0}
	if len(opts) > 0 {
		opt = opts[0]
	}
	if info, err := os.Stat(srcDir); err != nil {
		return fmt.Errorf("source directory: %w", err)
	} else if !info.IsDir() {
		return fmt.Errorf("source is not a directory: %s", srcDir)
	}

	exePath := filepath.Join(srcDir, exeName)
	if _, err := os.Stat(exePath); err != nil {
		return fmt.Errorf("exe not found: %s: %w", exeName, err)
	}

	splashExt := ""
	if splashPath != "" {
		if _, err := os.Stat(splashPath); err != nil {
			return fmt.Errorf("splash not found: %s: %w", splashPath, err)
		}
		splashExt = strings.ToLower(filepath.Ext(splashPath))
	}

	manifest, err := GenerateManifest(srcDir, appName, version, arch, exeName, splashExt)
	if err != nil {
		return fmt.Errorf("generate manifest: %w", err)
	}
	if opt.SplashMinMs > 0 {
		manifest.SplashMinMs = opt.SplashMinMs
	}
	manifest.LenientLockDetect = opt.LenientLockDetect

	fmt.Printf("Packaging %d files (%s %s, arch %s)...\n", len(manifest.Files), appName, version, arch)

	if err := os.MkdirAll(filepath.Dir(outputPath), 0755); err != nil {
		return fmt.Errorf("create output dir: %w", err)
	}

	outFile, err := os.Create(outputPath)
	if err != nil {
		return fmt.Errorf("create output: %w", err)
	}
	defer outFile.Close()

	var compWriter io.WriteCloser
	switch opt.Compression {
	case CompressionGzip:
		level := gzip.DefaultCompression
		if opt.Level >= 1 && opt.Level <= 9 {
			level = opt.Level
		}
		compWriter, err = gzip.NewWriterLevel(outFile, level)
		if err != nil {
			return fmt.Errorf("gzip writer: %w", err)
		}
	case CompressionXZ:
		xzConfig := xz.WriterConfig{}
		if opt.Level >= 1 && opt.Level <= 9 {
			xzConfig.DictCap = xzDictCapForLevel(opt.Level)
		}
		compWriter, err = xzConfig.NewWriter(outFile)
		if err != nil {
			return fmt.Errorf("xz writer: %w", err)
		}
	default:
		zstdLevel := zstd.SpeedDefault
		if opt.Level >= 1 && opt.Level <= 19 {
			switch {
			case opt.Level <= 3:
				zstdLevel = zstd.SpeedFastest
			case opt.Level <= 7:
				zstdLevel = zstd.SpeedDefault
			case opt.Level <= 12:
				zstdLevel = zstd.SpeedBetterCompression
			default:
				zstdLevel = zstd.SpeedBestCompression
			}
		}
		compWriter, err = zstd.NewWriter(outFile, zstd.WithEncoderLevel(zstdLevel))
		if err != nil {
			return fmt.Errorf("zstd writer: %w", err)
		}
	}
	defer compWriter.Close()

	tw := tar.NewWriter(compWriter)
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

	if splashPath != "" && splashExt != "" {
		splashData, err := os.ReadFile(splashPath)
		if err != nil {
			return fmt.Errorf("read splash: %w", err)
		}
		splashEntry := SplashName + splashExt
		if err := tw.WriteHeader(&tar.Header{
			Name:    splashEntry,
			Size:    int64(len(splashData)),
			Mode:    0644,
			ModTime: time.Now(),
		}); err != nil {
			return fmt.Errorf("write splash header: %w", err)
		}
		if _, err := tw.Write(splashData); err != nil {
			return fmt.Errorf("write splash: %w", err)
		}
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
	if err := compWriter.Close(); err != nil {
		return err
	}

	outInfo, _ := os.Stat(outputPath)
	sizeMB := float64(outInfo.Size()) / 1024 / 1024
	fmt.Printf("Package created: %s (%.1f MB)\n", outputPath, sizeMB)

	return nil
}

func xzDictCapForLevel(level int) int {
	caps := []int{
		1 << 16,
		1 << 17,
		1 << 18,
		1 << 19,
		1 << 20,
		1 << 21,
		1 << 22,
		1 << 23,
		1 << 24,
	}
	idx := level - 1
	if idx < 0 {
		idx = 0
	}
	if idx >= len(caps) {
		idx = len(caps) - 1
	}
	return caps[idx]
}
