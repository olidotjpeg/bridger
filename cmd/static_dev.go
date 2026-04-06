//go:build dev

package main

import "github.com/gin-gonic/gin"

// In dev mode Vite handles the frontend — nothing to serve here.
func serveStaticFiles(_ *gin.Engine) {}
