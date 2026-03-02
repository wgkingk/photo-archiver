package sqlite

import (
	"database/sql"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"time"

	_ "modernc.org/sqlite"
)

type Store struct {
	DB *sql.DB
}

type ImportItemRecord struct {
	ID         string
	JobID      string
	FileID     string
	SourcePath string
	TargetPath string
	Status     string
	Reason     string
	SHA256     string
	SizeBytes  int64
}

type FileRecord struct {
	ID          string
	SourcePath  string
	ArchivePath string
	FileName    string
	Ext         string
	SizeBytes   int64
	SHA256      string
	ShotAt      time.Time
	MTime       time.Time
}

type ImportJobSummary struct {
	ID           string
	Status       string
	TotalCount   int
	SuccessCount int
	SkippedCount int
	FailedCount  int
	TotalBytes   int64
	CopiedBytes  int64
	StartedAt    string
	FinishedAt   string
}

type ImportItemSummary struct {
	ID         string
	SourcePath string
	TargetPath string
	Status     string
	Reason     string
	SizeBytes  int64
}

type ImportJobMeta struct {
	ID         string
	SourceRoot string
	DestRoot   string
}

func Open(dbPath string) (*Store, error) {
	if dbPath == "" {
		return nil, errors.New("db path is required")
	}
	if err := os.MkdirAll(filepath.Dir(dbPath), 0o755); err != nil {
		return nil, fmt.Errorf("create db dir: %w", err)
	}
	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		return nil, fmt.Errorf("open sqlite: %w", err)
	}
	if err := db.Ping(); err != nil {
		_ = db.Close()
		return nil, fmt.Errorf("ping sqlite: %w", err)
	}
	return &Store{DB: db}, nil
}

func (s *Store) Close() error {
	if s == nil || s.DB == nil {
		return nil
	}
	return s.DB.Close()
}

func (s *Store) ApplySchema(schemaPath string) error {
	sqlBytes, err := os.ReadFile(schemaPath)
	if err != nil {
		return fmt.Errorf("read schema: %w", err)
	}
	if _, err := s.DB.Exec(string(sqlBytes)); err != nil {
		return fmt.Errorf("apply schema: %w", err)
	}
	return nil
}

func (s *Store) CreateImportJob(id, sourceRoot, destRoot, backup2Root string, totalCount int, totalBytes int64) error {
	_, err := s.DB.Exec(
		`INSERT INTO import_jobs (
			id, source_root, dest_root, backup2_root, status,
			total_count, total_bytes, started_at
		) VALUES (?, ?, ?, ?, 'running', ?, ?, CURRENT_TIMESTAMP)`,
		id, sourceRoot, destRoot, backup2Root, totalCount, totalBytes,
	)
	if err != nil {
		return fmt.Errorf("insert import job: %w", err)
	}
	return nil
}

func (s *Store) UpdateImportJobProgress(id string, successCount, skippedCount, failedCount int, copiedBytes int64) error {
	_, err := s.DB.Exec(
		`UPDATE import_jobs
		 SET success_count = ?, skipped_count = ?, failed_count = ?, copied_bytes = ?, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		successCount, skippedCount, failedCount, copiedBytes, id,
	)
	if err != nil {
		return fmt.Errorf("update import job progress: %w", err)
	}
	return nil
}

func (s *Store) FinishImportJob(id, status string) error {
	_, err := s.DB.Exec(
		`UPDATE import_jobs
		 SET status = ?, finished_at = CURRENT_TIMESTAMP, updated_at = CURRENT_TIMESTAMP
		 WHERE id = ?`,
		status, id,
	)
	if err != nil {
		return fmt.Errorf("finish import job: %w", err)
	}
	return nil
}

func (s *Store) InsertImportItem(item ImportItemRecord) error {
	_, err := s.DB.Exec(
		`INSERT INTO import_items
		(id, job_id, file_id, source_path, target_path, status, reason, sha256, size_bytes)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		item.ID, item.JobID, nullable(item.FileID), item.SourcePath, item.TargetPath,
		item.Status, item.Reason, item.SHA256, item.SizeBytes,
	)
	if err != nil {
		return fmt.Errorf("insert import item: %w", err)
	}
	return nil
}

func (s *Store) UpsertFile(file FileRecord) error {
	_, err := s.DB.Exec(
		`INSERT INTO files (
			id, source_path, archive_path, file_name, ext,
			size_bytes, sha256, shot_at, mtime
		) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)
		ON CONFLICT(archive_path) DO UPDATE SET
			source_path=excluded.source_path,
			size_bytes=excluded.size_bytes,
			sha256=excluded.sha256,
			shot_at=excluded.shot_at,
			mtime=excluded.mtime,
			updated_at=CURRENT_TIMESTAMP`,
		file.ID, file.SourcePath, file.ArchivePath, file.FileName, file.Ext,
		file.SizeBytes, file.SHA256, toNullTime(file.ShotAt), toNullTime(file.MTime),
	)
	if err != nil {
		return fmt.Errorf("upsert file: %w", err)
	}
	return nil
}

