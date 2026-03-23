package main

import (
	"fmt"

	"github.com/olidotjpeg/bridger/internal/db"
)

func main() {
	fmt.Print("Hello world")

	db.RunMigrations()
}
