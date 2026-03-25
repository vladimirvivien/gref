// Package main demonstrates using gref with a real database.
// Uses SQLite to show dynamic struct creation from query results.
package main

import (
	"database/sql"
	"fmt"
	"strings"

	_ "modernc.org/sqlite"

	"github.com/vladimirvivien/gref"
)

func main() {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	setup(db)

	fmt.Println("=== gref ORM Example with SQLite ===")
	fmt.Println()

	rows, _ := db.Query(`SELECT id, username, email, bio FROM users`)
	defer rows.Close()

	// Build struct type from query columns
	cols, _ := rows.ColumnTypes()
	userType := buildStructType("User", cols)

	fmt.Println("Dynamic struct from sql.ColumnTypes():")
	printStruct(userType.Struct())
	fmt.Println()

	// Scan and display rows
	fmt.Println("Query results:")
	for rows.Next() {
		user := userType.Struct()
		scanInto(rows, user)

		// Type-safe field access
		id, _ := gref.Get[int64](user.Field("Id"))
		name, _ := gref.Get[string](user.Field("Username"))
		email, _ := gref.Get[string](user.Field("Email"))

		fmt.Printf("  [%d] %s <%s>\n", id, name, email)
	}
}

// buildStructType creates a struct type from column metadata
func buildStructType(name string, cols []*sql.ColumnType) *gref.StructBuilder {
	b := gref.Def().Struct(name)
	for _, col := range cols {
		b.Field(gref.FieldDef{Name: toPascal(col.Name()), Type: col.ScanType()})
	}
	return b
}

// scanInto scans a row into struct fields (in order)
func scanInto(rows *sql.Rows, s gref.Struct) {
	fields := s.Fields().Collect()
	dest := make([]any, len(fields))
	for i, f := range fields {
		dest[i] = f.Addr().Interface()
	}
	rows.Scan(dest...)
}

func setup(db *sql.DB) {
	db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, email TEXT, bio TEXT)`)
	db.Exec(`INSERT INTO users VALUES (1,'alice','alice@example.com','Engineer'),
		(2,'bob','bob@example.com',NULL), (3,'charlie','charlie@example.com','Vacay')`)
}

func printStruct(s gref.Struct) {
	s.Fields().Each(func(f gref.Field) bool {
		fmt.Printf("  %-10s %s\n", f.Name(), f.Type())
		return true
	})
}

func toPascal(s string) string {
	parts := strings.Split(s, "_")
	for i, p := range parts {
		if p != "" {
			parts[i] = strings.ToUpper(p[:1]) + p[1:]
		}
	}
	return strings.Join(parts, "")
}
