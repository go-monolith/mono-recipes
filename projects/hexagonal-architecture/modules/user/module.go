// Package user holds the user module: an in-memory user directory exposed
// through the services declared in user/contract. Other modules import
// user/contract, never this package.
package user

import (
	"context"
	"fmt"
	"log"

	"github.com/go-monolith/mono"

	"github.com/example/hexagonal-architecture/modules/user/contract"
)

// UserModule provides user management services.
type UserModule struct {
	repo *UserRepository
}

var (
	_ mono.Module                = (*UserModule)(nil)
	_ mono.ServiceProviderModule = (*UserModule)(nil)
)

// NewModule creates the user module.
func NewModule() *UserModule {
	return &UserModule{
		repo: NewUserRepository(),
	}
}

func (m *UserModule) Name() string {
	return "user"
}

// RegisterServices is the driving adapter: it binds the domain handlers to the
// module's own contracts.
func (m *UserModule) RegisterServices(container mono.ServiceContainer) error {
	if err := contract.GetUser.Register(container, m.getUser); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.GetUser.Name, err)
	}
	if err := contract.ValidateUser.Register(container, m.validateUser); err != nil {
		return fmt.Errorf("failed to register %s: %w", contract.ValidateUser.Name, err)
	}

	log.Printf("[user] Registered services: %s, %s", contract.GetUser.Name, contract.ValidateUser.Name)
	return nil
}

func (m *UserModule) getUser(_ context.Context, req contract.GetUserRequest, _ *mono.Msg) (contract.GetUserResponse, error) {
	user, found := m.repo.FindByID(req.UserID)
	if !found {
		return contract.GetUserResponse{Found: false}, nil
	}
	return contract.GetUserResponse{User: user, Found: true}, nil
}

func (m *UserModule) validateUser(_ context.Context, req contract.ValidateUserRequest, _ *mono.Msg) (contract.ValidateUserResponse, error) {
	return contract.ValidateUserResponse{Valid: m.repo.Exists(req.UserID)}, nil
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
