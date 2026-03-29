package main

import (
	"fmt"
	"log"

	"github.com/olidotjpeg/bridger/internal/db"
	walk "github.com/olidotjpeg/bridger/internal/walker"
)

func main() {
	database, err := db.Database()

	if err != nil {
		log.Fatal(err)
	}

	err = db.RunMigrations(database)

	if err != nil {
		log.Fatal(err)
	}

	results, _ := walk.WalkDirectory("./internal/walker/TestData")

	fmt.Print(len(results))
}
