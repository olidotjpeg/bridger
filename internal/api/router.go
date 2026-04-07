package api

import (
	"database/sql"
	"log"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	database "github.com/olidotjpeg/bridger/internal/db"
	"github.com/olidotjpeg/bridger/internal/raw"
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
	api.PATCH("/images/:id", patchImagesWithRatingOrTag(db))
	api.GET("/images/:id/full", getFullResolutionImage(db))
	api.GET("/images/:id/tags", getImageTags(db))

	api.GET("/tags", getAllTags(db))
	api.POST("/tags", postNewTag(db))

	api.GET("/scan/status", getScanStatus(state))
	api.POST("/scan", startNewScan(db, state, cfg))

	return router
}

func getImagesInternal(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
		limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))
		offset := (page - 1) * limit

		validSort := map[string]bool{
			"capture_date": true,
			"rating":       true,
			"filename":     true,
		}

		sort := c.DefaultQuery("sort", "capture_date")
		if !validSort[sort] {
			sort = "capture_date"
		}

		order := c.DefaultQuery("order", "desc")
		if order != "asc" && order != "desc" {
			order = "desc"
		}

		var minRating *int
		if r, err := strconv.Atoi(c.Query("rating")); err == nil {
			minRating = &r
		}

		q := database.ImageQuery{
			Limit:     limit,
			Offset:    offset,
			Sort:      sort,
			Order:     order,
			MinRating: minRating,
		}

		images, count, _ := database.GetImagesWithCount(db, q)

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

		filePath, mimeType, previewPath, err := database.GetImagePath(db, id)
		if err != nil {
			c.Status(http.StatusNotFound)
			return
		}

		if raw.IsRaw(mimeType) {
			if previewPath != "" {
				c.Header("Content-Type", "image/jpeg")
				c.File(previewPath)
				return
			}
			c.Status(http.StatusNotFound)
			return
		}

		c.Header("Content-Type", mimeType)
		c.File(filePath)
	}
}

func getAllTags(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		tags, err := database.GetAllTags(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tags)
	}
}

func getImageTags(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")
		tags, err := database.GetImageTags(db, id)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, tags)
	}
}

func postNewTag(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name string `json:"name"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "name is required"})
			return
		}

		tag, err := database.CreateTag(db, body.Name)
		if err != nil {
			if database.IsConflict(err) {
				c.JSON(http.StatusConflict, gin.H{"message": "tag already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusCreated, tag)
	}
}

func patchImagesWithRatingOrTag(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		id := c.Param("id")

		var input database.PatchImageInput
		if err := c.ShouldBindJSON(&input); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		img, err := database.PatchImagesWithRatingOrTag(db, id, input)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		c.JSON(http.StatusOK, img)
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
