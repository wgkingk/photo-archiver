package httpapi

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"photo-archiver/internal/core/importer"
	"photo-archiver/internal/core/verifier"
	"photo-archiver/internal/storage/sqlite"
)

type Server struct {
	store   *sqlite.Store
	mu      sync.Mutex
	running map[string]context.CancelFunc
}

func New(store *sqlite.Store) *Server {
	return &Server{store: store, running: make(map[string]context.CancelFunc)}
}

func (s *Server) Handler() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("/health", s.handleHealth)
	mux.HandleFunc("/v1/scan", s.handleScan)
	mux.HandleFunc("/v1/import", s.handleImport)
	mux.HandleFunc("/v1/jobs", s.handleJobs)
	mux.HandleFunc("/v1/jobs/", s.handleJobByID)
	return withJSON(mux)
}

type scanRequest struct {
	SourceRoot string `json:"source_root"`
}

type importRequest struct {
	SourceRoot string `json:"source_root"`
	DestRoot   string `json:"dest_root"`
	DryRun     bool   `json:"dry_run"`
	VerifyMode string `json:"verify_mode"`
	Async      bool   `json:"async"`
}

type retryFailedRequest struct {
	DryRun     bool   `json:"dry_run"`
	VerifyMode string `json:"verify_mode"`
	Async      bool   `json:"async"`
	Limit      int    `json:"limit"`
}

func (s *Server) handleHealth(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleScan(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req scanRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	res, err := importer.Scan(req.SourceRoot)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleImport(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	var req importRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeError(w, http.StatusBadRequest, "invalid json body")
		return
	}
	if req.VerifyMode == "" {
		req.VerifyMode = verifier.ModeSize
	}
	if req.VerifyMode != verifier.ModeSize && req.VerifyMode != verifier.ModeHash {
		writeError(w, http.StatusBadRequest, "verify_mode must be size or hash")
		return
	}
	if strings.TrimSpace(req.SourceRoot) == "" || strings.TrimSpace(req.DestRoot) == "" {
		writeError(w, http.StatusBadRequest, "source_root and dest_root are required")
		return
	}
	jobID := importer.NewJobID()
	if req.Async {
		s.runAsyncJob(jobID, func(ctx context.Context) {
			_, _ = importer.RunWithJobIDContext(ctx, importer.Request{
				SourceRoot: req.SourceRoot,
				DestRoot:   req.DestRoot,
				DryRun:     req.DryRun,
				VerifyMode: req.VerifyMode,
			}, s.store, jobID)
		})
		writeJSON(w, http.StatusAccepted, map[string]string{"job_id": jobID, "status": "accepted"})
		return
	}

	res, err := importer.RunWithJobID(importer.Request{
		SourceRoot: req.SourceRoot,
		DestRoot:   req.DestRoot,
		DryRun:     req.DryRun,
		VerifyMode: req.VerifyMode,
	}, s.store, jobID)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, res)
}

func (s *Server) handleJobs(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	limit := 30
	if v := r.URL.Query().Get("limit"); v != "" {
		n, err := strconv.Atoi(v)
		if err == nil && n > 0 {
			limit = n
		}
	}
	jobs, err := s.store.ListImportJobs(limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"items": jobs})
}

func (s *Server) handleJobByID(w http.ResponseWriter, r *http.Request) {
	path := strings.TrimPrefix(r.URL.Path, "/v1/jobs/")
	if path == "" {
		writeError(w, http.StatusNotFound, "not found")
		return
	}
	parts := strings.Split(path, "/")
	jobID := parts[0]

	if len(parts) == 2 && parts[1] == "retry-failed" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleRetryFailed(w, r, jobID)
		return
	}

	if len(parts) == 2 && parts[1] == "cancel" {
		if r.Method != http.MethodPost {
			writeError(w, http.StatusMethodNotAllowed, "method not allowed")
			return
		}
		s.handleCancelJob(w, jobID)
		return
	}

	if len(parts) == 1 && r.Method == http.MethodDelete {
		job, err := s.store.GetImportJob(jobID)
		if err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		if job.Status == "running" {
			writeError(w, http.StatusBadRequest, "cannot delete a running job")
			return
		}
		if err := s.store.DeleteImportJob(jobID); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				writeError(w, http.StatusNotFound, err.Error())
				return
			}
			writeError(w, http.StatusInternalServerError, err.Error())
			return
		}
		writeJSON(w, http.StatusOK, map[string]string{"status": "deleted", "job_id": jobID})
		return
	}

	if r.Method != http.MethodGet {
		writeError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}
	job, err := s.store.GetImportJob(jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	items, err := s.store.ListImportItemsByJob(jobID, 300)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"job": job, "items": items})
}

func (s *Server) handleRetryFailed(w http.ResponseWriter, r *http.Request, originalJobID string) {
	var req retryFailedRequest
	_ = json.NewDecoder(r.Body).Decode(&req)
	if req.VerifyMode == "" {
		req.VerifyMode = verifier.ModeSize
	}
	if req.VerifyMode != verifier.ModeSize && req.VerifyMode != verifier.ModeHash {
		writeError(w, http.StatusBadRequest, "verify_mode must be size or hash")
		return
	}
	if req.Limit <= 0 {
		req.Limit = 5000
	}

	meta, err := s.store.GetImportJobMeta(originalJobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	paths, err := s.store.ListFailedSourcePaths(originalJobID, req.Limit)
	if err != nil {
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if len(paths) == 0 {
		writeError(w, http.StatusBadRequest, "no failed items to retry")
		return
	}

	newJobID := importer.NewJobID()
	if req.Async {
		s.runAsyncJob(newJobID, func(ctx context.Context) {
			_, _ = importer.RetryWithJobIDContext(ctx, paths, meta.SourceRoot, meta.DestRoot, req.DryRun, req.VerifyMode, newJobID, s.store)
		})
		writeJSON(w, http.StatusAccepted, map[string]any{"job_id": newJobID, "source_job_id": originalJobID, "retry_count": len(paths)})
		return
	}

	res, err := importer.RetryWithJobID(paths, meta.SourceRoot, meta.DestRoot, req.DryRun, req.VerifyMode, newJobID, s.store)
	if err != nil {
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}
	writeJSON(w, http.StatusOK, map[string]any{"source_job_id": originalJobID, "retry_count": len(paths), "result": res})
}

func (s *Server) handleCancelJob(w http.ResponseWriter, jobID string) {
	if s.cancelRunningJob(jobID) {
		writeJSON(w, http.StatusAccepted, map[string]string{"status": "cancelling", "job_id": jobID})
		return
	}
	job, err := s.store.GetImportJob(jobID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			writeError(w, http.StatusNotFound, err.Error())
			return
		}
		writeError(w, http.StatusInternalServerError, err.Error())
		return
	}
	if job.Status == "running" {
		writeError(w, http.StatusConflict, "job is running but not cancellable from this process")
		return
	}
	writeError(w, http.StatusBadRequest, "job is not running")
}

func (s *Server) runAsyncJob(jobID string, fn func(ctx context.Context)) {
	ctx, cancel := context.WithCancel(context.Background())
	s.mu.Lock()
	s.running[jobID] = cancel
	s.mu.Unlock()
	go func() {
		defer func() {
			s.mu.Lock()
			delete(s.running, jobID)
			s.mu.Unlock()
		}()
		fn(ctx)
	}()
}

func (s *Server) cancelRunningJob(jobID string) bool {
	s.mu.Lock()
	cancel, ok := s.running[jobID]
	s.mu.Unlock()
	if !ok {
		return false
	}
	cancel()
	return true
}

func withJSON(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}
