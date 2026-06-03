//go:build dev

package main

import "embed"

//go:embed sql/migrations
var migrations embed.FS

var embeddedDist embed.FS
