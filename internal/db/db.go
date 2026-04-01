package db

import (
	"database/sql"
	"log"
	"time"

	_ "github.com/mattn/go-sqlite3"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

type Image struct {
	Id            int    `json:"id"`
	FilePath      string `json:"file_path"`
	Filename      string `json:"filename"`
	CaptureDate   string `json:"capture_date"`
	Width         int    `json:"width"`
	Height        int    `json:"height"`
	Rating        int    `json:"rating"`
	MimeType      string `json:"mime_type"`
	ThumbnailPath string `json:"thumbnail_path"`
}

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

func UpsertImagePath(db *sql.DB, filePath walk.FileInfo, thumbPath string) (string, error) {
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
	INSERT INTO images (file_path, filename, file_size, mime_type, thumbnail_path, capture_date, width, height, index_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			file_size      = excluded.file_size,
			thumbnail_path = excluded.thumbnail_path,
			capture_date   = excluded.capture_date,
			width          = excluded.width,
			height         = excluded.height,
			index_at       = excluded.index_at
	`, filePath.Path, filePath.FileName, filePath.Size, filePath.MimeType, thumbPath, filePath.CaptureDate, filePath.Width, filePath.Height, time.Now().UTC())

	if err != nil {
		log.Printf("error inserting %s: %v", filePath.Path, err)
		return "", err
	}

	return action, nil
}

func GetImagesWithCount(db *sql.DB, limit, offset int) ([]Image, int, error) {
	var images []Image
	var count int

	err := db.QueryRow("SELECT COUNT(*) FROM images").Scan(&count)

	if err != nil {
		return nil, 0, err
	}

	rows, err := db.Query("SELECT id, file_path, filename, capture_date, width, height, rating, mime_type, thumbnail_path FROM images ORDER BY capture_date DESC LIMIT ? OFFSET ?", limit, offset)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var img Image
		rows.Scan(&img.Id, &img.FilePath, &img.Filename, &img.CaptureDate, &img.Width, &img.Height, &img.Rating, &img.MimeType, &img.ThumbnailPath)
		images = append(images, img)
	}

	return images, count, nil
}

func GetImagePath(db *sql.DB, id string) (string, string, error) {
	var filePath, mimeType string
	err := db.QueryRow("SELECT file_path, mime_type FROM images WHERE id = ?", id).Scan(&filePath, &mimeType)
	return filePath, mimeType, err
}
