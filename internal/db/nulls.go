// Package db provides database initialization and connection management.
package db

import (
	"database/sql"
	"time"
)

// NullStrPtr converts a sql.NullString to *string. Returns nil if not valid.
func NullStrPtr(ns sql.NullString) *string {
	if !ns.Valid {
		return nil
	}
	s := ns.String
	return &s
}

// NullIntPtr64 converts a sql.NullInt64 to *int64. Returns nil if not valid.
func NullIntPtr64(ni sql.NullInt64) *int64 {
	if !ni.Valid {
		return nil
	}
	i := ni.Int64
	return &i
}

// NullIntPtr converts a sql.NullInt64 to *int. Returns nil if not valid.
func NullIntPtr(ni sql.NullInt64) *int {
	if !ni.Valid {
		return nil
	}
	i := int(ni.Int64)
	return &i
}

// NullTimePtr converts a sql.NullTime to *string (RFC3339). Returns nil if not valid.
func NullTimePtr(nt sql.NullTime) *string {
	if !nt.Valid {
		return nil
	}
	s := nt.Time.Format(time.RFC3339)
	return &s
}

// NullBoolPtr converts a sql.NullBool to *bool. Returns nil if not valid.
func NullBoolPtr(nb sql.NullBool) *bool {
	if !nb.Valid {
		return nil
	}
	b := nb.Bool
	return &b
}

// StrToNullString converts a *string to sql.NullString.
// Returns an empty NullString (Valid=false) when s is nil.
func StrToNullString(s *string) sql.NullString {
	if s == nil {
		return sql.NullString{}
	}
	return sql.NullString{String: *s, Valid: true}
}

// IntToNullInt64 converts a *int to sql.NullInt64.
// Returns an empty NullInt64 (Valid=false) when i is nil.
func IntToNullInt64(i *int) sql.NullInt64 {
	if i == nil {
		return sql.NullInt64{}
	}
	return sql.NullInt64{Int64: int64(*i), Valid: true}
}
