package user

// UserInfo represents user information.
type UserInfo struct {
	ID    string `json:"id"`
	Name  string `json:"name"`
	Email string `json:"email"`
}

// GetUserRequest is the request for getting a user.
type GetUserRequest struct {
	UserID string `json:"user_id"`
}

// GetUserResponse is the response for getting a user.
type GetUserResponse struct {
	User  *UserInfo `json:"user,omitempty"`
	Found bool      `json:"found"`
}

// ValidateUserRequest is the request for validating a user.
type ValidateUserRequest struct {
	UserID string `json:"user_id"`
}

// ValidateUserResponse is the response for validating a user.
type ValidateUserResponse struct {
	Valid bool `json:"valid"`
}
