package indexer

import "time"

type Job struct {
	Path string
	Info FileInfo
}

type FileInfo struct {
	Size       int64
	ModifiedAt time.Time
}

type Progress struct {
	Scanned int
	Indexed int
	Errors  int
	Path    string
}
