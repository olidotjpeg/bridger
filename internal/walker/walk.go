package walk

import (
	"io/fs"
	"log"
	"mime"
	"path/filepath"
	"strings"
	"time"
)

type FileInfo struct {
	Path        string
	Size        int64
	FileName    string
	MimeType    string
	CaptureDate time.Time
	Width       int
	Height      int
}

func WalkDirectory(walkingPath string) ([]FileInfo, error) {
	var paths []FileInfo
	var extensions = []string{".png", ".jpg", ".jpeg"}

	err := filepath.WalkDir(walkingPath, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			if path == walkingPath {
				return err // propagate root path errors
			}
			log.Printf("Skipping %s: %v", path, err)
			return nil // skip errors on individual files
		}

		info, err := d.Info()

		if err != nil {
			log.Printf("Error while getting d.Info %s: %v", path, err)
			return nil
		}

		if d.IsDir() {
			return nil // Skip directories just in case
		}

		if !hasExtension(path, extensions) {
			return nil
		}

		absoluteFilePath, err := filepath.Abs(path)

		if err != nil {
			log.Printf("Error while getting filepath.Abs %s: %v", path, err)
			return nil
		}

		currentPath := FileInfo{
			Path:     absoluteFilePath,
			Size:     info.Size(),
			FileName: filepath.Base(path),
			MimeType: mime.TypeByExtension(strings.ToLower(filepath.Ext(path))),
		}

		paths = append(paths, currentPath)

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
