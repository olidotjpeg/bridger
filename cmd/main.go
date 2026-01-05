package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/olidotjpeg/bridge-clone/internal/catalog"
	"github.com/olidotjpeg/bridge-clone/internal/indexer"
	"github.com/olidotjpeg/bridge-clone/internal/metadata"
	"github.com/olidotjpeg/bridge-clone/internal/thumbnails"
)

func main() {
	if len(os.Args) < 2 {
		log.Fatal("Usage: photoapp <folder-to-index>")
	}

	root := os.Args[1]

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sigs := make(chan os.Signal, 1)
	signal.Notify(sigs, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sigs
		log.Println("...Shutting down...")
		cancel()
	}()

	store, err := catalog.OpenSQLite("catalog.db")
	if err != nil {
		log.Fatal(err)
	}
	meta := &metadata.NoopReader{}
	thumbs := &thumbnails.Logger{}

	idx := &indexer.Indexer{
		Workers:      4,
		JobQueueSize: 100,
	}

	// Dependency Injector
	idx.SetStore(store)
	idx.SetMetadata(meta)
	idx.SetThumbnailer(thumbs)

	go func() {
		for p := range idx.Progress() {
			if p.Scanned > 0 {
				log.Printf("Scanned: %s\n", p.Path)
			}
			if p.Indexed > 0 {
				log.Printf("Indexed: %s\n", p.Path)
			}
			if p.Errors > 0 {
				log.Printf("Error on: %s\n", p.Path)
			}
		}
	}()

	if err := idx.Run(ctx, []string{root}); err != nil {
		log.Fatalf("Indexing failed: %v", err)
	}

	log.Println("Indexing complete")
}
