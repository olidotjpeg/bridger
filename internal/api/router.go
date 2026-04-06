package api

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	database "github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/scanner"
)

type PaginatedResponse[T any] struct {
	Data  []T `json:"data"`
	Total int `json:"total"`
	Page  int `json:"page"`
	Limit int `json:"limit"`
}

type Config struct {
	WalkDir  string
	ThumbDir string
}

func SetupRouter(db *sql.DB, state *scanner.ScanState, cfg Config) *gin.Engine {
	router := gin.Default()

	api := router.Group("/api")

	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	api.GET("/images", getImagesInternal(db))

	api.GET("/images/:id/full", getFullResolutionImage(db))

	api.GET("/scan/status", getScanStatus(state))
	api.POST("/scan", startNewScan(db, state, cfg))

	return router
}

func getImagesInternal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset := (page - 1) * limit

		images, count, _ := database.GetImagesWithCount(db, limit, offset)

		c.JSON(http.StatusOK, PaginatedResponse[database.Image]{
			Data:  images,
			Total: count,
			Page:  page,
			Limit: limit,
		})
	}
}

func getFullResolutionImage(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		filePath, mimeType, err := database.GetImagePath(db, id)

		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		c.Header("Content-Type", mimeType)
		c.File(filePath)
	}
}

func startNewScan(db *sql.DB, state *scanner.ScanState, cfg Config) gin.HandlerFunc {
	return func(c *gin.Context) {

		if !state.TryStart() {
			c.JSON(http.StatusConflict, gin.H{"message": "There is already an active scan running"})
			return
		}

		go func() {
			if err := scanner.RunScan(cfg.WalkDir, cfg.ThumbDir, db, state); err != nil {
				log.Printf("scan error: %v", err)
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"message": "Scan started"})
	}
}

func getScanStatus(state *scanner.ScanState) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, state.Status())
	}
}
