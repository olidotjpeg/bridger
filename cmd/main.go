package main

import (
	"flag"
	"fmt"
	"log"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/exif"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

func main() {
	walkDir, dbPath := setupCLIFlags()
	vips.Startup(nil)
	defer vips.Shutdown()

	database, err := db.Database(dbPath)

	if err != nil {
		log.Fatal(err)
	}

	err = db.RunMigrations(database, "./sql/migrations")

	if err != nil {
		log.Fatal(err)
	}

	var inserted, updated, skipped, errors int

	fmt.Printf("Scanning %s...\n", walkDir)

	results, err := walk.WalkDirectory(walkDir)
	if err != nil {
		log.Fatal(err)
	}

	fmt.Printf("Found %d files\n", len(results))

	for _, result := range results {
		if exifData, err := exif.ExtractEXIF(result.Path); err == nil {
			result.CaptureDate = exifData.CaptureDate
			result.Width = exifData.Width
			result.Height = exifData.Height
		}

		action, err := db.UpsertImagePath(database, result)
		if err != nil {
			errors++
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
	}

	fmt.Printf("  Inserted: %d\n", inserted)
	fmt.Printf("  Updated:  %d\n", updated)
	fmt.Printf("  Skipped:  %d\n", skipped)
	fmt.Printf("  Errors:   %d\n", errors)
	fmt.Println("Done.")
}

func setupCLIFlags() (string, string) {
	dir := flag.String("dir", ".", "Root Directory to Scan")
	dbPath := flag.String("db", "./bridger.db", "Path to the SQLite Database File")
	flag.Parse()

	return *dir, *dbPath
}
