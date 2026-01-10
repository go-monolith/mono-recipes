package user

import (
	"context"
	"errors"

	"github.com/example/sqlc-postgres-demo/modules/user/db/generated"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgtype"
)

// ErrNotFound is returned when a user is not found.
var ErrNotFound = errors.New("user not found")

// ErrDuplicateEmail is returned when email already exists.
var ErrDuplicateEmail = errors.New("email already exists")

// UserRepository defines the interface for user data access.
// This abstraction follows the Dependency Inversion Principle (DIP),
// allowing the service layer to depend on an abstraction rather than a concrete implementation.
type UserRepository interface {
	// Create saves a new user to the storage.
	Create(ctx context.Context, name, email string) (*generated.User, error)
	// FindByID retrieves a user by ID.
	FindByID(ctx context.Context, id uuid.UUID) (*generated.User, error)
	// FindAll retrieves paginated users.
	FindAll(ctx context.Context, limit, offset int32) ([]generated.User, error)
	// Count returns the total number of users.
	Count(ctx context.Context) (int64, error)
	// Update updates an existing user.
	Update(ctx context.Context, id uuid.UUID, name, email *string) (*generated.User, error)
	// Delete removes a user by ID.
	Delete(ctx context.Context, id uuid.UUID) error
}

// PostgresRepository provides PostgreSQL-based user storage using sqlc.
// It implements the UserRepository interface.
type PostgresRepository struct {
	queries *generated.Queries
}

// Compile-time interface check.
var _ UserRepository = (*PostgresRepository)(nil)

// NewPostgresRepository creates a new PostgreSQL user repository.
func NewPostgresRepository(db generated.DBTX) *PostgresRepository {
	return &PostgresRepository{
		queries: generated.New(db),
	}
}

// Create saves a new user to the database.
func (r *PostgresRepository) Create(ctx context.Context, name, email string) (*generated.User, error) {
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
func (r *PostgresRepository) FindByID(ctx context.Context, id uuid.UUID) (*generated.User, error) {
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
func (r *PostgresRepository) FindAll(ctx context.Context, limit, offset int32) ([]generated.User, error) {
	return r.queries.ListUsers(ctx, generated.ListUsersParams{
		Limit:  limit,
		Offset: offset,
	})
}

// Count returns the total number of users.
func (r *PostgresRepository) Count(ctx context.Context) (int64, error) {
	return r.queries.CountUsers(ctx)
}

// Update updates an existing user.
func (r *PostgresRepository) Update(ctx context.Context, id uuid.UUID, name, email *string) (*generated.User, error) {
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
// Returns ErrNotFound if the user does not exist.
func (r *PostgresRepository) Delete(ctx context.Context, id uuid.UUID) error {
	// Check existence first to return ErrNotFound for non-existent users
	_, err := r.FindByID(ctx, id)
	if err != nil {
		return err
	}
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
