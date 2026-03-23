package db

import (
	"database/sql"

	_ "modernc.org/sqlite"
)

func db() error {
	db, err := sql.Open("sqlite", "./bridger.db")

	if err != nil {
		return err
	}
}
