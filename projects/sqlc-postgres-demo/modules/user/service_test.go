package user

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/example/sqlc-postgres-demo/modules/user/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// mockRepository is a test double implementing UserRepository.
// This demonstrates the benefit of Dependency Inversion Principle (DIP) -
// we can test the service layer in isolation without a real database.
type mockRepository struct {
	users         map[uuid.UUID]*generated.User
	createErr     error
	findByIDErr   error
	findAllErr    error
	countErr      error
	updateErr     error
	deleteErr     error
	emailExists   bool
	returnedCount int64
}

// Compile-time interface check.
var _ UserRepository = (*mockRepository)(nil)

func newMockRepository() *mockRepository {
	return &mockRepository{
		users: make(map[uuid.UUID]*generated.User),
	}
}

func (m *mockRepository) Create(ctx context.Context, name, email string) (*generated.User, error) {
	if m.createErr != nil {
		return nil, m.createErr
	}
	if m.emailExists {
		return nil, ErrDuplicateEmail
	}

	id := uuid.New()
	now := time.Now()
	user := &generated.User{
		ID:        pgtype.UUID{Bytes: id, Valid: true},
		Name:      name,
		Email:     email,
		CreatedAt: pgtype.Timestamptz{Time: now, Valid: true},
		UpdatedAt: pgtype.Timestamptz{Time: now, Valid: true},
	}
	m.users[id] = user
	return user, nil
}

func (m *mockRepository) FindByID(_ context.Context, id uuid.UUID) (*generated.User, error) {
	if m.findByIDErr != nil {
		return nil, m.findByIDErr
	}
	user, exists := m.users[id]
	if !exists {
		return nil, ErrNotFound
	}
	return user, nil
}

func (m *mockRepository) FindAll(_ context.Context, limit, offset int32) ([]generated.User, error) {
	if m.findAllErr != nil {
		return nil, m.findAllErr
	}

	users := make([]generated.User, 0, len(m.users))
	for _, u := range m.users {
		users = append(users, *u)
	}

	// Apply pagination
	start := int(offset)
	if start >= len(users) {
		return []generated.User{}, nil
	}

	end := start + int(limit)
	if end > len(users) {
		end = len(users)
	}

	return users[start:end], nil
}

func (m *mockRepository) Count(_ context.Context) (int64, error) {
	if m.countErr != nil {
		return 0, m.countErr
	}
	if m.returnedCount > 0 {
		return m.returnedCount, nil
	}
	return int64(len(m.users)), nil
}

func (m *mockRepository) Update(_ context.Context, id uuid.UUID, name, email *string) (*generated.User, error) {
	if m.updateErr != nil {
		return nil, m.updateErr
	}
	if m.emailExists && email != nil {
		return nil, ErrDuplicateEmail
	}

	user, exists := m.users[id]
	if !exists {
		return nil, ErrNotFound
	}

	if name != nil {
		user.Name = *name
	}
	if email != nil {
		user.Email = *email
	}
	user.UpdatedAt = pgtype.Timestamptz{Time: time.Now(), Valid: true}

	return user, nil
}

func (m *mockRepository) Delete(_ context.Context, id uuid.UUID) error {
	if m.deleteErr != nil {
		return m.deleteErr
	}
	delete(m.users, id)
	return nil
}

func TestUserService_Create(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		resp, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		if resp.Name != "Test User" {
			t.Errorf("expected name %q, got %q", "Test User", resp.Name)
		}
		if resp.Email != "test@example.com" {
			t.Errorf("expected email %q, got %q", "test@example.com", resp.Email)
		}
		if resp.ID == "" {
			t.Error("expected non-empty ID")
		}
	})

	t.Run("missing name", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Create(context.Background(), CreateUserRequest{
			Email: "test@example.com",
		})
		if err != errNameRequired {
			t.Errorf("expected errNameRequired, got %v", err)
		}
	})

	t.Run("missing email", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Create(context.Background(), CreateUserRequest{
			Name: "Test User",
		})
		if err != errEmailRequired {
			t.Errorf("expected errEmailRequired, got %v", err)
		}
	})

	t.Run("duplicate email", func(t *testing.T) {
		repo := newMockRepository()
		repo.emailExists = true
		svc := NewUserService(repo)

		_, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err != ErrDuplicateEmail {
			t.Errorf("expected ErrDuplicateEmail, got %v", err)
		}
	})
}

