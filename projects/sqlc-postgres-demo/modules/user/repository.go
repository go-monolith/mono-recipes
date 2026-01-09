package user

import (
	"context"
	"errors"

	"github.com/example/sqlc-postgres-demo/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound is returned when a user is not found.
var ErrNotFound = errors.New("user not found")

// ErrDuplicateEmail is returned when email already exists.
var ErrDuplicateEmail = errors.New("email already exists")

// Repository provides access to user storage using sqlc.
type Repository struct {
	queries *generated.Queries
}

// NewRepository creates a new user repository.
func NewRepository(db generated.DBTX) *Repository {
	return &Repository{
		queries: generated.New(db),
	}
}

// Create saves a new user to the database.
func (r *Repository) Create(ctx context.Context, name, email string) (*generated.User, error) {
	user, err := r.queries.CreateUser(ctx, generated.CreateUserParams{
		Name:  name,
		Email: email,
	})
	if err != nil {
		if isPgDuplicateKeyError(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}
	return &user, nil
}

// FindByID retrieves a user by ID.
func (r *Repository) FindByID(ctx context.Context, id uuid.UUID) (*generated.User, error) {
	user, err := r.queries.GetUser(ctx, pgtype.UUID{Bytes: id, Valid: true})
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, err
	}
	return &user, nil
}

// FindAll retrieves paginated users.
func (r *Repository) FindAll(ctx context.Context, limit, offset int32) ([]generated.User, error) {
	return r.queries.ListUsers(ctx, generated.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

// Count returns the total number of users.
func (r *Repository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountUsers(ctx)
}

// Update updates an existing user.
func (r *Repository) Update(ctx context.Context, id uuid.UUID, name, email *string) (*generated.User, error) {
	params := generated.UpdateUserParams{
		ID: pgtype.UUID{Bytes: id, Valid: true},
	}
	if name != nil {
		params.Name = pgtype.Text{String: *name, Valid: true}
	}
	if email != nil {
		params.Email = pgtype.Text{String: *email, Valid: true}
	}

	user, err := r.queries.UpdateUser(ctx, params)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		if isPgDuplicateKeyError(err) {
			return nil, ErrDuplicateEmail
		}
		return nil, err
	}
	return &user, nil
}

// Delete removes a user by ID.
func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteUser(ctx, pgtype.UUID{Bytes: id, Valid: true})
}

// isPgDuplicateKeyError checks if error is a PostgreSQL unique violation.
func isPgDuplicateKeyError(err error) bool {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return pgErr.Code == "23505"
	}
	return false
}
