package tests

import (
	"context"
	"testing"

	"github.com/Toph4er/family-library/internal/repository"
)

// The repository List()/Search() methods had raw FTS5 MATCH queries (no
// SafeFTS5Query) and are dead code (no callers). They were hardened to use
// pages.SafeFTS5Query so that, if ever wired up, user input like apostrophes,
// colons, or "*" can't produce an FTS5 syntax error / 500. These tests confirm
// the hardened methods accept previously-broken inputs without erroring.

var ftsPoisonQueries = []string{
	"We're going",
	"don't",
	`it's "quoted"`,
	"a - b",
	"col1:col2",
	"*",
	"(unclosed",
}

func TestRepositorySearch_HardenedAgainstFTSSyntax(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")
	repo := repository.NewBookRepository(env.db)
	ctx := context.Background()

	// No column filter.
	for _, q := range ftsPoisonQueries {
		if _, _, err := repo.Search(ctx, q, nil, 1, 20); err != nil {
			t.Errorf("Search(%q, nil) -> error: %v", q, err)
		}
	}

	// With a column filter (exercises the field:term branch).
	for _, q := range ftsPoisonQueries {
		if _, _, err := repo.Search(ctx, q, []string{"title", "authors"}, 1, 20); err != nil {
			t.Errorf("Search(%q, [title authors]) -> error: %v", q, err)
		}
	}

	// Positive: the sanitized query must still find the seeded book.
	books, total, err := repo.Search(ctx, "going on a trip", nil, 1, 20)
	if err != nil {
		t.Fatalf("Search('going on a trip') -> error: %v", err)
	}
	if total != 1 || len(books) != 1 || books[0].Title != "We're going on a trip" {
		t.Errorf("expected the seeded book, got total=%d books=%+v", total, books)
	}
}

func TestRepositoryList_HardenedAgainstFTSSyntax(t *testing.T) {
	env := setupTestEnv(t)
	seedBook(t, env, "We're going on a trip", "A. Smith")
	repo := repository.NewBookRepository(env.db)
	ctx := context.Background()

	for _, q := range ftsPoisonQueries {
		if _, _, err := repo.List(ctx, q, 1, 20); err != nil {
			t.Errorf("List(%q) -> error: %v", q, err)
		}
	}

	// Positive: the sanitized filter must still find the seeded book.
	books, total, err := repo.List(ctx, "going on a trip", 1, 20)
	if err != nil {
		t.Fatalf("List('going on a trip') -> error: %v", err)
	}
	if total != 1 || len(books) != 1 || books[0].Title != "We're going on a trip" {
		t.Errorf("expected the seeded book, got total=%d books=%+v", total, books)
	}
}
