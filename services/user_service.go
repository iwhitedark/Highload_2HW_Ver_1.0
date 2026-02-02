package services

import (
	"errors"
	"sync"
	"sync/atomic"

	"go-microservice/metrics"
	"go-microservice/models"
)

// UserService handles business logic for user operations
type UserService struct {
	users     map[int]*models.User
	mu        sync.RWMutex
	idCounter int64
}

var (
	userServiceInstance *UserService
	userServiceOnce     sync.Once
)

// GetUserService returns a singleton instance of UserService
func GetUserService() *UserService {
	userServiceOnce.Do(func() {
		userServiceInstance = &UserService{
			users:     make(map[int]*models.User),
			idCounter: 0,
		}
	})
	return userServiceInstance
}

// Create creates a new user and returns it with assigned ID
func (s *UserService) Create(user models.User) (*models.User, error) {
	// Sanitize input
	user.Sanitize()

	// Validate user data
	if err := user.Validate(); err != nil {
		return nil, err
	}

	// Generate new ID atomically
	newID := int(atomic.AddInt64(&s.idCounter, 1))
	user.ID = newID

	// Store user
	s.mu.Lock()
	s.users[newID] = &user
	count := len(s.users)
	s.mu.Unlock()

	// Update metrics
	metrics.SetActiveUsers(float64(count))

	return &user, nil
}

// GetByID retrieves a user by ID
func (s *UserService) GetByID(id int) (*models.User, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	user, exists := s.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Return a copy to prevent data races
	userCopy := *user
	return &userCopy, nil
}

// GetAll retrieves all users
func (s *UserService) GetAll() []*models.User {
	s.mu.RLock()
	defer s.mu.RUnlock()

	users := make([]*models.User, 0, len(s.users))
	for _, user := range s.users {
		// Return copies to prevent data races
		userCopy := *user
		users = append(users, &userCopy)
	}
	return users
}

// Update updates an existing user
func (s *UserService) Update(id int, updated models.User) (*models.User, error) {
	// Sanitize input
	updated.Sanitize()

	// Validate user data
	if err := updated.Validate(); err != nil {
		return nil, err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	existing, exists := s.users[id]
	if !exists {
		return nil, errors.New("user not found")
	}

	// Update fields while preserving ID
	existing.Name = updated.Name
	existing.Email = updated.Email

	// Return a copy
	userCopy := *existing
	return &userCopy, nil
}

// Delete removes a user by ID
func (s *UserService) Delete(id int) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.users[id]; !exists {
		return errors.New("user not found")
	}

	delete(s.users, id)

	// Update metrics
	metrics.SetActiveUsers(float64(len(s.users)))

	return nil
}

// Count returns the number of users
func (s *UserService) Count() int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return len(s.users)
}

// Exists checks if a user exists
func (s *UserService) Exists(id int) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	_, exists := s.users[id]
	return exists
}

// Clear removes all users (for testing purposes)
func (s *UserService) Clear() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.users = make(map[int]*models.User)
	atomic.StoreInt64(&s.idCounter, 0)
	metrics.SetActiveUsers(0)
}
