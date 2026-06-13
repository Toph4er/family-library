package handlers

import (
	"strings"
)

// updateField represents a single column=value pair for a dynamic UPDATE.
type updateField struct {
	column string
	value  interface{}
}

// buildUpdateClauses takes a list of updateField and returns the SET clause
// string and the args slice for use in db.Exec.
func buildUpdateClauses(fields []updateField) (string, []interface{}) {
	sets := []string{}
	args := []interface{}{}
	for _, f := range fields {
		sets = append(sets, f.column+" = ?")
		args = append(args, f.value)
	}
	return strings.Join(sets, ", "), args
}

// ptrIfNonEmpty returns a pointer to s if s is non-empty, nil otherwise.
func ptrIfNonEmpty(s string) *string {
	if s != "" {
		return &s
	}
	return nil
}
