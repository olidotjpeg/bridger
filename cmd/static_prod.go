//go:build !dev

package main

import (
	"embed"
	"io/fs"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
)

//go:embed web/dist
var embeddedDist embed.FS

func serveStaticFiles(router *gin.Engine) {
	sub, err := fs.Sub(embeddedDist, "web/dist")
	if err != nil {
		panic(err)
	}

	fileServer := http.FileServer(http.FS(sub))

	router.NoRoute(func(c *gin.Context) {
		path := strings.TrimPrefix(c.Request.URL.Path, "/")
		// If the file doesn't exist in the FS, serve index.html (SPA fallback)
		if _, err := fs.Stat(sub, path); err != nil {
			c.Request.URL.Path = "/"
		}
		fileServer.ServeHTTP(c.Writer, c.Request)
	})
}
