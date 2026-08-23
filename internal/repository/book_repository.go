package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jmoiron/sqlx"

	"github.com/Toph4er/family-library/internal/db"
	"github.com/Toph4er/family-library/internal/handlers/pages"
	"github.com/Toph4er/family-library/internal/models"
)

type sqliteBookRepository struct {
	db *sqlx.DB
}

func NewBookRepository(database *sqlx.DB) BookRepository {
	return &sqliteBookRepository{db: database}
}

// Create inserts a new book into the database.
func (r *sqliteBookRepository) Create(ctx context.Context, book *models.Book) error {
	query := `INSERT INTO books (isbn, title, subtitle, authors, illustrators, publisher, 
		publication_year, page_count, book_type, reading_levels, genres, themes, awards,
		gift_from, gift_relationship, date_received, condition, location, notes, child_rating,
		quantity, read_count, last_read_date, cover_image_url, cover_source, dewey_decimal_class, 
		language, subject_places, subject_people, subject_times, series, age_range, guest_visible_fields) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, 1, 0, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`

	result, err := r.db.ExecContext(ctx, query,
		db.StrToNullString(book.ISBN), book.Title, db.StrToNullString(book.Subtitle),
		db.StrToNullString(book.Authors), db.StrToNullString(book.Illustrators),
		db.StrToNullString(book.Publisher), db.IntToNullInt64(book.PublicationYear),
		db.IntToNullInt64(book.PageCount), db.StrToNullString(book.BookType),
		db.StrToNullString(book.ReadingLevels), db.StrToNullString(book.Genres),
		db.StrToNullString(book.Themes), db.StrToNullString(book.Awards),
		db.StrToNullString(book.GiftFrom), db.StrToNullString(book.GiftRelationship),
		db.StrToNullString(book.DateReceived), db.StrToNullString(book.Condition),
		db.StrToNullString(book.Location), db.StrToNullString(book.Notes),
		db.IntToNullInt64(book.ChildRating), book.Quantity, 0, nil,
		nil, "none",
		db.StrToNullString(book.DeweyDecimalClass), db.StrToNullString(book.Language),
		db.StrToNullString(book.SubjectPlaces), db.StrToNullString(book.SubjectPeople),
		db.StrToNullString(book.SubjectTimes), db.StrToNullString(book.Series),
		db.StrToNullString(book.AgeRange), book.GuestVisibleFields,
	)

	if err != nil {
		return fmt.Errorf("create book: %w", err)
	}

	id, _ := result.LastInsertId()
	book.ID = id

	if err := r.db.GetContext(ctx, book, "SELECT "+db.BookColumns+" FROM books WHERE id = ?", book.ID); err != nil { // #nosec G202 -- BookColumns is a constant
		return fmt.Errorf("fetch created book: %w", err)
	}

	if book.Quantity == 0 {
		book.Quantity = 1
	}

	return nil
}

// GetByID retrieves a book by its ID.
func (r *sqliteBookRepository) GetByID(ctx context.Context, id int64) (*models.Book, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+db.BookColumns+" FROM books WHERE id = ?", id) // #nosec G202 -- BookColumns is a constant
	book, err := db.ScanBook(row)
	if err != nil {
		return nil, fmt.Errorf("get book by id: %w", err)
	}
	return book, nil
}

// GetByISBN retrieves a book by its ISBN.
func (r *sqliteBookRepository) GetByISBN(ctx context.Context, isbn string) (*models.Book, error) {
	row := r.db.QueryRowContext(ctx, "SELECT "+db.BookColumns+" FROM books WHERE isbn = ?", isbn) // #nosec G202 -- BookColumns is a constant
	book, err := db.ScanBook(row)
	if err != nil {
		return nil, fmt.Errorf("get book by isbn: %w", err)
	}
	return book, nil
}

