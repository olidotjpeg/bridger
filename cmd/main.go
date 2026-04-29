package main

import (
	"database/sql"
	"flag"
	"log"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"time"

	"github.com/davidbyttow/govips/v2/vips"
	"github.com/olidotjpeg/bridger/internal/api"
	"github.com/olidotjpeg/bridger/internal/config"
	"github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/scanner"
	"github.com/olidotjpeg/bridger/internal/watcher"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	dir := flag.String("dir", "", "Root directory to scan (overrides config)")
	dbPath := flag.String("db", cfg.DBPath, "Path to SQLite database")
	thumbsDir := flag.String("thumbs", cfg.ThumbsPath, "Directory to store thumbnails")
	flag.Parse()

	if *dir != "" {
		cfg.ScanDirs = []string{*dir}
	}
	cfg.DBPath = *dbPath
	cfg.ThumbsPath = *thumbsDir

	needsSetup := config.NeedsSetup(cfg) && *dir == ""

	vips.Startup(nil)
	defer vips.Shutdown()

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

	if err := db.RunMigrations(database, "./sql/migrations"); err != nil {
		log.Fatal(err)
	}

	state := &scanner.ScanState{}
	reconfigCh := make(chan config.Config, 1)

	go watchReconfig(reconfigCh, database, state)

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
	serveStaticFiles(router)

	if needsSetup {
		go openBrowserWhenReady(browserURL())
	}

	router.Run()
}

func watchReconfig(ch <-chan config.Config, database *sql.DB, state *scanner.ScanState) {
	var stopWatcher func()
	for cfg := range ch {
		if stopWatcher != nil {
			stopWatcher()
		}
		c := cfg
		go func() {
			if err := scanner.RunScan(c.ScanDirs, c.ThumbsPath, database, state); err != nil {
				log.Printf("scan error: %v", err)
			}
		}()
		stop, err := watcher.Watch(cfg.ScanDirs, cfg.ThumbsPath, database, state)
		if err != nil {
			log.Printf("watcher: failed to start: %v", err)
			continue
		}
		stopWatcher = stop
	}
}

func openBrowserWhenReady(url string) {
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://localhost:8080/api/ping")
		if err == nil && resp.StatusCode == 200 {
			resp.Body.Close()
			openBrowser(url)
			return
		}
		time.Sleep(100 * time.Millisecond)
	}
	log.Println("server did not start in time; skipping browser open")
}

func openBrowser(url string) {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", url)
	case "linux":
		cmd = exec.Command("xdg-open", url)
	case "windows":
		cmd = exec.Command("rundll32", "url.dll,FileProtocolHandler", url)
	}
	if cmd != nil {
		_ = cmd.Start()
	}
}
