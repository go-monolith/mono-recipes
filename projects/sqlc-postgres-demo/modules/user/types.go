package user

import "time"

// CreateUserRequest is the request for creating a user.
type CreateUserRequest struct {
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetUserRequest is the request for getting a user by ID.
type GetUserRequest struct {
	ID string `json:"id"`
}

// UserResponse represents a user in API responses.
type UserResponse struct {
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Email     string    `json:"email"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// ListUsersRequest is the request for listing users with pagination.
type ListUsersRequest struct {
	Limit  int32 `json:"limit"`  // Default: 10, Max: 100
	Offset int32 `json:"offset"` // Default: 0
}

// ListUsersResponse is the response containing paginated users.
type ListUsersResponse struct {
	Users  []UserResponse `json:"users"`
	Total  int64          `json:"total"`
	Limit  int32          `json:"limit"`
	Offset int32          `json:"offset"`
}

// UpdateUserRequest is the request for updating a user.
type UpdateUserRequest struct {
	ID    string  `json:"id"`
	Name  *string `json:"name,omitempty"`
	Email *string `json:"email,omitempty"`
}

// DeleteUserRequest is the request for deleting a user.
type DeleteUserRequest struct {
	ID string `json:"id"`
}

// DeleteUserResponse is the response after deleting a user.
type DeleteUserResponse struct {
	Deleted bool   `json:"deleted"`
	ID      string `json:"id"`
}