// UpdatePartial updates only the non-nil fields in the request.
func (r *sqliteBookRepository) UpdatePartial(ctx context.Context, id int64, input *models.UpdateBookRequest) error {
	if input == nil {
		return fmt.Errorf("update book: nil input")
	}

	var sets []string
	var args []interface{}

	if input.ISBN != nil {
		sets = append(sets, "isbn = ?")
		args = append(args, db.StrToNullString(input.ISBN))
	}
	if input.Title != nil {
		sets = append(sets, "title = ?")
		args = append(args, db.StrToNullString(input.Title))
	}
	if input.Subtitle != nil {
		sets = append(sets, "subtitle = ?")
		args = append(args, db.StrToNullString(input.Subtitle))
	}
	if input.Authors != nil {
		sets = append(sets, "authors = ?")
		args = append(args, db.StrToNullString(input.Authors))
	}
	if input.Illustrators != nil {
		sets = append(sets, "illustrators = ?")
		args = append(args, db.StrToNullString(input.Illustrators))
	}
	if input.Publisher != nil {
		sets = append(sets, "publisher = ?")
		args = append(args, db.StrToNullString(input.Publisher))
	}
	if input.PublicationYear != nil {
		sets = append(sets, "publication_year = ?")
		args = append(args, db.IntToNullInt64(input.PublicationYear))
	}
	if input.PageCount != nil {
		sets = append(sets, "page_count = ?")
		args = append(args, db.IntToNullInt64(input.PageCount))
	}
	if input.BookType != nil {
		sets = append(sets, "book_type = ?")
		args = append(args, db.StrToNullString(input.BookType))
	}
	if input.ReadingLevels != nil {
		sets = append(sets, "reading_levels = ?")
		args = append(args, db.StrToNullString(input.ReadingLevels))
	}
	if input.Genres != nil {
		sets = append(sets, "genres = ?")
		args = append(args, db.StrToNullString(input.Genres))
	}
	if input.Themes != nil {
		sets = append(sets, "themes = ?")
		args = append(args, db.StrToNullString(input.Themes))
	}
	if input.Awards != nil {
		sets = append(sets, "awards = ?")
		args = append(args, db.StrToNullString(input.Awards))
	}
	if input.GiftFrom != nil {
		sets = append(sets, "gift_from = ?")
		args = append(args, db.StrToNullString(input.GiftFrom))
	}
	if input.GiftRelationship != nil {
		sets = append(sets, "gift_relationship = ?")
		args = append(args, db.StrToNullString(input.GiftRelationship))
	}
	if input.DateReceived != nil {
		sets = append(sets, "date_received = ?")
		args = append(args, db.StrToNullString(input.DateReceived))
	}
	if input.Condition != nil {
		sets = append(sets, "condition = ?")
		args = append(args, db.StrToNullString(input.Condition))
	}
	if input.Location != nil {
		sets = append(sets, "location = ?")
		args = append(args, db.StrToNullString(input.Location))
	}
	if input.Notes != nil {
		sets = append(sets, "notes = ?")
		args = append(args, db.StrToNullString(input.Notes))
	}
	if input.ChildRating != nil {
		sets = append(sets, "child_rating = ?")
		args = append(args, db.IntToNullInt64(input.ChildRating))
	}
	if input.Quantity != nil {
		sets = append(sets, "quantity = ?")
		args = append(args, db.IntToNullInt64(input.Quantity))
	}
	if input.ReadCount != nil {
		sets = append(sets, "read_count = ?")
		args = append(args, db.IntToNullInt64(input.ReadCount))
	}
	if input.LastReadDate != nil {
		sets = append(sets, "last_read_date = ?")
		args = append(args, db.StrToNullString(input.LastReadDate))
	}
	if input.CoverImageURL != nil {
		sets = append(sets, "cover_image_url = ?")
		args = append(args, db.StrToNullString(input.CoverImageURL))
	}
	if input.CoverSource != nil {
		sets = append(sets, "cover_source = ?")
		args = append(args, db.StrToNullString(input.CoverSource))
	}
	if input.DeweyDecimalClass != nil {
		sets = append(sets, "dewey_decimal_class = ?")
		args = append(args, db.StrToNullString(input.DeweyDecimalClass))
	}
	if input.AgeRange != nil {
		sets = append(sets, "age_range = ?")
		args = append(args, db.StrToNullString(input.AgeRange))
	}
	if input.Series != nil {
		sets = append(sets, "series = ?")
		args = append(args, db.StrToNullString(input.Series))
	}
	if input.Language != nil {
		sets = append(sets, "language = ?")
		args = append(args, db.StrToNullString(input.Language))
	}
	if input.SubjectPlaces != nil {
		sets = append(sets, "subject_places = ?")
		args = append(args, db.StrToNullString(input.SubjectPlaces))
	}
	if input.SubjectPeople != nil {
		sets = append(sets, "subject_people = ?")
		args = append(args, db.StrToNullString(input.SubjectPeople))
	}
	if input.SubjectTimes != nil {
		sets = append(sets, "subject_times = ?")
		args = append(args, db.StrToNullString(input.SubjectTimes))
	}

	if len(sets) == 0 {
		return fmt.Errorf("update book: no fields to update")
	}

	sets = append(sets, "updated_at = CURRENT_TIMESTAMP")
	args = append(args, id)

	//#nosec G202 -- sets contains only hardcoded column names, not user input
	query := "UPDATE books SET " + strings.Join(sets, ", ") + " WHERE id = ?"
	result, err := r.db.ExecContext(ctx, query, args...)
	if err != nil {
		return fmt.Errorf("update book: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("update book: no rows affected")
	}

	return nil
}

// List returns books with optional filtering and pagination.
// Uses FTS5 for full-text search when a filter is provided.
//
// NOTE: no callers — verify before removing. The FTS MATCH query is sanitized
// with pages.SafeFTS5Query so raw user input can't produce an FTS5 syntax error
// (consistent with the live handlers in internal/handlers/pages).
func (r *sqliteBookRepository) List(ctx context.Context, filter string, page, perPage int) ([]models.Book, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	var total int
	var books []models.Book

	if filter != "" {
		// FTS5 search. Select from books (single table, so db.BookColumns'
		// unqualified names stay unambiguous) and restrict to ids that match
		// the FTS index. Sanitize the user query so raw FTS5 syntax can't
		// break the MATCH expression.
		safeFilter := pages.SafeFTS5Query(filter)
		countQuery := `SELECT COUNT(*) FROM books WHERE id IN (SELECT rowid FROM books_fts WHERE books_fts MATCH ?)`
		if err := r.db.QueryRowContext(ctx, countQuery, safeFilter).Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count search results: %w", err)
		}

		dataQuery := `SELECT ` + db.BookColumns + ` FROM books WHERE id IN (SELECT rowid FROM books_fts WHERE books_fts MATCH ?) ORDER BY books.title ASC LIMIT ? OFFSET ?` // #nosec G202 -- BookColumns is a constant
		if err := r.db.SelectContext(ctx, &books, dataQuery, safeFilter, perPage, offset); err != nil {
			return nil, 0, fmt.Errorf("search books: %w", err)
		}
	} else {
		if err := r.db.QueryRowContext(ctx, "SELECT COUNT(*) FROM books").Scan(&total); err != nil {
			return nil, 0, fmt.Errorf("count books: %w", err)
		}

		dataQuery := `SELECT ` + db.BookColumns + ` FROM books ORDER BY title ASC LIMIT ? OFFSET ?` // #nosec G202 -- BookColumns is a constant
		if err := r.db.SelectContext(ctx, &books, dataQuery, perPage, offset); err != nil {
			return nil, 0, fmt.Errorf("list books: %w", err)
		}
	}

	return books, total, nil
}

