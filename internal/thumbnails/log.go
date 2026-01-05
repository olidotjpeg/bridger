package thumbnails

import "log"

type Logger struct{}

func (l *Logger) Enqueue(assetID, path string) {
	log.Printf("Thumbnail queued for %s\n", path)
}
