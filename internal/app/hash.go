package app

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/cespare/xxhash/v2"
)

type Manifest struct {
	AppName   string               `json:"appName"`
	Version   string               `json:"version"`
	Arch      string               `json:"arch"`
	Exe       string               `json:"exe"`
	Splash    string               `json:"splash,omitempty"`
	Timestamp string               `json:"timestamp"`
	Files     map[string]FileEntry `json:"files"`
}

type FileEntry struct {
	Hash string `json:"hash"`
	Size int64  `json:"size"`
}

func ComputeFileHash(path string) (string, error) {
	f, err := os.Open(path)
	if err != nil {
		return "", err
	}
	defer f.Close()

	h := xxhash.New()
	if _, err := io.Copy(h, f); err != nil {
		return "", err
	}
	return fmt.Sprintf("%016x", h.Sum64()), nil
}

func GenerateManifest(rootDir, appName, version, arch, exeName, splashExt string) (*Manifest, error) {
	splashField := ""
	if splashExt != "" {
		splashField = SplashName + splashExt
	}
	manifest := &Manifest{
		AppName:   appName,
		Version:   version,
		Arch:      arch,
		Exe:       exeName,
		Splash:    splashField,
		Files:     make(map[string]FileEntry),
		Timestamp: time.Now().UTC().Format(time.RFC3339),
	}

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		if info.IsDir() {
			return nil
		}

		rel, err := filepath.Rel(rootDir, path)
		if err != nil {
			return err
		}
		rel = filepath.ToSlash(rel)

		if rel == ManifestName {
			return nil
		}

		hash, err := ComputeFileHash(path)
		if err != nil {
			return fmt.Errorf("hash %s: %w", rel, err)
		}
		manifest.Files[rel] = FileEntry{Hash: hash, Size: info.Size()}
		return nil
	})

	return manifest, err
}

func (m *Manifest) Save(path string) error {
	data, err := json.MarshalIndent(m, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0644)
}

func LoadManifest(path string) (*Manifest, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	var m Manifest
	if err := json.Unmarshal(data, &m); err != nil {
		return nil, err
	}
	return &m, nil
}

func (m *Manifest) Verify(rootDir string) ([]string, error) {
	var mismatches []string

	for rel, entry := range m.Files {
		fullPath := filepath.Join(rootDir, filepath.FromSlash(rel))
		actualHash, err := ComputeFileHash(fullPath)
		if err != nil {
			return nil, fmt.Errorf("verify %s: %w", rel, err)
		}
		if actualHash != entry.Hash {
			mismatches = append(mismatches, rel)
		}
	}

	return mismatches, nil
}

func (m *Manifest) VerifySingle(rootDir, relPath string) (bool, error) {
	entry, ok := m.Files[filepath.ToSlash(relPath)]
	if !ok {
		return false, fmt.Errorf("file not in manifest: %s", relPath)
	}

	fullPath := filepath.Join(rootDir, filepath.FromSlash(relPath))
	actualHash, err := ComputeFileHash(fullPath)
	if err != nil {
		return false, err
	}
	return actualHash == entry.Hash, nil
}
