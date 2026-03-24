package main

import (
	"log"

	"github.com/olidotjpeg/bridger/internal/db"
)

func main() {
	database, err := db.Database()

	if err != nil {
		log.Fatal(err)
	}

	err = db.RunMigrations(database)

	log.Fatal(err)
}
