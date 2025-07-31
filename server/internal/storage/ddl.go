package storage

import (
	_ "embed"
	"strings"
)

//go:embed schema.sql
var ddlFile string

// DefaultDDLStatements returns the CREATE TABLE / INDEX statements from schema.sql
// for test setup. It splits on semicolons and trims whitespace & comments.
func DefaultDDLStatements() []string {
	parts := strings.Split(ddlFile, ";")
	var out []string
	for _, p := range parts {
		stmt := strings.TrimSpace(p)
		if stmt == "" {
			continue
		}
		// ensure each statement ends with semicolon for clarity if needed
		out = append(out, stmt)
	}
	return out
}
