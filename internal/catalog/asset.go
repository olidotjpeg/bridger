package catalog

import (
	"context"
	"database/sql"
	"time"

	"github.com/google/uuid"
)

type Asset struct {
	ID         string
	Path       string
	FileSize   int64
	ModifiedAt time.Time
	Metadata   map[string]any
}

func (s *SQLiteStore) UpsertAsset(
	ctx context.Context,
	a Asset,
) (bool, error) {

	var (
		existingID       string
		existingSize     int64
		existingModified int64
	)

	err := s.db.QueryRowContext(
		ctx,
		`SELECT id, file_size, modified_at
		FROM assets
		WHERE path = ?`,
		a.Path,
	).Scan(&existingID, &existingSize, &existingModified)

	now := time.Now().Unix()

	switch {
	case err == sql.ErrNoRows:
		a.ID = uuid.NewString()

		_, err := s.db.ExecContext(
			ctx,
			`INSERT INTO assets
			(id, path, file_size, modified_at, created_at, updated_at)
			VALUES (?, ?, ?, ?, ?, ?)`,
			a.ID,
			a.Path,
			a.FileSize,
			a.ModifiedAt.Unix(),
			now,
			now,
		)
		return true, err

	case err != nil:
		return false, err

	default:
		if existingSize == a.FileSize &&
			existingModified == a.ModifiedAt.Unix() {
			return false, nil
		}

		_, err := s.db.ExecContext(
			ctx,
			`UPDATE assets
			 SET file_size = ?,
			     modified_at = ?,
			     updated_at = ?
			 WHERE path = ?`,
			a.FileSize,
			a.ModifiedAt.Unix(),
			now,
			a.Path,
		)

		return true, err
	}
}