func (s *Store) ExistsFileBySHA256(sha256 string) (bool, error) {
	if sha256 == "" {
		return false, nil
	}
	row := s.DB.QueryRow(`SELECT EXISTS (SELECT 1 FROM files WHERE sha256 = ? LIMIT 1)`, sha256)
	var exists bool
	if err := row.Scan(&exists); err != nil {
		return false, fmt.Errorf("query file by sha256: %w", err)
	}
	return exists, nil
}

func (s *Store) ListImportJobs(limit int) ([]ImportJobSummary, error) {
	if limit <= 0 {
		limit = 20
	}
	rows, err := s.DB.Query(
		`SELECT id, status, total_count, success_count, skipped_count, failed_count,
		        total_bytes, copied_bytes,
		        COALESCE(started_at, ''), COALESCE(finished_at, '')
		 FROM import_jobs
		 ORDER BY created_at DESC
		 LIMIT ?`,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list import jobs: %w", err)
	}
	defer rows.Close()

	out := make([]ImportJobSummary, 0, limit)
	for rows.Next() {
		var item ImportJobSummary
		if err := rows.Scan(
			&item.ID,
			&item.Status,
			&item.TotalCount,
			&item.SuccessCount,
			&item.SkippedCount,
			&item.FailedCount,
			&item.TotalBytes,
			&item.CopiedBytes,
			&item.StartedAt,
			&item.FinishedAt,
		); err != nil {
			return nil, fmt.Errorf("scan import job row: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import jobs: %w", err)
	}
	return out, nil
}

func (s *Store) GetImportJob(id string) (ImportJobSummary, error) {
	row := s.DB.QueryRow(
		`SELECT id, status, total_count, success_count, skipped_count, failed_count,
		        total_bytes, copied_bytes,
		        COALESCE(started_at, ''), COALESCE(finished_at, '')
		 FROM import_jobs
		 WHERE id = ?`,
		id,
	)
	var item ImportJobSummary
	if err := row.Scan(
		&item.ID,
		&item.Status,
		&item.TotalCount,
		&item.SuccessCount,
		&item.SkippedCount,
		&item.FailedCount,
		&item.TotalBytes,
		&item.CopiedBytes,
		&item.StartedAt,
		&item.FinishedAt,
	); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ImportJobSummary{}, fmt.Errorf("get import job: %w", sql.ErrNoRows)
		}
		return ImportJobSummary{}, fmt.Errorf("get import job: %w", err)
	}
	return item, nil
}

func (s *Store) ListImportItemsByJob(jobID string, limit int) ([]ImportItemSummary, error) {
	if limit <= 0 {
		limit = 200
	}
	rows, err := s.DB.Query(
		`SELECT id, source_path, target_path, status, reason, size_bytes
		 FROM import_items
		 WHERE job_id = ?
		 ORDER BY created_at DESC
		 LIMIT ?`,
		jobID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list import items by job: %w", err)
	}
	defer rows.Close()
	out := make([]ImportItemSummary, 0, limit)
	for rows.Next() {
		var item ImportItemSummary
		if err := rows.Scan(&item.ID, &item.SourcePath, &item.TargetPath, &item.Status, &item.Reason, &item.SizeBytes); err != nil {
			return nil, fmt.Errorf("scan import item row: %w", err)
		}
		out = append(out, item)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate import items: %w", err)
	}
	return out, nil
}

func (s *Store) DeleteImportJob(id string) error {
	res, err := s.DB.Exec(`DELETE FROM import_jobs WHERE id = ?`, id)
	if err != nil {
		return fmt.Errorf("delete import job: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("delete import job rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("delete import job: %w", sql.ErrNoRows)
	}
	return nil
}

func (s *Store) ListFailedSourcePaths(jobID string, limit int) ([]string, error) {
	if limit <= 0 {
		limit = 5000
	}
	rows, err := s.DB.Query(
		`SELECT DISTINCT source_path
		 FROM import_items
		 WHERE job_id = ? AND status = 'failed'
		 ORDER BY created_at ASC
		 LIMIT ?`,
		jobID, limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list failed source paths: %w", err)
	}
	defer rows.Close()
	paths := make([]string, 0, limit)
	for rows.Next() {
		var p string
		if err := rows.Scan(&p); err != nil {
			return nil, fmt.Errorf("scan failed source path: %w", err)
		}
		paths = append(paths, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate failed source paths: %w", err)
	}
	return paths, nil
}

func (s *Store) GetImportJobMeta(id string) (ImportJobMeta, error) {
	row := s.DB.QueryRow(`SELECT id, source_root, dest_root FROM import_jobs WHERE id = ?`, id)
	var meta ImportJobMeta
	if err := row.Scan(&meta.ID, &meta.SourceRoot, &meta.DestRoot); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return ImportJobMeta{}, fmt.Errorf("get import job meta: %w", sql.ErrNoRows)
		}
		return ImportJobMeta{}, fmt.Errorf("get import job meta: %w", err)
	}
	return meta, nil
}

func nullable(v string) any {
	if v == "" {
		return nil
	}
	return v
}

func toNullTime(t time.Time) any {
	if t.IsZero() {
		return nil
	}
	return t.UTC()
}