func TestUserService_Get(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user first
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Get the user
		resp, err := svc.Get(context.Background(), GetUserRequest{ID: created.ID})
		if err != nil {
			t.Fatalf("Get() error = %v", err)
		}

		if resp.Name != created.Name {
			t.Errorf("expected name %q, got %q", created.Name, resp.Name)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Get(context.Background(), GetUserRequest{})
		if err != errIDRequired {
			t.Errorf("expected errIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Get(context.Background(), GetUserRequest{ID: "not-a-uuid"})
		if err != errIDInvalid {
			t.Errorf("expected errIDInvalid, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Get(context.Background(), GetUserRequest{
			ID: uuid.New().String(),
		})
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})
}

func TestUserService_List(t *testing.T) {
	t.Run("success with pagination", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create multiple users
		for i := range 5 {
			_, err := svc.Create(context.Background(), CreateUserRequest{
				Name:  "Test User",
				Email: fmt.Sprintf("test%d@example.com", i),
			})
			if err != nil {
				t.Fatalf("Create() error = %v", err)
			}
		}

		resp, err := svc.List(context.Background(), ListUsersRequest{
			Limit:  2,
			Offset: 0,
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if len(resp.Users) != 2 {
			t.Errorf("expected 2 users, got %d", len(resp.Users))
		}
		if resp.Total != 5 {
			t.Errorf("expected total 5, got %d", resp.Total)
		}
		if resp.Limit != 2 {
			t.Errorf("expected limit 2, got %d", resp.Limit)
		}
	})

	t.Run("default pagination", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		resp, err := svc.List(context.Background(), ListUsersRequest{})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Limit != 10 {
			t.Errorf("expected default limit 10, got %d", resp.Limit)
		}
		if resp.Offset != 0 {
			t.Errorf("expected default offset 0, got %d", resp.Offset)
		}
	})

	t.Run("clamped limit", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		resp, err := svc.List(context.Background(), ListUsersRequest{
			Limit: 200, // Exceeds max
		})
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}

		if resp.Limit != 100 {
			t.Errorf("expected clamped limit 100, got %d", resp.Limit)
		}
	})
}

func TestUserService_Update(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Original Name",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Update the user
		newName := "Updated Name"
		resp, err := svc.Update(context.Background(), UpdateUserRequest{
			ID:   created.ID,
			Name: &newName,
		})
		if err != nil {
			t.Fatalf("Update() error = %v", err)
		}

		if resp.Name != newName {
			t.Errorf("expected name %q, got %q", newName, resp.Name)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		newName := "Updated Name"
		_, err := svc.Update(context.Background(), UpdateUserRequest{
			Name: &newName,
		})
		if err != errIDRequired {
			t.Errorf("expected errIDRequired, got %v", err)
		}
	})

	t.Run("not found", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		newName := "Updated Name"
		_, err := svc.Update(context.Background(), UpdateUserRequest{
			ID:   uuid.New().String(),
			Name: &newName,
		})
		if !errors.Is(err, ErrNotFound) {
			t.Errorf("expected ErrNotFound, got %v", err)
		}
	})

	t.Run("empty name rejected", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Original Name",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Try to update with empty name
		emptyName := ""
		_, err = svc.Update(context.Background(), UpdateUserRequest{
			ID:   created.ID,
			Name: &emptyName,
		})
		if !errors.Is(err, errNameRequired) {
			t.Errorf("expected errNameRequired, got %v", err)
		}
	})

	t.Run("empty email rejected", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Original Name",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Try to update with empty email
		emptyEmail := ""
		_, err = svc.Update(context.Background(), UpdateUserRequest{
			ID:    created.ID,
			Email: &emptyEmail,
		})
		if !errors.Is(err, errEmailRequired) {
			t.Errorf("expected errEmailRequired, got %v", err)
		}
	})
}

