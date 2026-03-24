package db

import (
	"database/sql"

	_ "github.com/mattn/go-sqlite3"
)

func Database() (*sql.DB, error) {
	db, err := sql.Open("sqlite3", "./bridger.db")

	if err != nil {
		return db, err
	}

	if err := db.Ping(); err != nil {
		return db, err
	}

	return db, err
}
