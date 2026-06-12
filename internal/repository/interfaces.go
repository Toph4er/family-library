// Package repository defines data access interfaces for the library application.
package repository

import (
	"context"

	"git.rcsmaine.com/chris/library/internal/models"
)

// BookRepository handles all database operations for books.
type BookRepository interface {
	Create(ctx context.Context, book *models.Book) error
	GetByID(ctx context.Context, id int64) (*models.Book, error)
	GetByISBN(ctx context.Context, isbn string) (*models.Book, error)
	UpdatePartial(ctx context.Context, id int64, input *models.UpdateBookRequest) error
	List(ctx context.Context, filter string, page, perPage int) ([]models.Book, int, error)
	Search(ctx context.Context, query string, fields []string, page, perPage int) ([]models.Book, int, error)
	Delete(ctx context.Context, id int64) error
	GetDistinctTags(ctx context.Context, column string) ([]string, error)
}

// WishlistRepository handles all database operations for wishlist items.
type WishlistRepository interface {
	List(ctx context.Context) ([]models.WishlistItem, error)
	Create(ctx context.Context, item *models.WishlistItem) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.WishlistItem, error)
	UpdatePartial(ctx context.Context, id int64, item *models.WishlistItem) error
	Delete(ctx context.Context, id int64) error
	Fulfill(ctx context.Context, id int64) error
}

// UserRepository handles all database operations for user accounts.
type UserRepository interface {
	List(ctx context.Context) ([]*models.User, error)
	Create(ctx context.Context, user *models.CreateUserRequest) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.User, error)
	UpdatePartial(ctx context.Context, id int64, input *models.UpdateUserRequest) error
	Delete(ctx context.Context, id int64) error
}

// FamilyMemberRepository handles all database operations for family members.
type FamilyMemberRepository interface {
	List(ctx context.Context) ([]*models.FamilyMember, error)
	Create(ctx context.Context, member *models.FamilyMember) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.FamilyMember, error)
	Update(ctx context.Context, id int64, member *models.FamilyMember) error
	Delete(ctx context.Context, id int64) error
}

// ReadingLogRepository handles all database operations for reading logs.
type ReadingLogRepository interface {
	List(ctx context.Context) ([]*models.ReadingLogWithBook, error)
	Create(ctx context.Context, log *models.ReadingLog) (int64, error)
	GetByID(ctx context.Context, id int64) (*models.ReadingLogWithBook, error)
	UpdatePartial(ctx context.Context, id int64, input *models.UpdateReadingLogRequest) error
	Delete(ctx context.Context, id int64) error
}

// SettingsRepository handles all database operations for application settings.
type SettingsRepository interface {
	GetByKey(ctx context.Context, key string) (*models.Setting, error)
	Set(ctx context.Context, key, value string) error
	List(ctx context.Context) ([]*models.Setting, error)
	UpdateGuestVisibility(ctx context.Context, visibility map[string]interface{}) error
}
