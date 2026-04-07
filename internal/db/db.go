package db

import (
	"database/sql"
	"errors"
	"fmt"
	"log"
	"time"

	sqlite3 "github.com/mattn/go-sqlite3"
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
	PreviewPath   string `json:"preview_path,omitempty"`
}

type Tag struct {
	Id   int    `json:"id"`
	Name string `json:"name"`
}

func GetAllTags(db *sql.DB) ([]Tag, error) {
	rows, err := db.Query("SELECT id, name FROM tags ORDER BY name ASC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.Id, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func GetImageTags(db *sql.DB, id string) ([]Tag, error) {
	rows, err := db.Query(`
		SELECT t.id, t.name FROM tags t
		JOIN image_tags it ON it.tag_id = t.id
		WHERE it.image_id = ?
		ORDER BY t.name ASC
	`, id)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tags := []Tag{}
	for rows.Next() {
		var t Tag
		if err := rows.Scan(&t.Id, &t.Name); err != nil {
			return nil, err
		}
		tags = append(tags, t)
	}
	return tags, nil
}

func CreateTag(db *sql.DB, name string) (Tag, error) {
	res, err := db.Exec("INSERT INTO tags (name) VALUES (?)", name)
	if err != nil {
		return Tag{}, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		return Tag{}, err
	}
	return Tag{Id: int(id), Name: name}, nil
}

func IsConflict(err error) bool {
	var sqliteErr sqlite3.Error
	return errors.As(err, &sqliteErr) && sqliteErr.ExtendedCode == sqlite3.ErrConstraintUnique
}

type ImageQuery struct {
	Limit, Offset int
	Sort, Order   string
	MinRating     *int // nil = no filter
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

func UpsertImagePath(db *sql.DB, filePath walk.FileInfo, thumbPath, previewPath string) (string, error) {
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
	INSERT INTO images (file_path, filename, file_size, mime_type, thumbnail_path, preview_path, capture_date, width, height, index_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(file_path) DO UPDATE SET
			file_size      = excluded.file_size,
			thumbnail_path = excluded.thumbnail_path,
			preview_path   = excluded.preview_path,
			capture_date   = excluded.capture_date,
			width          = excluded.width,
			height         = excluded.height,
			index_at       = excluded.index_at
	`, filePath.Path, filePath.FileName, filePath.Size, filePath.MimeType, thumbPath, previewPath, filePath.CaptureDate, filePath.Width, filePath.Height, time.Now().UTC())

	if err != nil {
		log.Printf("error inserting %s: %v", filePath.Path, err)
		return "", err
	}

	return action, nil
}

func GetImagesWithCount(db *sql.DB, q ImageQuery) ([]Image, int, error) {
	var images []Image
	var count int

	where := ""
	var args []any
	if q.MinRating != nil {
		where = "WHERE rating >= ?"
		args = append(args, *q.MinRating)
	}

	err := db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM images %s", where), args...).Scan(&count)
	if err != nil {
		return nil, 0, err
	}

	query := fmt.Sprintf(
		"SELECT id, file_path, filename, capture_date, width, height, rating, mime_type, thumbnail_path, COALESCE(preview_path, '') FROM images %s ORDER BY %s %s NULLS LAST LIMIT ? OFFSET ?",
		where, q.Sort, q.Order,
	)
	rows, err := db.Query(query, append(args, q.Limit, q.Offset)...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	for rows.Next() {
		var img Image
		if err := rows.Scan(&img.Id, &img.FilePath, &img.Filename, &img.CaptureDate, &img.Width, &img.Height, &img.Rating, &img.MimeType, &img.ThumbnailPath, &img.PreviewPath); err != nil {
			return nil, 0, err
		}
		images = append(images, img)
	}

	return images, count, nil
}

func GetImagePath(db *sql.DB, id string) (filePath, mimeType, previewPath string, err error) {
	err = db.QueryRow("SELECT file_path, mime_type, COALESCE(preview_path, '') FROM images WHERE id = ?", id).Scan(&filePath, &mimeType, &previewPath)
	return
}

type PatchImageInput struct {
	Rating *int  `json:"rating"`
	Tags   []int `json:"tags"`
}

func PatchImagesWithRatingOrTag(db *sql.DB, id string, input PatchImageInput) (Image, error) {
	tx, err := db.Begin()
	if err != nil {
		return Image{}, err
	}
	defer tx.Rollback()

	if input.Rating != nil {
		_, err = tx.Exec("UPDATE images SET rating = ? WHERE id = ?", *input.Rating, id)
		if err != nil {
			return Image{}, err
		}
	}

	if input.Tags != nil {
		_, err = tx.Exec("DELETE FROM image_tags WHERE image_id = ?", id)
		if err != nil {
			return Image{}, err
		}
		for _, tagID := range input.Tags {
			_, err = tx.Exec("INSERT INTO image_tags (image_id, tag_id) VALUES (?, ?)", id, tagID)
			if err != nil {
				return Image{}, err
			}
		}
	}

	if err = tx.Commit(); err != nil {
		return Image{}, err
	}

	var img Image
	err = db.QueryRow(
		"SELECT id, file_path, filename, capture_date, width, height, rating, mime_type, thumbnail_path FROM images WHERE id = ?",
		id,
	).Scan(&img.Id, &img.FilePath, &img.Filename, &img.CaptureDate, &img.Width, &img.Height, &img.Rating, &img.MimeType, &img.ThumbnailPath)

	return img, err
}
