package repository

import (
	"database/sql"
	"fmt"
	"social-network/internal/apperror"
	"strings"
)

// requireOneRow is used to return an error if the statement affected no rows.
// This is because SQLite treats updating/deleting a row that doesn't exist (i.e., 0 rows affected) as a success
func requireOneRow(res sql.Result, entity string) error {
	n, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if n == 0 {
		return apperror.NotFound(entity + " not found")
	}
	return nil
}


func nullableString(s string) any {
	if s == "" {
		return nil
	}
	return s
}


func isSQLiteUniqueViolation(err error) bool {
	if err == nil {
		return false
	}

	msg := err.Error()

	return strings.Contains(msg, "UNIQUE constraint failed") || strings.Contains(msg, "unique constraint failed")
}