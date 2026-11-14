package user_test

import (
	"context"
	"errors"
	"fmt"
	"os"
	"sync"
	"testing"
	"time"

	"github.com/go-monolith/mono/monotest"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/example/sqlc-postgres-demo/modules/user"
	"github.com/example/sqlc-postgres-demo/modules/user/contract"
)

// TestUserModuleConformance is the one-per-module boundary check. The service
// layer is replaced by an in-memory fake (user.NewModuleWithService), so the
// test needs neither PostgreSQL nor NATS: it checks the module's name, that it
// registers its services through valid subjects, that every contract round
// trips, and that Start/Stop succeed.
func TestUserModuleConformance(t *testing.T) {
	monotest.CheckModule(t, user.NewModuleWithService(newFakeService()),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseCRUD(ctx, h, "alice@example.com")
		}))
}

// TestUserModuleConformancePostgres runs the same check against the real
// PostgreSQL-backed module. It is skipped unless a database is reachable at
// TEST_DATABASE_URL (default: the docker-compose database).
func TestUserModuleConformancePostgres(t *testing.T) {
	t.Setenv("DATABASE_URL", requireDatabase(t))
	email := fmt.Sprintf("test-%s@example.com", uuid.NewString())
	monotest.CheckModule(t, user.NewModule(),
		monotest.WithExercise(func(ctx context.Context, h *monotest.Harness) error {
			return exerciseCRUD(ctx, h, email)
		}))
}

// exerciseCRUD calls every user contract once, in CRUD order.
func exerciseCRUD(ctx context.Context, h *monotest.Harness, email string) error {
	c := h.Container()

	created, err := contract.CreateUser.Call(ctx, c, contract.CreateUserRequest{Name: "Alice", Email: email})
	if err != nil {
		return fmt.Errorf("create: %w", err)
	}

	got, err := contract.GetUser.Call(ctx, c, contract.GetUserRequest{ID: created.ID})
	if err != nil {
		return fmt.Errorf("get: %w", err)
	}
	if got.Email != email {
		return fmt.Errorf("get: email = %q, want %q", got.Email, email)
	}

	list, err := contract.ListUsers.Call(ctx, c, contract.ListUsersRequest{Limit: 100})
	if err != nil {
		return fmt.Errorf("list: %w", err)
	}
	if list.Total < 1 {
		return fmt.Errorf("list: total = %d, want >= 1", list.Total)
	}

	name := "Alice Smith"
	updated, err := contract.UpdateUser.Call(ctx, c, contract.UpdateUserRequest{ID: created.ID, Name: &name})
	if err != nil {
		return fmt.Errorf("update: %w", err)
	}
	if updated.Name != name {
		return fmt.Errorf("update: name = %q, want %q", updated.Name, name)
	}

	deleted, err := contract.DeleteUser.Call(ctx, c, contract.DeleteUserRequest{ID: created.ID})
	if err != nil {
		return fmt.Errorf("delete: %w", err)
	}
	if !deleted.Deleted {
		return errors.New("delete: deleted = false")
	}

	if _, err := contract.GetUser.Call(ctx, c, contract.GetUserRequest{ID: created.ID}); err == nil {
		return errors.New("get after delete: expected an error")
	}
	return nil
}

// requireDatabase returns the test database URL, or skips the test when no
// PostgreSQL server answers there.
func requireDatabase(t *testing.T) string {
	t.Helper()
	url := os.Getenv("TEST_DATABASE_URL")
	if url == "" {
		url = "postgres://demo:demo123@localhost:5432/users_db?sslmode=disable"
	}
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	pool, err := pgxpool.New(ctx, url)
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}
	defer pool.Close()
	if err := pool.Ping(ctx); err != nil {
		t.Skipf("Skipping test: database ping failed: %v", err)
	}
	return url
}

// fakeService is an in-memory user.UserService. It is safe for concurrent
// use, like the real service, because request-reply handlers run on a worker
// pool.
type fakeService struct {
	mu    sync.Mutex
	users map[string]contract.UserResponse
}

var _ user.UserService = (*fakeService)(nil)

func newFakeService() *fakeService {
	return &fakeService{users: make(map[string]contract.UserResponse)}
}

func (s *fakeService) Create(_ context.Context, req contract.CreateUserRequest) (contract.UserResponse, error) {
	if req.Name == "" || req.Email == "" {
		return contract.UserResponse{}, errors.New("name and email are required")
	}
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()
	u := contract.UserResponse{ID: uuid.NewString(), Name: req.Name, Email: req.Email, CreatedAt: now, UpdatedAt: now}
	s.users[u.ID] = u
	return u, nil
}

func (s *fakeService) Get(_ context.Context, req contract.GetUserRequest) (contract.UserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[req.ID]
	if !ok {
		return contract.UserResponse{}, user.ErrNotFound
	}
	return u, nil
}

func (s *fakeService) List(_ context.Context, req contract.ListUsersRequest) (contract.ListUsersResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	resp := contract.ListUsersResponse{Users: make([]contract.UserResponse, 0, len(s.users)), Total: int64(len(s.users)), Limit: req.Limit, Offset: req.Offset}
	for _, u := range s.users {
		resp.Users = append(resp.Users, u)
	}
	return resp, nil
}

func (s *fakeService) Update(_ context.Context, req contract.UpdateUserRequest) (contract.UserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	u, ok := s.users[req.ID]
	if !ok {
		return contract.UserResponse{}, user.ErrNotFound
	}
	if req.Name != nil {
		u.Name = *req.Name
	}
	if req.Email != nil {
		u.Email = *req.Email
	}
	u.UpdatedAt = time.Now()
	s.users[u.ID] = u
	return u, nil
}

func (s *fakeService) Delete(_ context.Context, req contract.DeleteUserRequest) (contract.DeleteUserResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if _, ok := s.users[req.ID]; !ok {
		return contract.DeleteUserResponse{ID: req.ID}, user.ErrNotFound
	}
	delete(s.users, req.ID)
	return contract.DeleteUserResponse{Deleted: true, ID: req.ID}, nil
}