func TestUserService_Delete(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Delete the user
		resp, err := svc.Delete(context.Background(), DeleteUserRequest{
			ID: created.ID,
		})
		if err != nil {
			t.Fatalf("Delete() error = %v", err)
		}

		if !resp.Deleted {
			t.Error("expected Deleted to be true")
		}
		if resp.ID != created.ID {
			t.Errorf("expected ID %q, got %q", created.ID, resp.ID)
		}

		// Verify deletion
		_, err = svc.Get(context.Background(), GetUserRequest{ID: created.ID})
		if err != ErrNotFound {
			t.Errorf("expected ErrNotFound after delete, got %v", err)
		}
	})

	t.Run("missing ID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Delete(context.Background(), DeleteUserRequest{})
		if err != errIDRequired {
			t.Errorf("expected errIDRequired, got %v", err)
		}
	})

	t.Run("invalid UUID", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		_, err := svc.Delete(context.Background(), DeleteUserRequest{ID: "not-a-uuid"})
		if err != errIDInvalid {
			t.Errorf("expected errIDInvalid, got %v", err)
		}
	})
}

func TestClampLimit(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected int32
	}{
		{"zero uses default", 0, 10},
		{"negative uses default", -5, 10},
		{"within range", 50, 50},
		{"at max", 100, 100},
		{"over max is clamped", 150, 100},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampLimit(tt.input)
			if result != tt.expected {
				t.Errorf("clampLimit(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestClampOffset(t *testing.T) {
	tests := []struct {
		name     string
		input    int32
		expected int32
	}{
		{"zero", 0, 0},
		{"negative is clamped", -5, 0},
		{"positive value", 50, 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := clampOffset(tt.input)
			if result != tt.expected {
				t.Errorf("clampOffset(%d) = %d, want %d", tt.input, result, tt.expected)
			}
		})
	}
}

func TestUserService_ErrorPropagation(t *testing.T) {
	t.Run("create propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		repo.createErr = errors.New("db connection failed")
		svc := NewUserService(repo)

		_, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err == nil || err.Error() != "db connection failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("get propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		repo.findByIDErr = errors.New("db query failed")
		svc := NewUserService(repo)

		// Add a user to the map so FindByID doesn't return ErrNotFound
		id := uuid.New()
		repo.users[id] = &generated.User{
			ID: pgtype.UUID{Bytes: id, Valid: true},
		}

		_, err := svc.Get(context.Background(), GetUserRequest{ID: id.String()})
		if err == nil || err.Error() != "db query failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("list propagates findAll error", func(t *testing.T) {
		repo := newMockRepository()
		repo.findAllErr = errors.New("db query failed")
		svc := NewUserService(repo)

		_, err := svc.List(context.Background(), ListUsersRequest{Limit: 10})
		if err == nil || err.Error() != "db query failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("list propagates count error", func(t *testing.T) {
		repo := newMockRepository()
		repo.countErr = errors.New("count query failed")
		svc := NewUserService(repo)

		_, err := svc.List(context.Background(), ListUsersRequest{Limit: 10})
		if err == nil || err.Error() != "count query failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("update propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user first
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Set error after creation
		repo.updateErr = errors.New("db update failed")

		newName := "Updated Name"
		_, err = svc.Update(context.Background(), UpdateUserRequest{
			ID:   created.ID,
			Name: &newName,
		})
		if err == nil || err.Error() != "db update failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})

	t.Run("delete propagates repository error", func(t *testing.T) {
		repo := newMockRepository()
		svc := NewUserService(repo)

		// Create a user first
		created, err := svc.Create(context.Background(), CreateUserRequest{
			Name:  "Test User",
			Email: "test@example.com",
		})
		if err != nil {
			t.Fatalf("Create() error = %v", err)
		}

		// Set error after creation
		repo.deleteErr = errors.New("db delete failed")

		_, err = svc.Delete(context.Background(), DeleteUserRequest{ID: created.ID})
		if err == nil || err.Error() != "db delete failed" {
			t.Errorf("expected repository error, got %v", err)
		}
	})
}
