package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/olidotjpeg/bridger/internal/config"
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
	ThumbDir   string
	NeedsSetup bool
	CurrentCfg *config.Config
	ReconfigCh chan<- config.Config
}

func SetupRouter(db *sql.DB, state *scanner.ScanState, cfg Config) *gin.Engine {
	router := gin.Default()

	api := router.Group("/api")

	api.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"message": "pong",
		})
	})

	api.GET("/images", getImagesInternal(db, cfg))
	api.PATCH("/images/:id", patchImagesWithRatingOrTag(db, cfg))
	api.GET("/images/:id/full", getFullResolutionImage(db))
	api.GET("/images/:id/tags", getImageTags(db))

	api.GET("/tags", getAllTags(db))
	api.POST("/tags", postNewTag(db))

	api.GET("/scan/status", getScanStatus(state))
	api.POST("/scan", startNewScan(db, state, cfg))

	api.GET("/config", getConfig(&cfg))
	api.PUT("/config", putConfig(&cfg))
	api.GET("/fs/list", listDirectory)

	return router
}

func thumbURL(absPath, thumbDir string) string {
	rel, err := filepath.Rel(thumbDir, absPath)
	if err != nil || strings.HasPrefix(rel, "..") {
		return absPath
	}
	return "/thumbs/" + rel
}

func getImagesInternal(db *sql.DB, cfg Config) gin.HandlerFunc {
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

		for i := range images {
			if images[i].ThumbnailPath != "" {
				images[i].ThumbnailPath = thumbURL(images[i].ThumbnailPath, cfg.ThumbDir)
			}
		}

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

		// Prevent stale caches: the file served for a given id can change after a rescan.
		c.Header("Cache-Control", "no-cache")

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

func patchImagesWithRatingOrTag(db *sql.DB, cfg Config) gin.HandlerFunc {
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

		if img.ThumbnailPath != "" {
			img.ThumbnailPath = thumbURL(img.ThumbnailPath, cfg.ThumbDir)
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
			if err := scanner.RunScan(cfg.CurrentCfg.ScanDirs, cfg.ThumbDir, db, state); err != nil {
				log.Printf("scan error: %v", err)
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"message": "Scan started"})
	}
}

func getConfig(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": cfg.NeedsSetup,
			"scan_dirs":   cfg.CurrentCfg.ScanDirs,
			"db_path":     cfg.CurrentCfg.DBPath,
			"thumbs_path": cfg.CurrentCfg.ThumbsPath,
		})
	}
}

func putConfig(cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			ScanDirs []string `json:"scan_dirs" binding:"required"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || len(body.ScanDirs) == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"message": "scan_dirs is required and must not be empty"})
			return
		}

		for _, d := range body.ScanDirs {
			info, err := os.Stat(d)
			if err != nil || !info.IsDir() {
				c.JSON(http.StatusBadRequest, gin.H{"message": "directory does not exist: " + d})
				return
			}
		}

		cfg.CurrentCfg.ScanDirs = body.ScanDirs
		if err := config.Save(cfg.CurrentCfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save config"})
			return
		}

		cfg.NeedsSetup = false
		cfg.ReconfigCh <- *cfg.CurrentCfg

		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}

type dirEntry struct {
	Name string `json:"name"`
	Path string `json:"path"`
}

func listDirectory(c *gin.Context) {
	reqPath := c.Query("path")
	if reqPath == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "could not determine home directory"})
			return
		}
		reqPath = home
	}

	// Resolve to absolute path and clean it
	abs, err := filepath.Abs(reqPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"message": "invalid path"})
		return
	}

	info, err := os.Stat(abs)
	if err != nil || !info.IsDir() {
		c.JSON(http.StatusBadRequest, gin.H{"message": "path is not a directory"})
		return
	}

	entries, err := os.ReadDir(abs)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"message": "could not read directory"})
		return
	}

	var dirs []dirEntry
	for _, e := range entries {
		if !e.IsDir() {
			continue
		}
		if strings.HasPrefix(e.Name(), ".") {
			continue
		}
		dirs = append(dirs, dirEntry{
			Name: e.Name(),
			Path: filepath.Join(abs, e.Name()),
		})
	}
	sort.Slice(dirs, func(i, j int) bool {
		return strings.ToLower(dirs[i].Name) < strings.ToLower(dirs[j].Name)
	})

	parent := filepath.Dir(abs)
	if parent == abs {
		parent = "" // at filesystem root
	}

	c.JSON(http.StatusOK, gin.H{
		"path":    abs,
		"parent":  parent,
		"entries": dirs,
	})
}

func getScanStatus(state *scanner.ScanState) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, state.Status())
	}
}
