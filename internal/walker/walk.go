package walk

import (
	"io/fs"
	"log"
	"path/filepath"
	"strings"
)

func WalkDirectory(walkingPath string) ([]string, error) {
	var paths []string
	var extensions = []string{".png", ".jpg", ".jpeg"}

	err := filepath.WalkDir(walkingPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("Skipping %s: %v", path, err)
			return nil // This continues walking
		}

		if d.IsDir() {
			return nil // Skip directories just in case
		}

		if !hasExtension(path, extensions) {
			return nil
		}

		paths = append(paths, path)

		return nil
	})

	return paths, err
}

func hasExtension(path string, extensions []string) bool {
	ext := strings.ToLower(filepath.Ext(path))

	for _, e := range extensions {
		if ext == e {
			return true
		}
	}

	return false
}
