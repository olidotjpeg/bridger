package watcher

import (
	"database/sql"
	"io/fs"
	"log"
	"os"
	"path/filepath"
	"strings"

	"github.com/fsnotify/fsnotify"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/exif"
	"github.com/olidotjpeg/bridger/internal/raw"
	"github.com/olidotjpeg/bridger/internal/scanner"
	"github.com/olidotjpeg/bridger/internal/thumbs"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

var supportedExtensions = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".cr2":  true,
	".nef":  true,
	".arw":  true,
	".raf":  true,
}

// addDirRecursive walks root and registers every subdirectory with the watcher.
// Errors on individual directories are logged and skipped so a single
// unreadable folder does not abort the whole walk.
func addDirRecursive(w *fsnotify.Watcher, root string) {
	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			log.Printf("watcher: skipping %s: %v", path, err)
			return nil
		}
		if d.IsDir() {
			if watchErr := w.Add(path); watchErr != nil {
				log.Printf("watcher: could not watch %s: %v", path, watchErr)
			}
		}
		return nil
	})
	if err != nil {
		log.Printf("watcher: error walking %s: %v", root, err)
	}
}

// Watch watches watchDirs (and all subdirectories) for new image files and
// indexes them automatically. It runs until the returned stop function is called.
func Watch(watchDirs []string, thumbDir string, database *sql.DB, state *scanner.ScanState) (stop func(), err error) {
	w, err := fsnotify.NewWatcher()
	if err != nil {
		return nil, err
	}

	for _, dir := range watchDirs {
		addDirRecursive(w, dir)
		log.Printf("Watching %s (and subdirectories) for new files…", dir)
	}

	done := make(chan struct{})
	go func() {
		defer w.Close()
		for {
			select {
			case event, ok := <-w.Events:
				if !ok {
					return
				}
				if event.Has(fsnotify.Create) {
					// If a new directory was created, start watching it so
					// files subsequently dropped inside are detected.
					if info, err := os.Stat(event.Name); err == nil && info.IsDir() {
						addDirRecursive(w, event.Name)
						log.Printf("watcher: now watching new directory %s", event.Name)
					} else {
						handleNewFile(event.Name, thumbDir, database, state)
					}
				}
			case err, ok := <-w.Errors:
				if !ok {
					return
				}
				log.Printf("watcher error: %v", err)
			case <-done:
				return
			}
		}
	}()

	return func() { close(done) }, nil
}

func handleNewFile(path string, thumbDir string, database *sql.DB, state *scanner.ScanState) {
	ext := strings.ToLower(filepath.Ext(path))
	if !supportedExtensions[ext] {
		return
	}

	// Skip files inside the thumbs dir
	absPath, err := filepath.Abs(path)
	if err != nil {
		log.Printf("watcher: could not resolve path %s: %v", path, err)
		return
	}
	absThumbDir, err := filepath.Abs(thumbDir)
	if err == nil {
		rel, err := filepath.Rel(absThumbDir, absPath)
		if err == nil && !strings.HasPrefix(rel, "..") {
			return // inside thumbs dir, skip
		}
	}

	log.Printf("watcher: new file detected: %s", absPath)

	info, err := walk.StatFile(absPath)
	if err != nil {
		log.Printf("watcher: could not stat %s: %v", absPath, err)
		return
	}

	if exifData, err := exif.ExtractEXIF(absPath); err == nil {
		info.CaptureDate = exifData.CaptureDate
		info.Width = exifData.Width
		info.Height = exifData.Height
	}

	thumbPath, _ := thumbs.GenerateThumbnail(absPath, thumbDir)

	previewPath := ""
	if raw.IsRaw(info.MimeType) {
		if p, err := raw.GeneratePreview(absPath, thumbDir); err == nil {
			previewPath = p
		}
	}

	if _, err := db.UpsertImagePath(database, *info, thumbPath, previewPath); err != nil {
		log.Printf("watcher: upsert failed for %s: %v", absPath, err)
	} else {
		log.Printf("watcher: indexed %s", absPath)
		// Bump the processed counter on the shared state so the UI reflects activity
		state.IncrementProcessed()
	}
}
