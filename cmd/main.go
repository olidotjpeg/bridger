package main

import (
	"flag"
	"log"
	"os"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/olidotjpeg/bridger/internal/api"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/scanner"
)

func main() {
	walkDir, dbPath, thumbDir := setupCLIFlags()
	vips.Startup(nil)
	defer vips.Shutdown()

	database, err := db.Database(dbPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(thumbDir, 0755); err != nil {
		log.Fatal(err)
	}

	if err := db.RunMigrations(database, "./sql/migrations"); err != nil {
		log.Fatal(err)
	}

	state := &scanner.ScanState{}

	go func() {
		if err := scanner.RunScan(walkDir, thumbDir, database, state); err != nil {
			log.Printf("scan error: %v", err)
		}
	}()

	router := api.SetupRouter(database, state, api.Config{
		WalkDir:  walkDir,
		ThumbDir: thumbDir,
	})

	router.Static("/thumbs", thumbDir)
	serveStaticFiles(router)

	router.Run()
}

func setupCLIFlags() (string, string, string) {
	dir := flag.String("dir", ".", "Root Directory to Scan")
	dbPath := flag.String("db", "./bridger.db", "Path to the SQLite Database File")
	thumbsDir := flag.String("thumbs", "./thumbs", "Directory to store thumbnails")

	flag.Parse()

	return *dir, *dbPath, *thumbsDir
}
