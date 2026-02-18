package main

import (
	"context"
	"database/sql"
	"flag"
	"fmt"
	"os"
	"strings"

	"go.uber.org/zap"
	_ "github.com/jackc/pgx/v5/stdlib"
	_ "modernc.org/sqlite"
)

const batchSize = 1000

func main() {
	// Parse command-line flags
	sourceDB := flag.String("source", "", "Source SQLite database path (e.g., ./data/taskai.db)")
	destDSN := flag.String("dest", "", "Destination Postgres DSN (e.g., postgres://user:pass@host/db)")
	dryRun := flag.Bool("dry-run", false, "Show what would be migrated without actually migrating")
	flag.Parse()

	if *sourceDB == "" || *destDSN == "" {
		fmt.Println("Usage: migrate-db -source <sqlite-path> -dest <postgres-dsn>")
		fmt.Println("\nExample:")
		fmt.Println("  migrate-db -source ./data/taskai.db -dest 'postgres://taskai:password@localhost/taskai'")
		os.Exit(1)
	}

	// Initialize logger
	logger, _ := zap.NewProduction()
	defer logger.Sync()

	logger.Info("Starting database migration",
		zap.String("source", *sourceDB),
		zap.String("dest", maskPassword(*destDSN)),
		zap.Bool("dry_run", *dryRun))

	// Connect to source SQLite database
	src, err := sql.Open("sqlite", *sourceDB)
	if err != nil {
		logger.Fatal("Failed to connect to source database", zap.Error(err))
	}
	defer src.Close()

	// Verify source connection
	if err := src.Ping(); err != nil {
		logger.Fatal("Failed to ping source database", zap.Error(err))
	}
	logger.Info("Connected to source SQLite database")

	if *dryRun {
		logger.Info("DRY RUN mode - no changes will be made to destination")
		if err := analyzeTables(src, logger); err != nil {
			logger.Fatal("Analysis failed", zap.Error(err))
		}
		return
	}

	// Connect to destination Postgres database
	dest, err := sql.Open("pgx", *destDSN)
	if err != nil {
		logger.Fatal("Failed to connect to destination database", zap.Error(err))
	}
	defer dest.Close()

	// Verify destination connection
	if err := dest.Ping(); err != nil {
		logger.Fatal("Failed to ping destination database", zap.Error(err))
	}
	logger.Info("Connected to destination Postgres database")

	// Perform migration
	ctx := context.Background()
	if err := migrateTables(ctx, src, dest, logger); err != nil {
		logger.Fatal("Migration failed", zap.Error(err))
	}

	logger.Info("Migration completed successfully!")
}

// analyzeTables shows what would be migrated in dry-run mode
func analyzeTables(src *sql.DB, logger *zap.Logger) error {
	ctx := context.Background()

	// Get list of tables from SQLite
	tables, err := getTables(ctx, src)
	if err != nil {
		return fmt.Errorf("failed to get tables: %w", err)
	}

	logger.Info("Found tables in source database", zap.Int("count", len(tables)))

	totalRows := 0
	for _, table := range tables {
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := src.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return fmt.Errorf("failed to count rows in %s: %w", table, err)
		}
		logger.Info("Table analysis",
			zap.String("table", table),
			zap.Int("rows", count))
		totalRows += count
	}

	logger.Info("Migration summary",
		zap.Int("total_tables", len(tables)),
		zap.Int("total_rows", totalRows))

	return nil
}

// migrateTables performs the actual migration
func migrateTables(ctx context.Context, src, dest *sql.DB, logger *zap.Logger) error {
	// Define table order (respecting foreign key dependencies)
	tableOrder := []string{
		"users",
		"user_activity",
		"api_keys",
		"email_provider",
		"invites",
		"teams",
		"team_members",
		"team_invitations",
		"projects",
		"project_members",
		"project_invitations",
		"swim_lanes",
		"sprints",
		"tags",
		"tasks",
		"task_tags",
		"task_comments",
		"task_attachments",
		"cloudinary_credentials",
	}

	// Migrate each table
	for _, table := range tableOrder {
		logger.Info("Migrating table", zap.String("table", table))

		// Check if table exists in source
		exists := tableExists(ctx, src, table)
		if !exists {
			logger.Warn("Table not found in source, skipping", zap.String("table", table))
			continue
		}

		// Get row count
		var count int
		query := fmt.Sprintf("SELECT COUNT(*) FROM %s", table)
		if err := src.QueryRowContext(ctx, query).Scan(&count); err != nil {
			return fmt.Errorf("failed to count rows in %s: %w", table, err)
		}

		if count == 0 {
			logger.Info("Table is empty, skipping", zap.String("table", table))
			continue
		}

		// Migrate data
		if err := migrateTable(ctx, src, dest, table, logger); err != nil {
			return fmt.Errorf("failed to migrate table %s: %w", table, err)
		}

		// Update sequence for BIGSERIAL columns
		if err := updateSequence(ctx, dest, table, logger); err != nil {
			logger.Warn("Failed to update sequence", zap.String("table", table), zap.Error(err))
		}

		// Verify row count
		var destCount int
		if err := dest.QueryRowContext(ctx, query).Scan(&destCount); err != nil {
			return fmt.Errorf("failed to verify row count in %s: %w", table, err)
		}

		if count != destCount {
			return fmt.Errorf("row count mismatch for %s: source=%d, dest=%d", table, count, destCount)
		}

		logger.Info("Table migrated successfully",
			zap.String("table", table),
			zap.Int("rows", count))
	}

	return nil
}

