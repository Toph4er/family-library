package handlers

import (
	"strings"
)

// updateField represents a single column=value pair for a dynamic UPDATE.
// When raw is true, the value is treated as a raw SQL expression (e.g.,
// "CURRENT_TIMESTAMP") instead of a parameterized argument.
type updateField struct {
	column string
	value  interface{}
	raw    bool
}

// rawField creates an updateField with a raw SQL expression (not parameterized).
// Use for SQL functions like CURRENT_TIMESTAMP, NULL, etc.
func rawField(column, expression string) updateField {
	return updateField{column: column, value: expression, raw: true}
}

// buildUpdateClauses takes a list of updateField and returns the SET clause
// string and the args slice for use in db.Exec.
func buildUpdateClauses(fields []updateField) (string, []interface{}) {
	sets := []string{}
	args := []interface{}{}
	for _, f := range fields {
		if f.raw {
			// #nosec G202 -- raw values are hardcoded SQL expressions (CURRENT_TIMESTAMP, NULL), never user input
			sets = append(sets, f.column+" = "+f.value.(string))
		} else {
			sets = append(sets, f.column+" = ?")
			args = append(args, f.value)
		}
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
