package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"log"

	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

func main() {
	sqlitePath := flag.String("sqlite", "./data/taskai.db", "Path to SQLite database")
	postgresDSN := flag.String("postgres", "", "Postgres DSN (required)")
	flag.Parse()

	if *postgresDSN == "" {
		log.Fatal("Postgres DSN is required. Example: postgres://user:pass@localhost/dbname")
	}

	log.Println("🔄 Starting migration from SQLite to Postgres...")
	log.Printf("SQLite: %s\n", *sqlitePath)
	log.Printf("Postgres: %s\n", maskPassword(*postgresDSN))

	// Open SQLite
	sqliteDB, err := sql.Open("sqlite", *sqlitePath)
	if err != nil {
		log.Fatal("Failed to open SQLite:", err)
	}
	defer sqliteDB.Close()

	// Open Postgres
	postgresDB, err := sql.Open("pgx", *postgresDSN)
	if err != nil {
		log.Fatal("Failed to open Postgres:", err)
	}
	defer postgresDB.Close()

	// Test connections
	if err := sqliteDB.Ping(); err != nil {
		log.Fatal("Failed to ping SQLite:", err)
	}
	if err := postgresDB.Ping(); err != nil {
		log.Fatal("Failed to ping Postgres:", err)
	}

	log.Println("✅ Database connections established")

	// Get list of tables from SQLite
	tables, err := getTables(sqliteDB)
	if err != nil {
		log.Fatal("Failed to get tables:", err)
	}

	log.Printf("📊 Found %d tables to migrate\n", len(tables))

	ctx := context.Background()

	// Migrate each table
	for _, table := range tables {
		if table == "schema_migrations" {
			log.Printf("⏭️  Skipping %s (will be handled separately)\n", table)
			continue
		}

		log.Printf("📦 Migrating table: %s\n", table)
		count, err := migrateTable(ctx, sqliteDB, postgresDB, table)
		if err != nil {
			log.Printf("❌ Failed to migrate %s: %v\n", table, err)
			continue
		}
		log.Printf("✅ Migrated %s: %d rows\n", table, count)
	}

	// Migrate schema_migrations last
	log.Println("📦 Migrating schema_migrations...")
	count, err := migrateTable(ctx, sqliteDB, postgresDB, "schema_migrations")
	if err != nil {
		log.Printf("❌ Failed to migrate schema_migrations: %v\n", err)
	} else {
		log.Printf("✅ Migrated schema_migrations: %d rows\n", count)
	}

	log.Println("🎉 Migration completed successfully!")
}

func getTables(db *sql.DB) ([]string, error) {
	rows, err := db.Query("SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var name string
		if err := rows.Scan(&name); err != nil {
			return nil, err
		}
		tables = append(tables, name)
	}
	return tables, rows.Err()
}

func migrateTable(ctx context.Context, from, to *sql.DB, table string) (int, error) {
	// Get all rows from SQLite
	rows, err := from.QueryContext(ctx, fmt.Sprintf("SELECT * FROM %s", table))
	if err != nil {
		return 0, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return 0, err
	}

	// Prepare insert statement
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}

	insertSQL := fmt.Sprintf(
		"INSERT INTO %s (%s) VALUES (%s) ON CONFLICT DO NOTHING",
		table,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "),
	)

	stmt, err := to.PrepareContext(ctx, insertSQL)
	if err != nil {
		return 0, fmt.Errorf("prepare statement: %w", err)
	}
	defer stmt.Close()

	count := 0
	for rows.Next() {
		// Create slice for values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return count, fmt.Errorf("scan row: %w", err)
		}

		// Convert []byte to string for text fields
		for i, val := range values {
			if b, ok := val.([]byte); ok {
				values[i] = string(b)
			}
		}

		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			return count, fmt.Errorf("insert row: %w", err)
		}
		count++
	}

	return count, rows.Err()
}

func joinStrings(strs []string, sep string) string {
	if len(strs) == 0 {
		return ""
	}
	result := strs[0]
	for i := 1; i < len(strs); i++ {
		result += sep + strs[i]
	}
	return result
}

func maskPassword(dsn string) string {
	// Simple password masking for logs
	start := 0
	for i := 0; i < len(dsn); i++ {
		if dsn[i] == ':' && i+2 < len(dsn) && dsn[i+1] == '/' && dsn[i+2] == '/' {
			start = i + 3
			break
		}
	}

	for i := start; i < len(dsn); i++ {
		if dsn[i] == ':' {
			for j := i + 1; j < len(dsn); j++ {
				if dsn[j] == '@' {
					return dsn[:i+1] + "***" + dsn[j:]
				}
			}
		}
	}
	return dsn
}
