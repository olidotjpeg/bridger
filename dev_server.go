//go:build dev

package main

import "github.com/gin-gonic/gin"

func startDevAPIServer(router *gin.Engine) {
	go router.Run(":8080")
}
