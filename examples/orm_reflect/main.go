// Package main demonstrates the same ORM example using stdlib reflect.
// Compare with orm_lite which uses gref.
package main

import (
	"database/sql"
	"fmt"
	"reflect"
	"strings"

	_ "modernc.org/sqlite"
)

func main() {
	db, _ := sql.Open("sqlite", ":memory:")
	defer db.Close()
	setup(db)

	fmt.Println("=== ORM Example with stdlib reflect ===")
	fmt.Println()

	rows, _ := db.Query(`SELECT id, username, email, bio FROM users`)
	defer rows.Close()

	// Build struct type from query columns
	cols, _ := rows.ColumnTypes()
	userType := buildStructType("User", cols)

	fmt.Println("Dynamic struct from sql.ColumnTypes():")
	printStruct(userType)
	fmt.Println()

	// Scan and display rows
	fmt.Println("Query results:")
	for rows.Next() {
		userPtr := reflect.New(userType)
		user := userPtr.Elem()
		scanInto(rows, user)

		// Field access requires FieldByName + type assertion
		id := user.FieldByName("Id").Int()
		name := user.FieldByName("Username").String()
		email := user.FieldByName("Email").String()

		fmt.Printf("  [%d] %s <%s>\n", id, name, email)
	}
}

// buildStructType creates a struct type from column metadata
func buildStructType(_ string, cols []*sql.ColumnType) reflect.Type {
	fields := make([]reflect.StructField, len(cols))
	for i, col := range cols {
		fields[i] = reflect.StructField{
			Name: toPascal(col.Name()),
			Type: col.ScanType(),
		}
	}
	return reflect.StructOf(fields)
}

// scanInto scans a row into struct fields (in order)
func scanInto(rows *sql.Rows, v reflect.Value) {
	dest := make([]any, v.NumField())
	for i := 0; i < v.NumField(); i++ {
		dest[i] = v.Field(i).Addr().Interface()
	}
	rows.Scan(dest...)
}

func setup(db *sql.DB) {
	db.Exec(`CREATE TABLE users (id INTEGER PRIMARY KEY, username TEXT, email TEXT, bio TEXT)`)
	db.Exec(`INSERT INTO users VALUES (1,'alice','alice@example.com','Engineer'),
		(2,'bob','bob@example.com',NULL), (3,'charlie','charlie@example.com','Vacay')`)
}

func printStruct(t reflect.Type) {
	for i := 0; i < t.NumField(); i++ {
		f := t.Field(i)
		fmt.Printf("  %-10s %s\n", f.Name, f.Type)
	}
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