// migrateTable copies all rows from source table to destination
func migrateTable(ctx context.Context, src, dest *sql.DB, table string, logger *zap.Logger) error {
	// Get column names
	columns, err := getColumns(ctx, src, table)
	if err != nil {
		return fmt.Errorf("failed to get columns: %w", err)
	}

	columnList := strings.Join(columns, ", ")
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
	}
	placeholderList := strings.Join(placeholders, ", ")

	// Prepare insert statement
	insertSQL := fmt.Sprintf("INSERT INTO %s (%s) VALUES (%s)", table, columnList, placeholderList)
	insertStmt, err := dest.PrepareContext(ctx, insertSQL)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer insertStmt.Close()

	// Select all rows from source
	selectSQL := fmt.Sprintf("SELECT %s FROM %s", columnList, table)
	rows, err := src.QueryContext(ctx, selectSQL)
	if err != nil {
		return fmt.Errorf("failed to query source table: %w", err)
	}
	defer rows.Close()

	// Insert rows in batches
	batch := 0
	count := 0

	tx, err := dest.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	for rows.Next() {
		// Scan row values
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to scan row: %w", err)
		}

		// Insert into destination
		if _, err := tx.StmtContext(ctx, insertStmt).ExecContext(ctx, values...); err != nil {
			tx.Rollback()
			return fmt.Errorf("failed to insert row: %w", err)
		}

		count++

		// Commit batch every batchSize rows
		if count%batchSize == 0 {
			if err := tx.Commit(); err != nil {
				return fmt.Errorf("failed to commit batch: %w", err)
			}
			batch++
			logger.Info("Batch committed",
				zap.String("table", table),
				zap.Int("batch", batch),
				zap.Int("rows", count))

			// Start new transaction
			tx, err = dest.BeginTx(ctx, nil)
			if err != nil {
				return fmt.Errorf("failed to begin new transaction: %w", err)
			}
		}
	}

	// Commit remaining rows
	if count%batchSize != 0 {
		if err := tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit final batch: %w", err)
		}
	}

	return rows.Err()
}

// updateSequence updates the PostgreSQL sequence for a table's id column
func updateSequence(ctx context.Context, dest *sql.DB, table string, logger *zap.Logger) error {
	// Check if table has an id column
	var hasID bool
	checkSQL := `
		SELECT EXISTS (
			SELECT 1 FROM information_schema.columns
			WHERE table_name = $1 AND column_name = 'id'
		)
	`
	if err := dest.QueryRowContext(ctx, checkSQL, table).Scan(&hasID); err != nil {
		return err
	}

	if !hasID {
		return nil
	}

	// Get max ID value
	var maxID sql.NullInt64
	query := fmt.Sprintf("SELECT MAX(id) FROM %s", table)
	if err := dest.QueryRowContext(ctx, query).Scan(&maxID); err != nil {
		return err
	}

	if !maxID.Valid || maxID.Int64 == 0 {
		return nil
	}

	// Update sequence
	sequenceName := fmt.Sprintf("%s_id_seq", table)
	updateSQL := fmt.Sprintf("SELECT setval('%s', $1)", sequenceName)
	if _, err := dest.ExecContext(ctx, updateSQL, maxID.Int64); err != nil {
		return err
	}

	logger.Info("Sequence updated",
		zap.String("table", table),
		zap.String("sequence", sequenceName),
		zap.Int64("value", maxID.Int64))

	return nil
}

// Helper functions

func tableExists(ctx context.Context, db *sql.DB, table string) bool {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name=?"
	var name string
	err := db.QueryRowContext(ctx, query, table).Scan(&name)
	return err == nil
}

func getTables(ctx context.Context, db *sql.DB) ([]string, error) {
	query := "SELECT name FROM sqlite_master WHERE type='table' AND name NOT LIKE 'sqlite_%' ORDER BY name"
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var tables []string
	for rows.Next() {
		var table string
		if err := rows.Scan(&table); err != nil {
			return nil, err
		}
		tables = append(tables, table)
	}

	return tables, rows.Err()
}

func getColumns(ctx context.Context, db *sql.DB, table string) ([]string, error) {
	query := fmt.Sprintf("PRAGMA table_info(%s)", table)
	rows, err := db.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var columns []string
	for rows.Next() {
		var cid int
		var name, typ string
		var notnull, pk int
		var dfltValue sql.NullString

		if err := rows.Scan(&cid, &name, &typ, &notnull, &dfltValue, &pk); err != nil {
			return nil, err
		}
		columns = append(columns, name)
	}

	return columns, rows.Err()
}

func maskPassword(dsn string) string {
	if strings.Contains(dsn, "@") {
		parts := strings.Split(dsn, "@")
		if len(parts) > 1 {
			// Mask everything before @
			return "postgres://***@" + parts[1]
		}
	}
	return dsn
}
