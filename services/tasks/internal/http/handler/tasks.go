package handler

import (
	"database/sql"
	"encoding/json"
	"kate/services/tasks/internal/client"
	"kate/services/tasks/internal/repository"
	"kate/services/tasks/internal/service"
	"kate/shared/middleware"
	"net/http"
	"strings"

	"go.uber.org/zap"
)

type TaskHandler struct {
	repo     *repository.SQLiteTaskRepository
	authGrpc *client.AuthGrpcClient
	logger   *zap.Logger
}

func NewTaskHandler(repo *repository.SQLiteTaskRepository, ag *client.AuthGrpcClient, logger *zap.Logger) *TaskHandler {
	return &TaskHandler{
		repo:     repo,
		authGrpc: ag,
		logger:   logger,
	}
}

func extractToken(r *http.Request) string {
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		return ""
	}
	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
		return ""
	}
	return parts[1]
}

func (h *TaskHandler) verifyToken(r *http.Request) (bool, string, int) {
	reqID := middleware.GetRequestID(r.Context())
	token := extractToken(r)
	if token == "" {
		h.logger.Info("missing token", zap.String("request_id", reqID))
		return false, "", http.StatusUnauthorized
	}

	valid, subject, err := h.authGrpc.VerifyToken(r.Context(), token)
	if err != nil {
		h.logger.Error("auth verify error",
			zap.String("request_id", reqID),
			zap.Error(err),
		)
		return false, "", http.StatusUnauthorized
	}

	if valid && subject == "" {
		subject = "unknown"
	}

	if !valid {
		h.logger.Info("invalid token", zap.String("request_id", reqID))
		return false, "", http.StatusUnauthorized
	}

	return true, subject, http.StatusOK
}

func (h *TaskHandler) handleError(w http.ResponseWriter, status int, msg string) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(map[string]string{"error": msg})
}

func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPost {
		h.handleError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	valid, _, statusCode := h.verifyToken(r)
	if !valid {
		h.handleError(w, statusCode, "unauthorized")
		return
	}

	var task service.Task
	if err := json.NewDecoder(r.Body).Decode(&task); err != nil {
		h.handleError(w, http.StatusBadRequest, "invalid json")
		return
	}

	created, err := h.repo.Create(task)
	if err != nil {
		h.logger.Error("create error", zap.String("request_id", reqID), zap.Error(err))
		h.handleError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(created)
}

func (h *TaskHandler) GetAllTasks(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodGet {
		h.handleError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	valid, _, statusCode := h.verifyToken(r)
	if !valid {
		h.handleError(w, statusCode, "unauthorized")
		return
	}

	tasks, err := h.repo.GetAll()
	if err != nil {
		h.logger.Error("get all error", zap.String("request_id", reqID), zap.Error(err))
		h.handleError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}

func (h *TaskHandler) GetTaskByID(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodGet {
		h.handleError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	valid, _, statusCode := h.verifyToken(r)
	if !valid {
		h.handleError(w, statusCode, "unauthorized")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		h.handleError(w, http.StatusBadRequest, "missing id")
		return
	}

	task, err := h.repo.GetByID(id)
	if err != nil {
		if err == sql.ErrNoRows {
			h.handleError(w, http.StatusNotFound, "task not found")
		} else {
			h.logger.Error("get by id error", zap.String("request_id", reqID), zap.Error(err))
			h.handleError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(task)
}

func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodPatch {
		h.handleError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	valid, _, statusCode := h.verifyToken(r)
	if !valid {
		h.handleError(w, statusCode, "unauthorized")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		h.handleError(w, http.StatusBadRequest, "missing id")
		return
	}

	var updates service.Task
	if err := json.NewDecoder(r.Body).Decode(&updates); err != nil {
		h.handleError(w, http.StatusBadRequest, "invalid json")
		return
	}

	updated, err := h.repo.Update(id, updates)
	if err != nil {
		if err == sql.ErrNoRows {
			h.handleError(w, http.StatusNotFound, "task not found")
		} else {
			h.logger.Error("update error", zap.String("request_id", reqID), zap.Error(err))
			h.handleError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(updated)
}

func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodDelete {
		h.handleError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	valid, _, statusCode := h.verifyToken(r)
	if !valid {
		h.handleError(w, statusCode, "unauthorized")
		return
	}

	id := strings.TrimPrefix(r.URL.Path, "/v1/tasks/")
	if id == "" {
		h.handleError(w, http.StatusBadRequest, "missing id")
		return
	}

	err := h.repo.Delete(id)
	if err != nil {
		if err == sql.ErrNoRows {
			h.handleError(w, http.StatusNotFound, "task not found")
		} else {
			h.logger.Error("delete error", zap.String("request_id", reqID), zap.Error(err))
			h.handleError(w, http.StatusInternalServerError, "internal error")
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *TaskHandler) SearchTasks(w http.ResponseWriter, r *http.Request) {
	reqID := middleware.GetRequestID(r.Context())

	if r.Method != http.MethodGet {
		h.handleError(w, http.StatusMethodNotAllowed, "method not allowed")
		return
	}

	valid, _, statusCode := h.verifyToken(r)
	if !valid {
		h.handleError(w, statusCode, "unauthorized")
		return
	}

	title := r.URL.Query().Get("title")
	if title == "" {
		h.handleError(w, http.StatusBadRequest, "title parameter is required")
		return
	}

	tasks, err := h.repo.SearchByTitle(title)
	if err != nil {
		h.logger.Error("search error",
			zap.String("request_id", reqID),
			zap.String("title", title),
			zap.Error(err),
		)
		h.handleError(w, http.StatusInternalServerError, "internal error")
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(tasks)
}
