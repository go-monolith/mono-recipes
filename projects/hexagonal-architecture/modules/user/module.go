package user

import (
	"context"
	"encoding/json"
	"fmt"
	"log"

	"github.com/go-monolith/mono"
	"github.com/go-monolith/mono/pkg/helper"
)

// UserModule provides user management services.
type UserModule struct {
	repo *UserRepository
}

// Compile-time interface checks.
var _ mono.Module = (*UserModule)(nil)
var _ mono.ServiceProviderModule = (*UserModule)(nil)

// NewModule creates a new UserModule.
func NewModule() *UserModule {
	return &UserModule{
		repo: NewUserRepository(),
	}
}

// Name returns the module name.
func (m *UserModule) Name() string {
	return "user"
}

// RegisterServices registers request-reply services in the service container.
// This implements the ServiceProviderModule interface.
func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
	// Register get-user service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"get-user",
		json.Unmarshal,
		json.Marshal,
		m.getUser,
	); err != nil {
		return fmt.Errorf("failed to register get-user service: %w", err)
	}

	// Register validate-user service
	if err := helper.RegisterTypedRequestReplyService(
		container,
		"validate-user",
		json.Unmarshal,
		json.Marshal,
		m.validateUser,
	); err != nil {
		return fmt.Errorf("failed to register validate-user service: %w", err)
	}

	log.Printf("[user] Registered services: get-user, validate-user")
	return nil
}

// getUser handles the get-user service request.
func (m *UserModule) getUser(_ context.Context, req GetUserRequest, _ *mono.Msg) (GetUserResponse, error) {
	user, found := m.repo.FindByID(req.UserID)
	if !found {
		return GetUserResponse{Found: false}, nil
	}

	return GetUserResponse{
		User:  user,
		Found: true,
	}, nil
}

// validateUser handles the validate-user service request.
func (m *UserModule) validateUser(_ context.Context, req ValidateUserRequest, _ *mono.Msg) (ValidateUserResponse, error) {
	exists := m.repo.Exists(req.UserID)
	return ValidateUserResponse{Valid: exists}, nil
}

// Start initializes the module.
func (m *UserModule) Start(_ context.Context) error {
	// Seed demo users
	m.repo.SeedDemoUsers()
	log.Println("[user] Module started with demo users")
	return nil
}

// Stop shuts down the module.
func (m *UserModule) Stop(_ context.Context) error {
	log.Println("[user] Module stopped")
	return nil
}
