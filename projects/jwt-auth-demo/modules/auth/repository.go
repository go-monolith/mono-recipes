package auth

import (
	"errors"

	domain "github.com/example/jwt-auth-demo/domain/user"
	"gorm.io/gorm"
)

var (
	// ErrUserNotFound is returned when a user is not found.
	ErrUserNotFound = errors.New("user not found")
	// ErrUserExists is returned when a user already exists.
	ErrUserExists = errors.New("user with this email already exists")
)

// UserRepository handles user persistence using GORM.
type UserRepository struct {
	db *gorm.DB
}

// NewUserRepository creates a new UserRepository.
func NewUserRepository(db *gorm.DB) *UserRepository {
	return &UserRepository{
		db: db,
	}
}

// Create creates a new user in the database.
func (r *UserRepository) Create(user *domain.User) error {
	result := r.db.Create(user)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrDuplicatedKey) {
			return ErrUserExists
		}
		return result.Error
	}
	return nil
}

// FindByID finds a user by ID.
func (r *UserRepository) FindByID(id string) (*domain.User, error) {
	var user domain.User
	result := r.db.First(&user, "id = ?", id)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

// FindByEmail finds a user by email.
func (r *UserRepository) FindByEmail(email string) (*domain.User, error) {
	var user domain.User
	result := r.db.First(&user, "email = ?", email)
	if result.Error != nil {
		if errors.Is(result.Error, gorm.ErrRecordNotFound) {
			return nil, ErrUserNotFound
		}
		return nil, result.Error
	}
	return &user, nil
}

// EmailExists checks if a user with the given email exists.
func (r *UserRepository) EmailExists(email string) (bool, error) {
	var count int64
	result := r.db.Model(&domain.User{}).Where("email = ?", email).Count(&count)
	if result.Error != nil {
		return false, result.Error
	}
	return count > 0, nil
}
