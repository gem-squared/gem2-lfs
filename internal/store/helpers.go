package store

import (
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"strings"
)

// newID generates a random 16-character hex ID.
func newID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return hex.EncodeToString(b)
}

// nullStr is a nullable string that scans SQL NULLs as empty string.
type nullStr string

func (n *nullStr) Scan(value any) error {
	var ns sql.NullString
	if err := ns.Scan(value); err != nil {
		return err
	}
	if ns.Valid {
		*n = nullStr(ns.String)
	} else {
		*n = ""
	}
	return nil
}

func defaultStr(s, def string) string {
	if s == "" {
		return def
	}
	return s
}

func joinStr(parts []string, sep string) string {
	return strings.Join(parts, sep)
}

// ftsSearchIDs performs 3-strategy FTS5 search fallback:
//   1. Exact phrase match via FTS5 MATCH with quoted query
//   2. Tokenized match via FTS5 MATCH with OR-joined tokens
//   3. Fuzzy LIKE fallback on specified columns
//
// Returns a SQL fragment "rowid IN (...)" and its args.
// The table parameter is the source table (e.g. "tasks"), ftsTable is the FTS
// virtual table (e.g. "tasks_fts"), and likeColumns are columns to LIKE-search.
func (s *DB) ftsSearchIDs(table, ftsTable, query string, likeColumns []string) (string, []any) {
	// Strategy 1: exact phrase via FTS5.
	exactQuery := `"` + strings.ReplaceAll(query, `"`, `""`) + `"`
	var count int
	row := s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s MATCH ?", ftsTable, ftsTable), exactQuery)
	if row.Scan(&count) == nil && count > 0 {
		return fmt.Sprintf("rowid IN (SELECT rowid FROM %s WHERE %s MATCH ?)", ftsTable, ftsTable), []any{exactQuery}
	}

	// Strategy 2: tokenized OR match via FTS5.
	tokens := strings.Fields(query)
	if len(tokens) > 0 {
		tokenQuery := strings.Join(tokens, " OR ")
		row = s.db.QueryRow(fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s MATCH ?", ftsTable, ftsTable), tokenQuery)
		if row.Scan(&count) == nil && count > 0 {
			return fmt.Sprintf("rowid IN (SELECT rowid FROM %s WHERE %s MATCH ?)", ftsTable, ftsTable), []any{tokenQuery}
		}
	}

	// Strategy 3: fuzzy LIKE fallback on source table columns.
	if len(likeColumns) > 0 {
		var likeParts []string
		var likeArgs []any
		pattern := "%" + query + "%"
		for _, col := range likeColumns {
			likeParts = append(likeParts, fmt.Sprintf("%s LIKE ?", col))
			likeArgs = append(likeArgs, pattern)
		}
		return fmt.Sprintf("(%s)", strings.Join(likeParts, " OR ")), likeArgs
	}

	// No match possible.
	return "1=0", nil
}
