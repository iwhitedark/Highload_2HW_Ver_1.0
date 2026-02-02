package utils

import (
	"fmt"
	"log"
	"os"
	"sync"
	"time"
)

// AuditLogger handles asynchronous logging of user operations
type AuditLogger struct {
	logChan chan LogEntry
	wg      sync.WaitGroup
	logger  *log.Logger
}

// LogEntry represents a single audit log entry
type LogEntry struct {
	Action    string
	UserID    int
	Details   string
	Timestamp time.Time
}

var (
	auditLogger *AuditLogger
	once        sync.Once
)

// GetAuditLogger returns a singleton instance of AuditLogger
func GetAuditLogger() *AuditLogger {
	once.Do(func() {
		auditLogger = &AuditLogger{
			logChan: make(chan LogEntry, 10000), // Buffered channel for high throughput
			logger:  log.New(os.Stdout, "[AUDIT] ", log.LstdFlags),
		}
		auditLogger.start()
	})
	return auditLogger
}

// start begins processing log entries asynchronously
func (a *AuditLogger) start() {
	go func() {
		for entry := range a.logChan {
			a.logger.Printf("Action: %s | UserID: %d | Details: %s | Time: %s",
				entry.Action,
				entry.UserID,
				entry.Details,
				entry.Timestamp.Format(time.RFC3339))
			a.wg.Done()
		}
	}()
}

// LogUserAction logs a user action asynchronously
func (a *AuditLogger) LogUserAction(action string, userID int, details string) {
	a.wg.Add(1)
	select {
	case a.logChan <- LogEntry{
		Action:    action,
		UserID:    userID,
		Details:   details,
		Timestamp: time.Now(),
	}:
	default:
		// Channel full, log synchronously as fallback
		a.wg.Done()
		a.logger.Printf("[OVERFLOW] Action: %s | UserID: %d | Details: %s",
			action, userID, details)
	}
}

// Close gracefully shuts down the logger
func (a *AuditLogger) Close() {
	a.wg.Wait()
	close(a.logChan)
}

// LogUserAction is a convenience function for logging user actions
func LogUserAction(action string, userID int) {
	GetAuditLogger().LogUserAction(action, userID, "")
}

// LogUserActionWithDetails logs a user action with additional details
func LogUserActionWithDetails(action string, userID int, details string) {
	GetAuditLogger().LogUserAction(action, userID, details)
}

// NotificationService handles async notifications
type NotificationService struct {
	notifyChan chan Notification
	wg         sync.WaitGroup
}

// Notification represents a notification to be sent
type Notification struct {
	UserID  int
	Type    string
	Message string
}

var (
	notificationService *NotificationService
	notifyOnce          sync.Once
)

// GetNotificationService returns a singleton instance of NotificationService
func GetNotificationService() *NotificationService {
	notifyOnce.Do(func() {
		notificationService = &NotificationService{
			notifyChan: make(chan Notification, 10000),
		}
		notificationService.start()
	})
	return notificationService
}

// start begins processing notifications asynchronously
func (n *NotificationService) start() {
	go func() {
		for notif := range n.notifyChan {
			// Simulate sending notification (in production, this would send email/push/etc.)
			log.Printf("[NOTIFICATION] Type: %s | UserID: %d | Message: %s",
				notif.Type, notif.UserID, notif.Message)
			n.wg.Done()
		}
	}()
}

// SendNotification sends a notification asynchronously
func (n *NotificationService) SendNotification(userID int, notifType, message string) {
	n.wg.Add(1)
	select {
	case n.notifyChan <- Notification{
		UserID:  userID,
		Type:    notifType,
		Message: message,
	}:
	default:
		n.wg.Done()
		log.Printf("[NOTIFICATION OVERFLOW] Type: %s | UserID: %d", notifType, userID)
	}
}

// SendUserNotification is a convenience function for sending user notifications
func SendUserNotification(userID int, notifType, message string) {
	GetNotificationService().SendNotification(userID, notifType, message)
}

// ErrorHandler handles async error processing
type ErrorHandler struct {
	errorChan chan ErrorEntry
	wg        sync.WaitGroup
	logger    *log.Logger
}

// ErrorEntry represents an error to be logged
type ErrorEntry struct {
	Operation string
	Error     error
	Context   string
	Timestamp time.Time
}

var (
	errorHandler *ErrorHandler
	errorOnce    sync.Once
)

// GetErrorHandler returns a singleton instance of ErrorHandler
func GetErrorHandler() *ErrorHandler {
	errorOnce.Do(func() {
		errorHandler = &ErrorHandler{
			errorChan: make(chan ErrorEntry, 10000),
			logger:    log.New(os.Stderr, "[ERROR] ", log.LstdFlags),
		}
		errorHandler.start()
	})
	return errorHandler
}

// start begins processing errors asynchronously
func (e *ErrorHandler) start() {
	go func() {
		for entry := range e.errorChan {
			e.logger.Printf("Operation: %s | Error: %v | Context: %s | Time: %s",
				entry.Operation,
				entry.Error,
				entry.Context,
				entry.Timestamp.Format(time.RFC3339))
			e.wg.Done()
		}
	}()
}

// HandleError logs an error asynchronously
func (e *ErrorHandler) HandleError(operation string, err error, context string) {
	e.wg.Add(1)
	select {
	case e.errorChan <- ErrorEntry{
		Operation: operation,
		Error:     err,
		Context:   context,
		Timestamp: time.Now(),
	}:
	default:
		e.wg.Done()
		e.logger.Printf("[OVERFLOW] Operation: %s | Error: %v", operation, err)
	}
}

// LogError is a convenience function for logging errors asynchronously
func LogError(operation string, err error, context string) {
	GetErrorHandler().HandleError(operation, err, context)
}

// LogErrorf logs a formatted error asynchronously
func LogErrorf(operation string, err error, format string, args ...interface{}) {
	GetErrorHandler().HandleError(operation, err, fmt.Sprintf(format, args...))
}
