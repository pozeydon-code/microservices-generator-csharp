package generator

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
)

// Manifest tracks what was generated and from which config
type Manifest struct {
	ConfigHash string                    `json:"configHash"`
	Files      map[string]FileDependency `json:"files"`
}

// FileDependency tracks which config paths generated a file
type FileDependency struct {
	ConfigPaths []string `json:"configPaths"`
	Hash        string   `json:"hash"`
}

// LoadManifest loads the manifest from the output directory
func LoadManifest(outputDir string) (*Manifest, error) {
	manifestPath := filepath.Join(outputDir, ".microgen", "manifest.json")
	data, err := os.ReadFile(manifestPath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil // No manifest exists yet
		}
		return nil, fmt.Errorf("read manifest: %w", err)
	}

	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return nil, fmt.Errorf("parse manifest: %w", err)
	}

	return &manifest, nil
}

// SaveManifest saves the manifest to the output directory
func SaveManifest(outputDir string, manifest *Manifest) error {
	manifestDir := filepath.Join(outputDir, ".microgen")
	if err := os.MkdirAll(manifestDir, 0755); err != nil {
		return fmt.Errorf("create manifest directory: %w", err)
	}

	manifestPath := filepath.Join(manifestDir, "manifest.json")
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal manifest: %w", err)
	}

	if err := os.WriteFile(manifestPath, data, 0644); err != nil {
		return fmt.Errorf("write manifest: %w", err)
	}

	return nil
}

// HashConfig computes a hash of the config
func HashConfig(cfg interface{}) (string, error) {
	data, err := json.Marshal(cfg)
	if err != nil {
		return "", fmt.Errorf("marshal config: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// HashFile computes a hash of file content
func HashFile(path string) (string, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", fmt.Errorf("read file: %w", err)
	}

	hash := sha256.Sum256(data)
	return fmt.Sprintf("%x", hash), nil
}

// NeedsRegeneration checks if a file needs to be regenerated
func (m *Manifest) NeedsRegeneration(path string, configPaths []string, currentConfigHash string) bool {
	if m == nil {
		return true // No manifest, regenerate everything
	}

	fileDep, exists := m.Files[path]
	if !exists {
		return true // File not in manifest, needs generation
	}

	// If config hash changed, check if any of this file's dependencies changed
	if m.ConfigHash != currentConfigHash {
		// For simplicity, if any config path is in the changed set, regenerate
		// A more sophisticated approach would track individual path changes
		return true
	}

	return false
}
