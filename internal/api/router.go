package api

import (
	"database/sql"
	"log"
	"net/http"
	"os"
	"path/filepath"
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
	CurrentCfg *config.Config
	ReconfigCh chan<- config.Config
	SaveConfig func(*config.Config) error // defaults to config.Save if nil
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

	api.GET("/dates", getDates(db))

	api.GET("/scan/status", getScanStatus(state))
	api.POST("/scan", startNewScan(db, state, cfg))

	api.GET("/config", getConfig(db, &cfg))
	api.PUT("/config", putConfig(db, &cfg))

	api.GET("/projects", getProjects(db))
	api.POST("/projects", postProject(db, &cfg))
	api.PUT("/projects/:id", putProject(db, &cfg))
	api.DELETE("/projects/:id", deleteProject(db, &cfg))

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

		var dateFrom, dateTo *string
		if f := c.Query("from"); f != "" {
			dateFrom = &f
		}
		if t := c.Query("to"); t != "" {
			dateTo = &t
		}

		var projectID *int
		if p, err := strconv.Atoi(c.Query("project_id")); err == nil {
			projectID = &p
		}

		q := database.ImageQuery{
			Limit:     limit,
			Offset:    offset,
			Sort:      sort,
			Order:     order,
			MinRating: minRating,
			DateFrom:  dateFrom,
			DateTo:    dateTo,
			ProjectID: projectID,
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

func getDates(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		groups, err := database.GetDateGroups(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, groups)
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
		if err == sql.ErrNoRows {
			c.JSON(http.StatusNotFound, gin.H{"message": "image not found"})
			return
		}
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
			if err := scanner.RunScan(cfg.ThumbDir, db, state); err != nil {
				log.Printf("scan error: %v", err)
			}
		}()
		c.JSON(http.StatusAccepted, gin.H{"message": "Scan started"})
	}
}

func getConfig(db *sql.DB, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		projects, _ := database.GetAllProjects(db)
		needsSetup := len(projects) == 0 && len(cfg.CurrentCfg.ScanDirs) == 0
		c.JSON(http.StatusOK, gin.H{
			"needs_setup": needsSetup,
			"db_path":     cfg.CurrentCfg.DBPath,
			"thumbs_path": cfg.CurrentCfg.ThumbsPath,
		})
	}
}

// putConfig is used by the setup wizard to provide initial scan directories.
// It creates one project per directory (named after the folder basename), then
// clears scan_dirs from config and triggers the first scan.
func putConfig(db *sql.DB, cfg *Config) gin.HandlerFunc {
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

		for _, d := range body.ScanDirs {
			name := filepath.Base(d)
			proj, err := database.CreateProject(db, name)
			if database.IsConflict(err) {
				// project with this name already exists — find it and reuse
				projects, _ := database.GetAllProjects(db)
				for _, p := range projects {
					if p.Name == name {
						proj = p
						break
					}
				}
			} else if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to create project: " + err.Error()})
				return
			}
			if err := database.AddDirToProject(db, proj.Id, d); err != nil && !database.IsConflict(err) {
				c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to add dir to project: " + err.Error()})
				return
			}
		}

		cfg.CurrentCfg.ScanDirs = nil
		saveFn := cfg.SaveConfig
		if saveFn == nil {
			saveFn = config.Save
		}
		if err := saveFn(cfg.CurrentCfg); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": "failed to save config"})
			return
		}

		cfg.ReconfigCh <- *cfg.CurrentCfg

		c.JSON(http.StatusOK, gin.H{"message": "ok"})
	}
}


func getScanStatus(state *scanner.ScanState) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.JSON(http.StatusOK, state.Status())
	}
}

func getProjects(db *sql.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		projects, err := database.GetAllProjects(db)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}
		c.JSON(http.StatusOK, projects)
	}
}

func postProject(db *sql.DB, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		var body struct {
			Name string   `json:"name" binding:"required"`
			Dirs []string `json:"dirs"`
		}
		if err := c.ShouldBindJSON(&body); err != nil || body.Name == "" {
			c.JSON(http.StatusBadRequest, gin.H{"message": "name is required"})
			return
		}

		proj, err := database.CreateProject(db, body.Name)
		if err != nil {
			if database.IsConflict(err) {
				c.JSON(http.StatusConflict, gin.H{"message": "project name already exists"})
				return
			}
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		for _, d := range body.Dirs {
			info, err := os.Stat(d)
			if err != nil || !info.IsDir() {
				c.JSON(http.StatusBadRequest, gin.H{"message": "directory does not exist: " + d})
				return
			}
			if err := database.AddDirToProject(db, proj.Id, d); err != nil && !database.IsConflict(err) {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}

		cfg.ReconfigCh <- *cfg.CurrentCfg

		projects, _ := database.GetAllProjects(db)
		for _, p := range projects {
			if p.Id == proj.Id {
				c.JSON(http.StatusCreated, p)
				return
			}
		}
		c.JSON(http.StatusCreated, proj)
	}
}

func putProject(db *sql.DB, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
			return
		}

		var body struct {
			Name *string  `json:"name"`
			Dirs []string `json:"dirs"`
		}
		if err := c.ShouldBindJSON(&body); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid request body"})
			return
		}

		if body.Name != nil && *body.Name != "" {
			if err := database.UpdateProjectName(db, id, *body.Name); err != nil {
				if database.IsConflict(err) {
					c.JSON(http.StatusConflict, gin.H{"message": "project name already exists"})
					return
				}
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
		}

		if body.Dirs != nil {
			for _, d := range body.Dirs {
				info, err := os.Stat(d)
				if err != nil || !info.IsDir() {
					c.JSON(http.StatusBadRequest, gin.H{"message": "directory does not exist: " + d})
					return
				}
			}

			// Remove all current dirs for this project, then re-add the provided set.
			if _, err := db.Exec("DELETE FROM project_dirs WHERE project_id = ?", id); err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
				return
			}
			for _, d := range body.Dirs {
				if err := database.AddDirToProject(db, id, d); err != nil && !database.IsConflict(err) {
					c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
					return
				}
			}

			cfg.ReconfigCh <- *cfg.CurrentCfg
		}

		projects, _ := database.GetAllProjects(db)
		for _, p := range projects {
			if p.Id == id {
				c.JSON(http.StatusOK, p)
				return
			}
		}
		c.JSON(http.StatusNotFound, gin.H{"message": "project not found"})
	}
}

func deleteProject(db *sql.DB, cfg *Config) gin.HandlerFunc {
	return func(c *gin.Context) {
		id, err := strconv.Atoi(c.Param("id"))
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"message": "invalid id"})
			return
		}

		if err := database.DeleteProject(db, id); err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"message": err.Error()})
			return
		}

		cfg.ReconfigCh <- *cfg.CurrentCfg

		c.Status(http.StatusNoContent)
	}
}
