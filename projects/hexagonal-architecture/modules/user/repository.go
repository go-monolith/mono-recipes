package user

import (
	"sync"
)

// UserRepository provides in-memory user storage.
type UserRepository struct {
	users map[string]*UserInfo
	mu    sync.RWMutex
}

// NewUserRepository creates a new user repository.
func NewUserRepository() *UserRepository {
	return &UserRepository{
		users: make(map[string]*UserInfo),
	}
}

// SeedDemoUsers adds demo users to the repository.
func (r *UserRepository) SeedDemoUsers() {
	r.mu.Lock()
	defer r.mu.Unlock()

	demoUsers := []*UserInfo{
		{ID: "user-1", Name: "Alice Johnson", Email: "alice@example.com"},
		{ID: "user-2", Name: "Bob Smith", Email: "bob@example.com"},
		{ID: "user-3", Name: "Charlie Brown", Email: "charlie@example.com"},
	}

	for _, u := range demoUsers {
		r.users[u.ID] = u
	}
}

// FindByID finds a user by ID.
func (r *UserRepository) FindByID(userID string) (*UserInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	user, found := r.users[userID]
	return user, found
}

// Exists checks if a user exists.
func (r *UserRepository) Exists(userID string) bool {
	r.mu.RLock()
	defer r.mu.RUnlock()

	_, found := r.users[userID]
	return found
}
