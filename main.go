package main

import (
	"database/sql"
	"io/fs"
	"log"
	"os"
	"path/filepath"

	"github.com/olidotjpeg/bridger/internal/api"
	"github.com/olidotjpeg/bridger/internal/config"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/scanner"
	"github.com/olidotjpeg/bridger/internal/watcher"
	"github.com/wailsapp/wails/v2"
	"github.com/wailsapp/wails/v2/pkg/options"
	"github.com/wailsapp/wails/v2/pkg/options/assetserver"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(filepath.Dir(cfg.DBPath), 0700); err != nil {
		log.Fatal(err)
	}

	database, err := db.Database(cfg.DBPath)
	if err != nil {
		log.Fatal(err)
	}

	if err := os.MkdirAll(cfg.ThumbsPath, 0755); err != nil {
		log.Fatal(err)
	}

	migrationsFS, err := fs.Sub(migrations, "sql/migrations")
	if err != nil {
		log.Fatal(err)
	}

	if err := db.RunMigrations(database, migrationsFS); err != nil {
		log.Fatal(err)
	}

	needsSetup := config.NeedsSetup(cfg)
	state := &scanner.ScanState{}
	reconfigCh := make(chan config.Config, 1)

	if !needsSetup {
		reconfigCh <- *cfg
	}

	router := api.SetupRouter(database, state, api.Config{
		ThumbDir:   cfg.ThumbsPath,
		NeedsSetup: needsSetup,
		CurrentCfg: cfg,
		ReconfigCh: reconfigCh,
	})
	router.Static("/thumbs", cfg.ThumbsPath)

	startDevAPIServer(router)

	app := NewApp()
	go watchReconfig(reconfigCh, database, state, app.emitScanDone)

	if err := wails.Run(&options.App{
		Title:  "Bridger",
		Width:  1440,
		Height: 900,
		AssetServer: &assetserver.Options{
			Assets:  embeddedDist,
			Handler: router,
		},
		OnStartup: app.startup,
		Bind:      []interface{}{app},
	}); err != nil {
		log.Fatal(err)
	}
}

func watchReconfig(ch <-chan config.Config, database *sql.DB, state *scanner.ScanState, onScanDone func()) {
	var stopWatcher func()
	for cfg := range ch {
		if stopWatcher != nil {
			stopWatcher()
		}
		c := cfg
		state.TryStart()
		go func() {
			if err := scanner.RunScan(c.ScanDirs, c.ThumbsPath, database, state); err != nil {
				log.Printf("scan error: %v", err)
			}
			onScanDone()
		}()
		stop, err := watcher.Watch(cfg.ScanDirs, cfg.ThumbsPath, database, state)
		if err != nil {
			log.Printf("watcher: failed to start: %v", err)
			continue
		}
		stopWatcher = stop
	}
}
