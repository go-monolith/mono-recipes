// Package contract is the published port of the user module: the DTOs of its
// services and the typed contracts other modules call them through. It is the
// only package of the user module another module may import.
//
// A contract package imports only the standard library,
// github.com/go-monolith/mono/pkg/types and .../pkg/helper.
package contract

import "github.com/go-monolith/mono/pkg/helper"

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

// GetUser returns a user by ID. Subject: services.user.get-user
var GetUser = helper.RequestReplyService[GetUserRequest, GetUserResponse]("user", "get-user")

// ValidateUser reports whether a user exists. Subject: services.user.validate-user
var ValidateUser = helper.RequestReplyService[ValidateUserRequest, ValidateUserResponse]("user", "validate-user")
