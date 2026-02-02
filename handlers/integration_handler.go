package handlers

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/gorilla/mux"

	"go-microservice/services"
	"go-microservice/utils"
)

// IntegrationHandler handles HTTP requests for MinIO integration operations
type IntegrationHandler struct {
	integrationService *services.IntegrationService
	userService        *services.UserService
}

// NewIntegrationHandler creates a new IntegrationHandler
func NewIntegrationHandler() *IntegrationHandler {
	return &IntegrationHandler{
		integrationService: services.GetIntegrationService(),
		userService:        services.GetUserService(),
	}
}

// HealthResponse represents a health check response
type HealthResponse struct {
	Status    string `json:"status"`
	MinIO     string `json:"minio"`
	Timestamp string `json:"timestamp"`
}

// BackupResponse represents a backup operation response
type BackupResponse struct {
	Message string `json:"message"`
	UserID  int    `json:"user_id,omitempty"`
	Count   int    `json:"count,omitempty"`
}

// BackupListResponse represents a list of backups response
type BackupListResponse struct {
	Backups []string `json:"backups"`
	Count   int      `json:"count"`
}

// HealthCheck handles GET /api/health
func (h *IntegrationHandler) HealthCheck(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	minioStatus := "connected"
	if !h.integrationService.IsConnected() {
		minioStatus = "disconnected"
	} else if err := h.integrationService.HealthCheck(ctx); err != nil {
		minioStatus = "unhealthy: " + err.Error()
	}

	response := HealthResponse{
		Status:    "ok",
		MinIO:     minioStatus,
		Timestamp: time.Now().Format(time.RFC3339),
	}

	writeJSON(w, http.StatusOK, response)
}

// BackupUser handles POST /api/backup/users/{id}
func (h *IntegrationHandler) BackupUser(w http.ResponseWriter, r *http.Request) {
	if !h.integrationService.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, "MinIO service not available")
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		go utils.LogError("BackupUser", err, "invalid user ID format")
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.integrationService.BackupUser(ctx, user); err != nil {
		go utils.LogError("BackupUser", err, "failed to backup user")
		writeError(w, http.StatusInternalServerError, "Failed to backup user")
		return
	}

	// Async logging
	go utils.LogUserAction("BACKUP", user.ID)

	writeJSON(w, http.StatusOK, BackupResponse{
		Message: "User backed up successfully",
		UserID:  user.ID,
	})
}

// BackupAllUsers handles POST /api/backup/users
func (h *IntegrationHandler) BackupAllUsers(w http.ResponseWriter, r *http.Request) {
	if !h.integrationService.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, "MinIO service not available")
		return
	}

	users := h.userService.GetAll()
	if len(users) == 0 {
		writeJSON(w, http.StatusOK, BackupResponse{
			Message: "No users to backup",
			Count:   0,
		})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Minute)
	defer cancel()

	if err := h.integrationService.BackupAllUsers(ctx, users); err != nil {
		go utils.LogError("BackupAllUsers", err, "failed to backup users")
		writeError(w, http.StatusInternalServerError, "Failed to backup users: "+err.Error())
		return
	}

	// Async logging
	go utils.LogUserActionWithDetails("BACKUP_ALL", 0, "backed up all users")

	writeJSON(w, http.StatusOK, BackupResponse{
		Message: "All users backed up successfully",
		Count:   len(users),
	})
}

// RestoreUser handles POST /api/restore/users/{id}
func (h *IntegrationHandler) RestoreUser(w http.ResponseWriter, r *http.Request) {
	if !h.integrationService.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, "MinIO service not available")
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		go utils.LogError("RestoreUser", err, "invalid user ID format")
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	user, err := h.integrationService.RestoreUser(ctx, id)
	if err != nil {
		go utils.LogError("RestoreUser", err, "failed to restore user")
		writeError(w, http.StatusNotFound, "Backup not found or failed to restore")
		return
	}

	// Async logging
	go utils.LogUserAction("RESTORE", user.ID)

	writeJSON(w, http.StatusOK, user)
}

// DeleteBackup handles DELETE /api/backup/users/{id}
func (h *IntegrationHandler) DeleteBackup(w http.ResponseWriter, r *http.Request) {
	if !h.integrationService.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, "MinIO service not available")
		return
	}

	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		go utils.LogError("DeleteBackup", err, "invalid user ID format")
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	if err := h.integrationService.DeleteUserBackup(ctx, id); err != nil {
		go utils.LogError("DeleteBackup", err, "failed to delete backup")
		writeError(w, http.StatusInternalServerError, "Failed to delete backup")
		return
	}

	// Async logging
	go utils.LogUserAction("DELETE_BACKUP", id)

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "Backup deleted successfully"})
}

// ListBackups handles GET /api/backup/users
func (h *IntegrationHandler) ListBackups(w http.ResponseWriter, r *http.Request) {
	if !h.integrationService.IsConnected() {
		writeError(w, http.StatusServiceUnavailable, "MinIO service not available")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 30*time.Second)
	defer cancel()

	backups, err := h.integrationService.ListBackups(ctx)
	if err != nil {
		go utils.LogError("ListBackups", err, "failed to list backups")
		writeError(w, http.StatusInternalServerError, "Failed to list backups")
		return
	}

	writeJSON(w, http.StatusOK, BackupListResponse{
		Backups: backups,
		Count:   len(backups),
	})
}

// ConnectMinIO handles POST /api/integration/connect
func (h *IntegrationHandler) ConnectMinIO(w http.ResponseWriter, r *http.Request) {
	var config services.MinIOConfig
	if err := json.NewDecoder(r.Body).Decode(&config); err != nil {
		// Use default config if none provided
		config = services.GetDefaultConfig()
	}

	if err := h.integrationService.Connect(config); err != nil {
		go utils.LogError("ConnectMinIO", err, "failed to connect to MinIO")
		writeError(w, http.StatusInternalServerError, "Failed to connect to MinIO: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "Connected to MinIO successfully"})
}

// RegisterRoutes registers all integration routes with the router
func (h *IntegrationHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/health", h.HealthCheck).Methods("GET")
	router.HandleFunc("/api/integration/connect", h.ConnectMinIO).Methods("POST")
	router.HandleFunc("/api/backup/users", h.BackupAllUsers).Methods("POST")
	router.HandleFunc("/api/backup/users", h.ListBackups).Methods("GET")
	router.HandleFunc("/api/backup/users/{id:[0-9]+}", h.BackupUser).Methods("POST")
	router.HandleFunc("/api/backup/users/{id:[0-9]+}", h.DeleteBackup).Methods("DELETE")
	router.HandleFunc("/api/restore/users/{id:[0-9]+}", h.RestoreUser).Methods("POST")
}
