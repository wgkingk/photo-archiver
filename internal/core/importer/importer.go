package importer

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"time"

	"photo-archiver/internal/core/planner"
	"photo-archiver/internal/core/verifier"
	"photo-archiver/internal/media/exifmeta"
	"photo-archiver/internal/storage/sqlite"
)

var supportedExt = map[string]struct{}{
	".jpg":  {},
	".jpeg": {},
	".heic": {},
	".png":  {},
	".cr2":  {},
	".cr3":  {},
	".nef":  {},
	".arw":  {},
	".dng":  {},
	".mp4":  {},
	".mov":  {},
}

type Request struct {
	SourceRoot string
	DestRoot   string
	DryRun     bool
	VerifyMode string
}

type Result struct {
	JobID        string
	TotalCount   int
	SuccessCount int
	SkippedCount int
	FailedCount  int
	TotalBytes   int64
	CopiedBytes  int64
	Status       string
}

type sourceFile struct {
	path   string
	name   string
	size   int64
	shotAt time.Time
	mtime  time.Time
	byEXIF bool
}

type ScanResult struct {
	TotalCount    int   `json:"total_count"`
	TotalBytes    int64 `json:"total_bytes"`
	ExifCount     int   `json:"exif_count"`
	FallbackCount int   `json:"fallback_count"`
}

func NewJobID() string {
	return newID()
}

func Run(req Request, store *sqlite.Store) (Result, error) {
	return RunWithJobIDContext(context.Background(), req, store, newID())
}

func RunWithJobID(req Request, store *sqlite.Store, jobID string) (Result, error) {
	return RunWithJobIDContext(context.Background(), req, store, jobID)
}

func RunWithJobIDContext(ctx context.Context, req Request, store *sqlite.Store, jobID string) (Result, error) {
	if req.SourceRoot == "" || req.DestRoot == "" {
		return Result{}, fmt.Errorf("source and dest are required")
	}
	if req.VerifyMode == "" {
		req.VerifyMode = verifier.ModeSize
	}
	if jobID == "" {
		jobID = newID()
	}
	if err := ensureDir(req.DestRoot); err != nil {
		return Result{}, err
	}
	files, totalBytes, err := scanSource(req.SourceRoot)
	if err != nil {
		return Result{}, err
	}
	if err := store.CreateImportJob(jobID, req.SourceRoot, req.DestRoot, "", len(files), totalBytes); err != nil {
		return Result{}, err
	}
	return processFiles(ctx, files, req.DestRoot, req.DryRun, req.VerifyMode, jobID, store)
}

func RetryWithJobID(sourcePaths []string, sourceRoot, destRoot string, dryRun bool, verifyMode, jobID string, store *sqlite.Store) (Result, error) {
	return RetryWithJobIDContext(context.Background(), sourcePaths, sourceRoot, destRoot, dryRun, verifyMode, jobID, store)
}

func RetryWithJobIDContext(ctx context.Context, sourcePaths []string, sourceRoot, destRoot string, dryRun bool, verifyMode, jobID string, store *sqlite.Store) (Result, error) {
	if len(sourcePaths) == 0 {
		return Result{}, fmt.Errorf("no failed files to retry")
	}
	if destRoot == "" {
		return Result{}, fmt.Errorf("dest is required")
	}
	if verifyMode == "" {
		verifyMode = verifier.ModeSize
	}
	if jobID == "" {
		jobID = newID()
	}
	if err := ensureDir(destRoot); err != nil {
		return Result{}, err
	}
	files, totalBytes, err := sourceFilesFromPaths(sourcePaths)
	if err != nil {
		return Result{}, err
	}
	if err := store.CreateImportJob(jobID, sourceRoot, destRoot, "", len(files), totalBytes); err != nil {
		return Result{}, err
	}
	return processFiles(ctx, files, destRoot, dryRun, verifyMode, jobID, store)
}

