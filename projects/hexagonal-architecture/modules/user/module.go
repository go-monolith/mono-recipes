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

var _ mono.Module = (*UserModule)(nil)
var _ mono.ServiceProviderModule = (*UserModule)(nil)

func NewModule() *UserModule {
	return &UserModule{
		repo: NewUserRepository(),
	}
}

func (m *UserModule) Name() string {
	return "user"
}

func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
	if err := helper.RegisterTypedRequestReplyService(
		container, "get-user", json.Unmarshal, json.Marshal, m.getUser,
	); err != nil {
		return fmt.Errorf("failed to register get-user service: %w", err)
	}

	if err := helper.RegisterTypedRequestReplyService(
		container, "validate-user", json.Unmarshal, json.Marshal, m.validateUser,
	); err != nil {
		return fmt.Errorf("failed to register validate-user service: %w", err)
	}

	log.Printf("[user] Registered services: get-user, validate-user")
	return nil
}

func (m *UserModule) getUser(_ context.Context, req GetUserRequest, _ *mono.Msg) (GetUserResponse, error) {
	user, found := m.repo.FindByID(req.UserID)
	if !found {
		return GetUserResponse{Found: false}, nil
	}
	return GetUserResponse{User: user, Found: true}, nil
}

func (m *UserModule) validateUser(_ context.Context, req ValidateUserRequest, _ *mono.Msg) (ValidateUserResponse, error) {
	return ValidateUserResponse{Valid: m.repo.Exists(req.UserID)}, nil
}

func (m *UserModule) Start(_ context.Context) error {
	m.repo.SeedDemoUsers()
	log.Println("[user] Module started with demo users")
	return nil
}

func (m *UserModule) Stop(_ context.Context) error {
	log.Println("[user] Module stopped")
	return nil
}
