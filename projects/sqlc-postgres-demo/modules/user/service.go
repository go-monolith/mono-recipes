package user

import (
	"context"
	"errors"

	"github.com/example/sqlc-postgres-demo/modules/user/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"

	"github.com/example/sqlc-postgres-demo/modules/user/contract"
)

// Validation errors.
var (
	errNameRequired  = errors.New("name is required")
	errEmailRequired = errors.New("email is required")
	errIDRequired    = errors.New("id is required")
	errIDInvalid     = errors.New("id is not a valid UUID")
)

// UserService defines the interface for user business operations.
// This abstraction follows the Interface Segregation Principle (ISP) and
// Dependency Inversion Principle (DIP), allowing the module to depend on
// an abstraction rather than a concrete implementation.
type UserService interface {
	// Create creates a new user with the given name and email.
	Create(ctx context.Context, req contract.CreateUserRequest) (contract.UserResponse, error)
	// Get retrieves a user by ID.
	Get(ctx context.Context, req contract.GetUserRequest) (contract.UserResponse, error)
	// List retrieves paginated users.
	List(ctx context.Context, req contract.ListUsersRequest) (contract.ListUsersResponse, error)
	// Update updates an existing user.
	Update(ctx context.Context, req contract.UpdateUserRequest) (contract.UserResponse, error)
	// Delete removes a user by ID.
	Delete(ctx context.Context, req contract.DeleteUserRequest) (contract.DeleteUserResponse, error)
}

// UserServiceImpl implements UserService using a UserRepository.
// This follows the Single Responsibility Principle (SRP) - handling business
// logic separately from data access and framework concerns.
type UserServiceImpl struct {
	repo UserRepository
}

// Compile-time interface check.
var _ UserService = (*UserServiceImpl)(nil)

// NewUserService creates a new UserService with the given repository.
func NewUserService(repo UserRepository) UserService {
	return &UserServiceImpl{
		repo: repo,
	}
}

// Create handles the user creation request.
func (s *UserServiceImpl) Create(ctx context.Context, req contract.CreateUserRequest) (contract.UserResponse, error) {
	if req.Name == "" {
		return contract.UserResponse{}, errNameRequired
	}
	if req.Email == "" {
		return contract.UserResponse{}, errEmailRequired
	}

	user, err := s.repo.Create(ctx, req.Name, req.Email)
	if err != nil {
		return contract.UserResponse{}, err
	}

	return toUserResponse(user), nil
}

// Get handles the user retrieval request.
func (s *UserServiceImpl) Get(ctx context.Context, req contract.GetUserRequest) (contract.UserResponse, error) {
	if req.ID == "" {
		return contract.UserResponse{}, errIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return contract.UserResponse{}, errIDInvalid
	}

	user, err := s.repo.FindByID(ctx, id)
	if err != nil {
		return contract.UserResponse{}, err
	}

	return toUserResponse(user), nil
}

// List handles the user list request with pagination.
func (s *UserServiceImpl) List(ctx context.Context, req contract.ListUsersRequest) (contract.ListUsersResponse, error) {
	limit := clampLimit(req.Limit)
	offset := clampOffset(req.Offset)

	users, err := s.repo.FindAll(ctx, limit, offset)
	if err != nil {
		return contract.ListUsersResponse{}, err
	}

	total, err := s.repo.Count(ctx)
	if err != nil {
		return contract.ListUsersResponse{}, err
	}

	userResponses := make([]contract.UserResponse, len(users))
	for i := range users {
		userResponses[i] = toUserResponse(&users[i])
	}

	return contract.ListUsersResponse{
		Users:  userResponses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
}

// Update handles the user update request.
func (s *UserServiceImpl) Update(ctx context.Context, req contract.UpdateUserRequest) (contract.UserResponse, error) {
	if req.ID == "" {
		return contract.UserResponse{}, errIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return contract.UserResponse{}, errIDInvalid
	}

	// Validate non-empty values when pointers are provided
	if req.Name != nil && *req.Name == "" {
		return contract.UserResponse{}, errNameRequired
	}
	if req.Email != nil && *req.Email == "" {
		return contract.UserResponse{}, errEmailRequired
	}

	user, err := s.repo.Update(ctx, id, req.Name, req.Email)
	if err != nil {
		return contract.UserResponse{}, err
	}

	return toUserResponse(user), nil
}

// Delete handles the user deletion request.
func (s *UserServiceImpl) Delete(ctx context.Context, req contract.DeleteUserRequest) (contract.DeleteUserResponse, error) {
	if req.ID == "" {
		return contract.DeleteUserResponse{}, errIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return contract.DeleteUserResponse{}, errIDInvalid
	}

	if err := s.repo.Delete(ctx, id); err != nil {
		return contract.DeleteUserResponse{ID: req.ID}, err
	}

	return contract.DeleteUserResponse{Deleted: true, ID: req.ID}, nil
}

// clampLimit ensures limit is within valid bounds (1-100, default 10).
func clampLimit(limit int32) int32 {
	if limit <= 0 {
		return 10
	}
	if limit > 100 {
		return 100
	}
	return limit
}

// clampOffset ensures offset is non-negative.
func clampOffset(offset int32) int32 {
	if offset < 0 {
		return 0
	}
	return offset
}

func toUserResponse(user *generated.User) contract.UserResponse {
	return contract.UserResponse{
		ID:        uuidToString(user.ID),
		Name:      user.Name,
		Email:     user.Email,
		CreatedAt: user.CreatedAt.Time,
		UpdatedAt: user.UpdatedAt.Time,
	}
}

func uuidToString(id pgtype.UUID) string {
	if !id.Valid {
		return ""
	}
	return uuid.UUID(id.Bytes).String()
}
