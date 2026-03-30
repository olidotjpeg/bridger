package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

func Database(dbPath string) (*sql.DB, error) {
	db, err := sql.Open("sqlite3", dbPath)

	if err != nil {
		return db, err
	}

	if err := db.Ping(); err != nil {
		return db, err
	}

	return db, err
}

func UpsertImagePath(db *sql.DB, filePath walk.FileInfo) (string, error) {
	var existingSize int64
	err := db.QueryRow("SELECT file_size FROM images WHERE file_path = ?", filePath.Path).Scan(&existingSize)

	if err != nil && err != sql.ErrNoRows {
		log.Printf("error querying %s: %v", filePath.Path, err)
		return "", err
	}

	if err == nil && existingSize == filePath.Size {
		return "skipped", nil
	}

	action := "inserted"
	if err == nil {
		action = "updated"
	}

	_, err = db.Exec(`
	INSERT INTO images (file_path, filename, file_size, mime_type, capture_date, width, height, index_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			file_size    = excluded.file_size,
			capture_date = excluded.capture_date,
			width        = excluded.width,
			height       = excluded.height,
			index_at     = excluded.index_at
	`, filePath.Path, filePath.FileName, filePath.Size, filePath.MimeType, filePath.CaptureDate, filePath.Width, filePath.Height, time.Now().UTC())

	if err != nil {
		log.Printf("error inserting %s: %v", filePath.Path, err)
		return "", err
	}

	return action, nil
}
