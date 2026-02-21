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

	// Define table migration order (respecting foreign key dependencies)
	tableOrder := []string{
		"users",           // No dependencies
		"projects",        // Depends on users (owner_id)
		"teams",          // Depends on users (owner_id)
		"team_members",   // Depends on teams, users
		"team_invitations", // Depends on teams
		"project_members", // Depends on projects, users
		"project_invitations", // Depends on projects, users
		"swim_lanes",     // Depends on projects
		"tasks",          // Depends on projects, users
		"sprints",        // Depends on projects
		"tags",           // Depends on projects
		"task_tags",      // Depends on tasks, tags
		"task_comments",  // Depends on tasks, users
		"task_attachments", // Depends on tasks, users
		"user_activity",  // Depends on users
		"api_keys",       // Depends on users
		"cloudinary_credentials", // Depends on users
		"invites",        // Depends on users (inviter_id)
		"email_provider", // No dependencies
		"wiki_pages",     // Depends on projects, users
		"yjs_updates",    // Depends on wiki_pages
		"page_versions",  // Depends on wiki_pages
		"wiki_blocks",    // Depends on wiki_pages
	}

	// Migrate tables in order
	for _, table := range tableOrder {
		// Check if table exists
		found := false
		for _, t := range tables {
			if t == table {
				found = true
				break
			}
		}
		if !found {
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

	// Migrate any remaining tables not in the order list
	for _, table := range tables {
		if table == "schema_migrations" {
			continue
		}

		// Check if already migrated
		alreadyMigrated := false
		for _, ordered := range tableOrder {
			if table == ordered {
				alreadyMigrated = true
				break
			}
		}
		if alreadyMigrated {
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

		// Convert values for Postgres compatibility
		values = convertValues(table, columns, values)

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

func convertValues(table string, columns []string, values []interface{}) []interface{} {
	// Define boolean columns per table (SQLite stores as INTEGER 0/1)
	booleanColumns := map[string][]string{
		"users":       {"is_admin", "totp_enabled"},
		"projects":    {"is_public", "github_sync_enabled"},
		"invites":     {"used"},
	}

	result := make([]interface{}, len(values))
	for i, val := range values {
		// Convert []byte to string for text fields
		if b, ok := val.([]byte); ok {
			result[i] = string(b)
			continue
		}

		// Convert INTEGER (0/1) to BOOLEAN for known boolean columns
		if boolCols, ok := booleanColumns[table]; ok {
			for _, boolCol := range boolCols {
				if columns[i] == boolCol {
					if intVal, ok := val.(int64); ok {
						result[i] = intVal != 0
						goto next
					}
				}
			}
		}

		result[i] = val
	next:
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
