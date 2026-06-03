//go:build !dev

package main

import "embed"

//go:embed all:web/dist
var embeddedDist embed.FS

//go:embed sql/migrations
var migrations embed.FS
