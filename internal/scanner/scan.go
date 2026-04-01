package scanner

import (
	"database/sql"
	"fmt"
	"log"
	"sync"

	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/exif"
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

func RunScan(walkDir string, thumbDir string, database *sql.DB, state *ScanState) {
	var inserted, updated, skipped, errors int

	fmt.Printf("Scanning %s...\n", walkDir)

	results, err := walk.WalkDirectory(walkDir, thumbDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d files\n", len(results))

	state.mu.Lock()
	state.Running = true
	state.Total = len(results)
	state.Processed = 0
	state.Errors = 0
	state.mu.Unlock()

	for _, result := range results {
		if exifData, err := exif.ExtractEXIF(result.Path); err == nil {
			result.CaptureDate = exifData.CaptureDate
			result.Width = exifData.Width
			result.Height = exifData.Height
		}

		thumbPath, _ := thumbs.GenerateThumbnail(result.Path, thumbDir)

		action, err := db.UpsertImagePath(database, result, thumbPath)
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

	state.mu.Lock()
	state.Running = false
	state.mu.Unlock()

	fmt.Printf("  Inserted: %d\n", inserted)
	fmt.Printf("  Updated:  %d\n", updated)
	fmt.Printf("  Skipped:  %d\n", skipped)
	fmt.Printf("  Errors:   %d\n", errors)
	fmt.Println("Done.")
}
