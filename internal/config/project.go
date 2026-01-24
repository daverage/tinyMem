package config

import (
	"os"
	"path/filepath"
)

// FindProjectRoot looks for the .tinyMem directory starting from the current
// working directory and moving up the directory tree
func FindProjectRoot() (string, error) {
	currentDir, err := os.Getwd()
	if err != nil {
		return "", err
	}

	dir := currentDir
	for {
		tinyMemPath := filepath.Join(dir, ".tinyMem")
		if _, err := os.Stat(tinyMemPath); err == nil {
			// Found .tinyMem directory
			return dir, nil
		}

		// Move up one directory
		parentDir := filepath.Dir(dir)
		if parentDir == dir {
			// Reached the root of the filesystem
			break
		}
		dir = parentDir
	}

	// If no .tinyMem directory found, return current directory
	return currentDir, nil
}

// GetTinyMemDir returns the path to the .tinyMem directory relative to the project root
func GetTinyMemDir(projectRoot string) string {
	return filepath.Join(projectRoot, ".tinyMem")
}

// EnsureTinyMemDirs creates the necessary .tinyMem subdirectories
func EnsureTinyMemDirs(tinyMemDir string) error {
	subdirs := []string{
		filepath.Join(tinyMemDir, "logs"),
		filepath.Join(tinyMemDir, "run"),
		filepath.Join(tinyMemDir, "store"),
	}

	for _, subdir := range subdirs {
		if err := os.MkdirAll(subdir, 0755); err != nil {
			return err
		}
	}

	return nil
}