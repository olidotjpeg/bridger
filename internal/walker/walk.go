package walk

import (
	"io/fs"
	"log"
	"mime"
	"os"
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

var mimeTypes = map[string]string{
	".cr2": "image/x-canon-cr2",
	".nef": "image/x-nikon-nef",
	".arw": "image/x-sony-arw",
	".raf": "image/x-fuji-raf",
}

func WalkDirectory(walkingPath string, thumbDir string) ([]FileInfo, error) {
	var paths []FileInfo
	var extensions = []string{".png", ".jpg", ".jpeg", ".cr2", ".nef", ".arw", ".raf"}

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
			if filepath.Clean(path) == filepath.Clean(thumbDir) {
				return fs.SkipDir
			}
			return nil
		}

		if !hasExtension(path, extensions) {
			return nil
		}

		absoluteFilePath, err := filepath.Abs(path)

		if err != nil {
			log.Printf("Error while getting filepath.Abs %s: %v", path, err)
			return nil
		}

		mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
		if mimeType == "" {
			mimeType = mimeTypes[strings.ToLower(filepath.Ext(path))]
		}

		currentPath := FileInfo{
			Path:     absoluteFilePath,
			Size:     info.Size(),
			FileName: filepath.Base(path),
			MimeType: mimeType,
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

// StatFile returns a FileInfo for a single file path without walking a directory.
// This is used by the watcher to index individual newly-created files.
func StatFile(path string) (*FileInfo, error) {
	info, err := os.Stat(path)
	if err != nil {
		return nil, err
	}

	mimeType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if mimeType == "" {
		mimeType = mimeTypes[strings.ToLower(filepath.Ext(path))]
	}

	return &FileInfo{
		Path:     path,
		Size:     info.Size(),
		FileName: filepath.Base(path),
		MimeType: mimeType,
	}, nil
}
