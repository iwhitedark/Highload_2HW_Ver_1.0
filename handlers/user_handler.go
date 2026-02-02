package handlers

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/gorilla/mux"

	"go-microservice/models"
	"go-microservice/services"
	"go-microservice/utils"
)

// UserHandler handles HTTP requests for user operations
type UserHandler struct {
	userService *services.UserService
}

// NewUserHandler creates a new UserHandler
func NewUserHandler() *UserHandler {
	return &UserHandler{
		userService: services.GetUserService(),
	}
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error   string `json:"error"`
	Message string `json:"message,omitempty"`
}

// SuccessResponse represents a success response
type SuccessResponse struct {
	Message string `json:"message"`
}

// writeJSON writes JSON response
func writeJSON(w http.ResponseWriter, status int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	json.NewEncoder(w).Encode(data)
}

// writeError writes an error response
func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, ErrorResponse{Error: http.StatusText(status), Message: message})
}

// GetAllUsers handles GET /api/users
func (h *UserHandler) GetAllUsers(w http.ResponseWriter, r *http.Request) {
	users := h.userService.GetAll()

	// Async logging
	go utils.LogUserAction("LIST_USERS", 0)

	writeJSON(w, http.StatusOK, users)
}

// GetUserByID handles GET /api/users/{id}
func (h *UserHandler) GetUserByID(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		// Async error logging
		go utils.LogError("GetUserByID", err, "invalid user ID format")
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	user, err := h.userService.GetByID(id)
	if err != nil {
		// Async logging
		go utils.LogUserActionWithDetails("GET_USER_NOT_FOUND", id, err.Error())
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Async logging
	go utils.LogUserAction("GET_USER", user.ID)

	writeJSON(w, http.StatusOK, user)
}

// CreateUser handles POST /api/users
func (h *UserHandler) CreateUser(w http.ResponseWriter, r *http.Request) {
	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		// Async error logging
		go utils.LogError("CreateUser", err, "failed to decode request body")
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	// Create user (validation happens in service)
	savedUser, err := h.userService.Create(user)
	if err != nil {
		// Async error logging
		go utils.LogError("CreateUser", err, "validation failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Async logging with proper ID (after creation)
	go utils.LogUserAction("CREATE", savedUser.ID)

	// Async notification
	go utils.SendUserNotification(savedUser.ID, "WELCOME", "User account created successfully")

	writeJSON(w, http.StatusCreated, savedUser)
}

// UpdateUser handles PUT /api/users/{id}
func (h *UserHandler) UpdateUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		go utils.LogError("UpdateUser", err, "invalid user ID format")
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	var user models.User
	if err := json.NewDecoder(r.Body).Decode(&user); err != nil {
		go utils.LogError("UpdateUser", err, "failed to decode request body")
		writeError(w, http.StatusBadRequest, "Invalid request body")
		return
	}

	updatedUser, err := h.userService.Update(id, user)
	if err != nil {
		if err.Error() == "user not found" {
			go utils.LogUserActionWithDetails("UPDATE_USER_NOT_FOUND", id, err.Error())
			writeError(w, http.StatusNotFound, "User not found")
			return
		}
		go utils.LogError("UpdateUser", err, "validation failed")
		writeError(w, http.StatusBadRequest, err.Error())
		return
	}

	// Async logging
	go utils.LogUserAction("UPDATE", updatedUser.ID)

	// Async notification
	go utils.SendUserNotification(updatedUser.ID, "PROFILE_UPDATED", "Your profile has been updated")

	writeJSON(w, http.StatusOK, updatedUser)
}

// DeleteUser handles DELETE /api/users/{id}
func (h *UserHandler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	vars := mux.Vars(r)
	id, err := strconv.Atoi(vars["id"])
	if err != nil {
		go utils.LogError("DeleteUser", err, "invalid user ID format")
		writeError(w, http.StatusBadRequest, "Invalid user ID")
		return
	}

	if err := h.userService.Delete(id); err != nil {
		go utils.LogUserActionWithDetails("DELETE_USER_NOT_FOUND", id, err.Error())
		writeError(w, http.StatusNotFound, "User not found")
		return
	}

	// Async logging
	go utils.LogUserAction("DELETE", id)

	// Async notification (in production, would notify related services/users)
	go utils.SendUserNotification(id, "ACCOUNT_DELETED", "User account has been deleted")

	writeJSON(w, http.StatusOK, SuccessResponse{Message: "User deleted successfully"})
}

// RegisterRoutes registers all user routes with the router
func (h *UserHandler) RegisterRoutes(router *mux.Router) {
	router.HandleFunc("/api/users", h.GetAllUsers).Methods("GET")
	router.HandleFunc("/api/users/{id:[0-9]+}", h.GetUserByID).Methods("GET")
	router.HandleFunc("/api/users", h.CreateUser).Methods("POST")
	router.HandleFunc("/api/users/{id:[0-9]+}", h.UpdateUser).Methods("PUT")
	router.HandleFunc("/api/users/{id:[0-9]+}", h.DeleteUser).Methods("DELETE")
}
