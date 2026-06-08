package scanner

import (
	"database/sql"
	"fmt"
	"sync"

	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/exif"
	"github.com/olidotjpeg/bridger/internal/raw"
	"github.com/olidotjpeg/bridger/internal/thumbs"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

type ScanState struct {
	mu        sync.Mutex // the lock (unexported, nobody touches this directly)
	Running   bool
	Processed int
	Total     int
	Errors    int
}

type ScanStatus struct {
	Running   bool `json:"running"`
	Processed int  `json:"processed"`
	Total     int  `json:"total"`
	Errors    int  `json:"errors"`
}

func (s *ScanState) Status() ScanStatus {
	s.mu.Lock()
	defer s.mu.Unlock()
	return ScanStatus{
		Running:   s.Running,
		Processed: s.Processed,
		Total:     s.Total,
		Errors:    s.Errors,
	}
}

func (s *ScanState) IsRunning() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.Running
}

// IncrementProcessed increments the Processed counter by one.
// Safe to call from any goroutine (e.g. the file watcher).
func (s *ScanState) IncrementProcessed() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Processed++
}

// TryStart atomically checks and sets Running to true.
// Returns false if a scan is already in progress.
func (s *ScanState) TryStart() bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Running {
		return false
	}
	s.Running = true
	return true
}

func RunScan(thumbDir string, database *sql.DB, state *ScanState) error {
	dirProjectMap, err := db.GetDirProjectMap(database)
	if err != nil {
		state.mu.Lock()
		state.Running = false
		state.mu.Unlock()
		return fmt.Errorf("failed to load project dirs: %w", err)
	}

	if len(dirProjectMap) == 0 {
		state.mu.Lock()
		state.Running = false
		state.mu.Unlock()
		return nil
	}

	var inserted, updated, skipped, errors int

	type fileWithProject struct {
		info      walk.FileInfo
		projectID int
	}

	var results []fileWithProject
	var scannedDirs []string
	for dir, projectID := range dirProjectMap {
		fmt.Printf("Scanning %s...\n", dir)
		dirResults, err := walk.WalkDirectory(dir, thumbDir)
		if err != nil {
			fmt.Printf("  Skipping %s: %v\n", dir, err)
			continue
		}
		for _, r := range dirResults {
			results = append(results, fileWithProject{info: r, projectID: projectID})
		}
		scannedDirs = append(scannedDirs, dir)
	}

	fmt.Printf("Found %d files\n", len(results))

	state.mu.Lock()
	state.Total = len(results)
	state.Processed = 0
	state.Errors = 0
	state.mu.Unlock()

	for _, entry := range results {
		result := entry.info
		if exifData, err := exif.ExtractEXIF(result.Path); err == nil {
			result.EXIFData = *exifData
		}

		previewPath := ""
		thumbPath := ""
		if raw.IsRaw(result.MimeType) {
			if p, err := raw.GeneratePreview(result.Path, thumbDir); err == nil {
				previewPath = p
				thumbPath, _ = thumbs.GenerateThumbnail(p, thumbDir)
			}
		} else {
			thumbPath, _ = thumbs.GenerateThumbnail(result.Path, thumbDir)
		}

		action, err := db.UpsertImagePath(database, result, thumbPath, previewPath, entry.projectID)
		if err != nil {
			errors++
			state.mu.Lock()
			state.Errors++
			state.mu.Unlock()
			continue
		}
		switch action {
		case "inserted":
			inserted++
		case "updated":
			updated++
		case "skipped":
			skipped++
		}

		state.mu.Lock()
		state.Processed++
		state.mu.Unlock()
	}

	foundPaths := make(map[string]bool, len(results))
	for _, r := range results {
		foundPaths[r.info.Path] = true
	}
	if pruned, err := db.PruneStaleEntries(database, scannedDirs, foundPaths); err != nil {
		fmt.Printf("  Prune error: %v\n", err)
	} else if pruned > 0 {
		fmt.Printf("  Pruned:   %d\n", pruned)
	}

	state.mu.Lock()
	state.Running = false
	state.mu.Unlock()

	fmt.Printf("  Inserted: %d\n", inserted)
	fmt.Printf("  Updated:  %d\n", updated)
	fmt.Printf("  Skipped:  %d\n", skipped)
	fmt.Printf("  Errors:   %d\n", errors)
	fmt.Println("Done.")
	return nil
}
