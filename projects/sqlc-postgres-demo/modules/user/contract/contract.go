// Package contract is the published port of the user module: the DTOs of its
// request-reply services and the typed contracts that both the module
// (provider side) and any caller (consumer side) use. It is the ONLY package
// of the user module another module may import.
//
// Rule: a contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper. Never the user
// module package and never the sqlc-generated db/generated package: the DTOs
// below are mapped from generated.User inside the module.
package contract

import (
	"time"

	"github.com/go-monolith/mono/pkg/helper"
)

// Typed request-reply contracts. Each one is declared once: the user module
// registers its handler through it and callers call through it, so the
// service name, the services.user.<service> subject and the JSON encoding
// live here only.
var (
	// CreateUser is services.user.create.
	CreateUser = helper.RequestReplyService[CreateUserRequest, UserResponse]("user", "create")
	// GetUser is services.user.get.
	GetUser = helper.RequestReplyService[GetUserRequest, UserResponse]("user", "get")
	// ListUsers is services.user.list.
	ListUsers = helper.RequestReplyService[ListUsersRequest, ListUsersResponse]("user", "list")
	// UpdateUser is services.user.update.
	UpdateUser = helper.RequestReplyService[UpdateUserRequest, UserResponse]("user", "update")
	// DeleteUser is services.user.delete.
	DeleteUser = helper.RequestReplyService[DeleteUserRequest, DeleteUserResponse]("user", "delete")
)

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
