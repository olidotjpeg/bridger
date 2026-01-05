package indexer

import (
	"context"
	"io/fs"
	"path/filepath"
	"strings"
	"sync"

	"github.com/google/uuid"
	"github.com/olidotjpeg/bridge-clone/internal/catalog"
)

type AssetStore interface {
	UpsertAsset(ctx context.Context, a catalog.Asset) (bool, error)
}

type MetadataReader interface {
	Read(ctx context.Context, path string) (map[string]any, error)
}

type Thumbnailer interface {
	Enqueue(assetID, path string)
	Dequeue() (int, error)
	Front() (int, error)
	IsEmpty() bool
	Size() int
}

type Indexer struct {
	Workers      int
	JobQueueSize int

	db        AssetStore
	meta      MetadataReader
	thumbs    Thumbnailer
	progressC chan Progress
}

var supportedExt = map[string]bool{
	".jpg":  true,
	".jpeg": true,
	".png":  true,
	".tif":  true,
	".tiff": true,
	".heic": true,
	".raw":  true,
	".nef":  true,
	".cr2":  true,
}

func IsSupported(path string) bool {
	ext := filepath.Ext(path)
	return supportedExt[strings.ToLower(ext)]
}

func (i *Indexer) walk(
	ctx context.Context,
	root string,
	jobs chan<- Job,
) error {

	return filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return nil
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if d.IsDir() {
			if strings.HasPrefix(d.Name(), ".") {
				return filepath.SkipDir
			}
			return nil
		}

		if !IsSupported(path) {
			return nil
		}

		info, err := d.Info()
		if err != nil {
			return nil
		}

		jobs <- Job{
			Path: path,
			Info: FileInfo{
				Size:       info.Size(),
				ModifiedAt: info.ModTime(),
			},
		}

		i.progressC <- Progress{
			Scanned: 1,
			Path:    path,
		}

		return nil
	})
}

func (i *Indexer) worker(
	ctx context.Context,
	wg *sync.WaitGroup,
	jobs <-chan Job,
) {
	defer wg.Done()

	for job := range jobs {
		asset := catalog.Asset{
			ID:         uuid.NewString(),
			Path:       job.Path,
			FileSize:   job.Info.Size,
			ModifiedAt: job.Info.ModifiedAt,
		}

		changed, err := i.db.UpsertAsset(ctx, asset)
		if err != nil {
			i.progressC <- Progress{Errors: 1, Path: job.Path}
			continue
		}

		// Already indexed
		if !changed {
			continue
		}

		meta, err := i.meta.Read(ctx, job.Path)
		if err == nil {
			asset.Metadata = meta
		}

		i.thumbs.Enqueue(asset.ID, job.Path)

		i.progressC <- Progress{
			Indexed: 1,
			Path:    job.Path,
		}
	}
}

func (i *Indexer) Run(ctx context.Context, roots []string) error {
	jobs := make(chan Job, i.JobQueueSize)
	i.progressC = make(chan Progress, 100)

	var wg sync.WaitGroup

	for w := 0; w < i.Workers; w++ {
		wg.Add(1)
		go i.worker(ctx, &wg, jobs)
	}

	for _, root := range roots {
		if err := i.walk(ctx, root, jobs); err != nil {
			close(jobs)
			wg.Wait()
			return err
		}
	}

	close(jobs)
	wg.Wait()
	close(i.progressC)
	return nil
}

type ExifReader struct{}

func (e *ExifReader) Read(ctx context.Context, path string) (map[string]any, error) {
	// For now just return minimal or empty EXIF data
	return map[string]any{}, nil
}

func (i *Indexer) Progress() <-chan Progress {
	return i.progressC
}

func (i *Indexer) SetStore(s AssetStore) {
	i.db = s
}

func (i *Indexer) SetMetadata(m MetadataReader) {
	i.meta = m
}

func (i *Indexer) SetThumbnailer(t Thumbnailer) {
	i.thumbs = t
}
