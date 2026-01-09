package user

import (
	"context"
	"errors"

	"github.com/example/sqlc-postgres-demo/db/generated"
	"github.com/go-monolith/mono"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

// Validation errors.
var (
	errNameRequired  = errors.New("name is required")
	errEmailRequired = errors.New("email is required")
	errIDRequired    = errors.New("id is required")
	errIDInvalid     = errors.New("id is not a valid UUID")
)

// createUser handles the user.create service request.
func (m *UserModule) createUser(ctx context.Context, req CreateUserRequest, _ *mono.Msg) (UserResponse, error) {
	if req.Name == "" {
		return UserResponse{}, errNameRequired
	}
	if req.Email == "" {
		return UserResponse{}, errEmailRequired
	}

	user, err := m.repo.Create(ctx, req.Name, req.Email)
	if err != nil {
		return UserResponse{}, err
	}

	return toUserResponse(user), nil
}

// getUser handles the user.get service request.
func (m *UserModule) getUser(ctx context.Context, req GetUserRequest, _ *mono.Msg) (UserResponse, error) {
	if req.ID == "" {
		return UserResponse{}, errIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return UserResponse{}, errIDInvalid
	}

	user, err := m.repo.FindByID(ctx, id)
	if err != nil {
		return UserResponse{}, err
	}

	return toUserResponse(user), nil
}

// listUsers handles the user.list service request with pagination.
func (m *UserModule) listUsers(ctx context.Context, req ListUsersRequest, _ *mono.Msg) (ListUsersResponse, error) {
	limit := clampLimit(req.Limit)
	offset := clampOffset(req.Offset)

	users, err := m.repo.FindAll(ctx, limit, offset)
	if err != nil {
		return ListUsersResponse{}, err
	}

	total, err := m.repo.Count(ctx)
	if err != nil {
		return ListUsersResponse{}, err
	}

	userResponses := make([]UserResponse, len(users))
	for i := range users {
		userResponses[i] = toUserResponse(&users[i])
	}

	return ListUsersResponse{
		Users:  userResponses,
		Total:  total,
		Limit:  limit,
		Offset: offset,
	}, nil
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

// updateUser handles the user.update service request.
func (m *UserModule) updateUser(ctx context.Context, req UpdateUserRequest, _ *mono.Msg) (UserResponse, error) {
	if req.ID == "" {
		return UserResponse{}, errIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return UserResponse{}, errIDInvalid
	}

	user, err := m.repo.Update(ctx, id, req.Name, req.Email)
	if err != nil {
		return UserResponse{}, err
	}

	return toUserResponse(user), nil
}

// deleteUser handles the user.delete service request.
func (m *UserModule) deleteUser(ctx context.Context, req DeleteUserRequest, _ *mono.Msg) (DeleteUserResponse, error) {
	if req.ID == "" {
		return DeleteUserResponse{}, errIDRequired
	}

	id, err := uuid.Parse(req.ID)
	if err != nil {
		return DeleteUserResponse{}, errIDInvalid
	}

	if err := m.repo.Delete(ctx, id); err != nil {
		return DeleteUserResponse{ID: req.ID}, err
	}

	return DeleteUserResponse{Deleted: true, ID: req.ID}, nil
}

func toUserResponse(user *generated.User) UserResponse {
	return UserResponse{
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