// Search performs a full-text search across FTS5-indexed fields.
// The query is matched against title, authors, isbn, genres, themes, awards,
// and reading_levels. If fields are specified, they are used as column
// filters (e.g., "title:term" searches only the title column).
//
// NOTE: no callers — verify before removing. The FTS MATCH query is sanitized
// with pages.SafeFTS5Query so raw user input can't produce an FTS5 syntax error
// (consistent with the live handlers in internal/handlers/pages).
func (r *sqliteBookRepository) Search(ctx context.Context, query string, fields []string, page, perPage int) ([]models.Book, int, error) {
	if page < 1 {
		page = 1
	}
	if perPage < 1 || perPage > 100 {
		perPage = 20
	}
	offset := (page - 1) * perPage

	if query == "" {
		return r.List(ctx, "", page, perPage)
	}

	// Sanitize the user query so raw FTS5 syntax can't break the MATCH
	// expression, then optionally restrict it to the given columns.
	safeQuery := pages.SafeFTS5Query(query)
	ftsQuery := safeQuery
	if len(fields) > 0 {
		// Use FTS5 column filters: "field1:term OR field2:term"
		var clauses []string
		for _, field := range fields {
			clauses = append(clauses, field+":"+safeQuery)
		}
		ftsQuery = strings.Join(clauses, " OR ")
	}

	// Select from books (single table) so db.BookColumns' unqualified names
	// stay unambiguous, restricting to ids that match the FTS index.
	countQuery := `SELECT COUNT(*) FROM books WHERE id IN (SELECT rowid FROM books_fts WHERE books_fts MATCH ?)`
	dataQuery := `SELECT ` + db.BookColumns + ` FROM books WHERE id IN (SELECT rowid FROM books_fts WHERE books_fts MATCH ?) ORDER BY books.title ASC LIMIT ? OFFSET ?` // #nosec G202 -- BookColumns is a constant

	var total int
	if err := r.db.QueryRowContext(ctx, countQuery, ftsQuery).Scan(&total); err != nil {
		return nil, 0, fmt.Errorf("count search results: %w", err)
	}

	var books []models.Book
	if err := r.db.SelectContext(ctx, &books, dataQuery, ftsQuery, perPage, offset); err != nil {
		return nil, 0, fmt.Errorf("search books: %w", err)
	}

	return books, total, nil
}

// Delete removes a book by ID.
func (r *sqliteBookRepository) Delete(ctx context.Context, id int64) error {
	result, err := r.db.ExecContext(ctx, "DELETE FROM books WHERE id = ?", id)
	if err != nil {
		return fmt.Errorf("delete book: %w", err)
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected == 0 {
		return fmt.Errorf("delete book: no rows affected")
	}

	return nil
}

// GetDistinctTags returns unique values from a JSON array column.
func (r *sqliteBookRepository) GetDistinctTags(ctx context.Context, column string) ([]string, error) {
	//#nosec G202 -- column is a hardcoded field name from callers (tags, series, etc.), not user input
	query := `SELECT DISTINCT value FROM books, json_each(books.` + column + `) WHERE value IS NOT NULL AND value != '' ORDER BY value COLLATE NOCASE`

	rows, err := r.db.QueryContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("get distinct tags: %w", err)
	}
	defer rows.Close()

	var tags []string
	for rows.Next() {
		var tag string
		if err := rows.Scan(&tag); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		tags = append(tags, tag)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("tags iteration: %w", err)
	}

	return tags, nil
}
