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

	// Migrate existing scan_dirs from config into projects (one-time, on upgrade).
	if len(cfg.ScanDirs) > 0 {
		projects, _ := db.GetAllProjects(database)
		if len(projects) == 0 {
			for _, dir := range cfg.ScanDirs {
				proj, err := db.CreateProject(database, filepath.Base(dir))
				if err != nil && !db.IsConflict(err) {
					log.Printf("config migration: failed to create project for %s: %v", dir, err)
					continue
				}
				if db.IsConflict(err) {
					all, _ := db.GetAllProjects(database)
					for _, p := range all {
						if p.Name == filepath.Base(dir) {
							proj = p
							break
						}
					}
				}
				if err := db.AddDirToProject(database, proj.Id, dir); err != nil && !db.IsConflict(err) {
					log.Printf("config migration: failed to add dir %s: %v", dir, err)
				}
			}
			cfg.ScanDirs = nil
			if err := config.Save(cfg); err != nil {
				log.Printf("config migration: failed to save config: %v", err)
			}
		}
	}

	state := &scanner.ScanState{}
	reconfigCh := make(chan config.Config, 1)

	dirs, _ := db.GetAllScanDirs(database)
	if len(dirs) > 0 {
		reconfigCh <- *cfg
	}

	router := api.SetupRouter(database, state, api.Config{
		ThumbDir:   cfg.ThumbsPath,
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
			if err := scanner.RunScan(c.ThumbsPath, database, state); err != nil {
				log.Printf("scan error: %v", err)
			}
			onScanDone()
		}()
		dirs, err := db.GetAllScanDirs(database)
		if err != nil {
			log.Printf("watcher: failed to load dirs: %v", err)
			continue
		}
		if len(dirs) == 0 {
			continue
		}
		stop, err := watcher.Watch(dirs, cfg.ThumbsPath, database, state)
		if err != nil {
			log.Printf("watcher: failed to start: %v", err)
			continue
		}
		stopWatcher = stop
	}
}