func processFiles(ctx context.Context, files []sourceFile, destRoot string, dryRun bool, verifyMode, jobID string, store *sqlite.Store) (Result, error) {
	res := Result{JobID: jobID, TotalCount: len(files), Status: "running"}
	for _, sf := range files {
		res.TotalBytes += sf.size
	}

	reservedTargets := make(map[string]struct{}, len(files))

	for _, sf := range files {
		if ctx != nil && ctx.Err() != nil {
			res.Status = "cancelled"
			if err := store.FinishImportJob(jobID, res.Status); err != nil {
				return res, err
			}
			return res, nil
		}

		targetPath := planner.ResolveTargetPath(destRoot, sf.shotAt, sf.name, func(path string) bool {
			if _, ok := reservedTargets[path]; ok {
				return true
			}
			_, err := os.Stat(path)
			return err == nil
		})
		reservedTargets[targetPath] = struct{}{}

		srcHash, hashErr := verifier.HashFile(sf.path)
		if hashErr != nil {
			res.FailedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "failed",
				Reason:     hashErr.Error(),
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}

		exists, err := store.ExistsFileBySHA256(srcHash)
		if err != nil {
			res.FailedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "failed",
				Reason:     err.Error(),
				SHA256:     srcHash,
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}
		if exists {
			res.SkippedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "skipped",
				Reason:     "duplicate_hash",
				SHA256:     srcHash,
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}

		if dryRun {
			res.SkippedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "planned",
				Reason:     "dry_run",
				SHA256:     srcHash,
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}

		if err := copyFile(sf.path, targetPath); err != nil {
			res.FailedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "failed",
				Reason:     err.Error(),
				SHA256:     srcHash,
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}

		if err := verifier.Verify(sf.path, targetPath, verifyMode, srcHash); err != nil {
			res.FailedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "failed",
				Reason:     err.Error(),
				SHA256:     srcHash,
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}

		fileID := newID()
		recErr := store.UpsertFile(sqlite.FileRecord{
			ID:          fileID,
			SourcePath:  sf.path,
			ArchivePath: targetPath,
			FileName:    filepath.Base(targetPath),
			Ext:         strings.TrimPrefix(strings.ToLower(filepath.Ext(sf.name)), "."),
			SizeBytes:   sf.size,
			SHA256:      srcHash,
			ShotAt:      sf.shotAt,
			MTime:       sf.mtime,
		})
		if recErr != nil {
			res.FailedCount++
			_ = store.InsertImportItem(sqlite.ImportItemRecord{
				ID:         newID(),
				JobID:      jobID,
				SourcePath: sf.path,
				TargetPath: targetPath,
				Status:     "failed",
				Reason:     recErr.Error(),
				SHA256:     srcHash,
				SizeBytes:  sf.size,
			})
			_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
			continue
		}

		res.SuccessCount++
		res.CopiedBytes += sf.size
		_ = store.InsertImportItem(sqlite.ImportItemRecord{
			ID:         newID(),
			JobID:      jobID,
			FileID:     fileID,
			SourcePath: sf.path,
			TargetPath: targetPath,
			Status:     "success",
			Reason:     "",
			SHA256:     srcHash,
			SizeBytes:  sf.size,
		})
		_ = store.UpdateImportJobProgress(jobID, res.SuccessCount, res.SkippedCount, res.FailedCount, res.CopiedBytes)
	}

	res.Status = finalStatus(res)
	if ctx != nil && ctx.Err() != nil {
		res.Status = "cancelled"
	}
	if err := store.FinishImportJob(jobID, res.Status); err != nil {
		return res, err
	}
	return res, nil
}

func Scan(sourceRoot string) (ScanResult, error) {
	files, totalBytes, err := scanSource(sourceRoot)
	if err != nil {
		return ScanResult{}, err
	}
	out := ScanResult{
		TotalCount: len(files),
		TotalBytes: totalBytes,
	}
	for _, f := range files {
		if f.byEXIF {
			out.ExifCount++
			continue
		}
		out.FallbackCount++
	}
	return out, nil
}

func scanSource(sourceRoot string) ([]sourceFile, int64, error) {
	info, err := os.Stat(sourceRoot)
	if err != nil {
		return nil, 0, fmt.Errorf("stat source root: %w", err)
	}
	if !info.IsDir() {
		return nil, 0, fmt.Errorf("source root must be directory")
	}

	files := make([]sourceFile, 0, 256)
	var totalBytes int64
	err = filepath.WalkDir(sourceRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(d.Name()))
		if _, ok := supportedExt[ext]; !ok {
			return nil
		}
		fi, err := d.Info()
		if err != nil {
			return err
		}
		shotAt, byEXIF := exifmeta.ShotAt(path, fi.ModTime())
		files = append(files, sourceFile{
			path:   path,
			name:   d.Name(),
			size:   fi.Size(),
			shotAt: shotAt,
			mtime:  fi.ModTime(),
			byEXIF: byEXIF,
		})
		totalBytes += fi.Size()
		return nil
	})
	if err != nil {
		return nil, 0, fmt.Errorf("scan source: %w", err)
	}
	return files, totalBytes, nil
}

func sourceFilesFromPaths(sourcePaths []string) ([]sourceFile, int64, error) {
	files := make([]sourceFile, 0, len(sourcePaths))
	var totalBytes int64
	for _, p := range sourcePaths {
		fi, err := os.Stat(p)
		if err != nil {
			continue
		}
		if fi.IsDir() {
			continue
		}
		ext := strings.ToLower(filepath.Ext(fi.Name()))
		if _, ok := supportedExt[ext]; !ok {
			continue
		}
		shotAt, byEXIF := exifmeta.ShotAt(p, fi.ModTime())
		files = append(files, sourceFile{
			path:   p,
			name:   fi.Name(),
			size:   fi.Size(),
			shotAt: shotAt,
			mtime:  fi.ModTime(),
			byEXIF: byEXIF,
		})
		totalBytes += fi.Size()
	}
	if len(files) == 0 {
		return nil, 0, fmt.Errorf("no valid files found to process")
	}
	return files, totalBytes, nil
}

func copyFile(srcPath, dstPath string) error {
	if err := ensureDir(filepath.Dir(dstPath)); err != nil {
		return err
	}
	src, err := os.Open(srcPath)
	if err != nil {
		return fmt.Errorf("open source: %w", err)
	}
	defer src.Close()

	dst, err := os.Create(dstPath)
	if err != nil {
		return fmt.Errorf("create target: %w", err)
	}
	defer func() {
		_ = dst.Close()
	}()

	if _, err := io.Copy(dst, src); err != nil {
		return fmt.Errorf("copy file: %w", err)
	}
	return nil
}

func ensureDir(dir string) error {
	if dir == "" {
		return fmt.Errorf("dir is required")
	}
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("mkdir %s: %w", dir, err)
	}
	return nil
}

func finalStatus(res Result) string {
	if res.FailedCount == 0 {
		return "success"
	}
	if res.SuccessCount > 0 || res.SkippedCount > 0 {
		return "partial_failed"
	}
	return "failed"
}

func newID() string {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return fmt.Sprintf("fallback-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(b)
}
