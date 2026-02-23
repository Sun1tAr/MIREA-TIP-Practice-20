package http

import (
	"encoding/json"
	"net/http"
	"strings"

	"github.com/sirupsen/logrus"
	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/shared/middleware"
	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/tasks/internal/client/authclient"
	"github.com/sun1tar/MIREA-TIP-Practice-20/tech-ip-sem2/tasks/internal/service"
)

// TaskHandler содержит зависимости для обработки HTTP-запросов
type TaskHandler struct {
	taskService *service.TaskService
	authClient  *authclient.Client
	logger      *logrus.Logger
}

// NewTaskHandler создаёт новый экземпляр обработчика
func NewTaskHandler(ts *service.TaskService, ac *authclient.Client, logger *logrus.Logger) *TaskHandler {
	return &TaskHandler{
		taskService: ts,
		authClient:  ac,
		logger:      logger,
	}
}

// verifyToken проверяет токен через gRPC вызов к Auth service
func (h *TaskHandler) verifyToken(w http.ResponseWriter, r *http.Request) bool {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"request_id": requestID,
	})

	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		logEntry.Warn("missing authorization header")
		http.Error(w, `{"error":"missing authorization header"}`, http.StatusUnauthorized)
		return false
	}

	parts := strings.Split(authHeader, " ")
	if len(parts) != 2 || parts[0] != "Bearer" {
		logEntry.WithField("auth_header", authHeader).Warn("invalid authorization header format")
		http.Error(w, `{"error":"invalid authorization header format"}`, http.StatusUnauthorized)
		return false
	}
	token := parts[1]

	valid, _, err := h.authClient.VerifyToken(r.Context(), token)
	if err != nil {
		logEntry.WithError(err).Error("authentication service unavailable")
		http.Error(w, `{"error":"authentication service unavailable"}`, http.StatusServiceUnavailable)
		return false
	}
	if !valid {
		logEntry.WithField("token_present", token != "").Warn("invalid token")
		http.Error(w, `{"error":"invalid token"}`, http.StatusUnauthorized)
		return false
	}

	logEntry.Debug("token verified successfully")
	return true
}

// Структуры запросов
type createTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
}

type updateTaskRequest struct {
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date"`
	Done        bool   `json:"done"`
}

// Структура ответа с задачей
type taskResponse struct {
	ID          string `json:"id"`
	Title       string `json:"title"`
	Description string `json:"description"`
	DueDate     string `json:"due_date,omitempty"`
	Done        bool   `json:"done"`
}

// toTaskResponse преобразует внутреннюю модель Task в response
func toTaskResponse(t service.Task) taskResponse {
	return taskResponse{
		ID:          t.ID,
		Title:       t.Title,
		Description: t.Description,
		DueDate:     t.DueDate,
		Done:        t.Done,
	}
}

// CreateTask обрабатывает POST /v1/tasks
func (h *TaskHandler) CreateTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "CreateTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	var req createTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logEntry.WithError(err).Warn("invalid request body")
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	if req.Title == "" {
		logEntry.Warn("title is required")
		http.Error(w, `{"error":"title is required"}`, http.StatusBadRequest)
		return
	}

	task := service.Task{
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
		Done:        false,
	}
	created := h.taskService.Create(task)

	logEntry.WithField("task_id", created.ID).Info("task created successfully")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(toTaskResponse(created))
}

// ListTasks обрабатывает GET /v1/tasks
func (h *TaskHandler) ListTasks(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "ListTasks",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	tasks := h.taskService.List()
	resp := make([]taskResponse, len(tasks))
	for i, t := range tasks {
		resp[i] = toTaskResponse(t)
	}

	logEntry.WithField("count", len(tasks)).Debug("tasks listed")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(resp)
}

// GetTask обрабатывает GET /v1/tasks/{id}
func (h *TaskHandler) GetTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "GetTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	id := r.PathValue("id")
	task, ok := h.taskService.Get(id)
	if !ok {
		logEntry.WithField("task_id", id).Warn("task not found")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	logEntry.WithField("task_id", id).Debug("task retrieved")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

// UpdateTask обрабатывает PATCH /v1/tasks/{id}
func (h *TaskHandler) UpdateTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "UpdateTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	id := r.PathValue("id")
	var req updateTaskRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		logEntry.WithError(err).Warn("invalid request body")
		http.Error(w, `{"error":"invalid request body"}`, http.StatusBadRequest)
		return
	}

	updatedTask := service.Task{
		Title:       req.Title,
		Description: req.Description,
		DueDate:     req.DueDate,
		Done:        req.Done,
	}
	task, ok := h.taskService.Update(id, updatedTask)
	if !ok {
		logEntry.WithField("task_id", id).Warn("task not found for update")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	logEntry.WithField("task_id", id).Info("task updated successfully")

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(toTaskResponse(task))
}

// DeleteTask обрабатывает DELETE /v1/tasks/{id}
func (h *TaskHandler) DeleteTask(w http.ResponseWriter, r *http.Request) {
	requestID := middleware.GetRequestID(r.Context())
	logEntry := h.logger.WithFields(logrus.Fields{
		"component":  "http_handler",
		"handler":    "DeleteTask",
		"request_id": requestID,
	})

	if !h.verifyToken(w, r) {
		return
	}

	id := r.PathValue("id")
	ok := h.taskService.Delete(id)
	if !ok {
		logEntry.WithField("task_id", id).Warn("task not found for deletion")
		http.Error(w, `{"error":"task not found"}`, http.StatusNotFound)
		return
	}

	logEntry.WithField("task_id", id).Info("task deleted successfully")
	w.WriteHeader(http.StatusNoContent)
}
