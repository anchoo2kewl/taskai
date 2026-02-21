package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"
)

// BackupData represents the complete database backup
type BackupData struct {
	Version       int         `json:"version"`        // Schema migration version
	ExportedAt    time.Time   `json:"exported_at"`
	ExportedBy    int64       `json:"exported_by"`
	Tables        TableData   `json:"tables"`
}

// TableData contains all table data
type TableData struct {
	Users                  []map[string]interface{} `json:"users"`
	Teams                  []map[string]interface{} `json:"teams"`
	TeamMembers            []map[string]interface{} `json:"team_members"`
	Projects               []map[string]interface{} `json:"projects"`
	ProjectMembers         []map[string]interface{} `json:"project_members"`
	ProjectInvitations     []map[string]interface{} `json:"project_invitations"`
	Sprints                []map[string]interface{} `json:"sprints"`
	SwimLanes              []map[string]interface{} `json:"swim_lanes"`
	Tasks                  []map[string]interface{} `json:"tasks"`
	TaskComments           []map[string]interface{} `json:"task_comments"`
	TaskAttachments        []map[string]interface{} `json:"task_attachments"`
	Tags                   []map[string]interface{} `json:"tags"`
	TaskTags               []map[string]interface{} `json:"task_tags"`
	Invites                []map[string]interface{} `json:"invites"`
	TeamInvitations        []map[string]interface{} `json:"team_invitations"`
	UserActivity           []map[string]interface{} `json:"user_activity"`
	APIKeys                []map[string]interface{} `json:"api_keys"`
	CloudinaryCredentials  []map[string]interface{} `json:"cloudinary_credentials"`
	EmailProvider          []map[string]interface{} `json:"email_provider"`
}

// HandleExportData exports all database data (admin only)
func (s *Server) HandleExportData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated", "unauthorized")
		return
	}

	// Check if user is admin
	if !s.isAdmin(r.Context(), userID) {
		respondError(w, http.StatusForbidden, "admin access required", "forbidden")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 60*time.Second)
	defer cancel()

	s.logger.Info("Starting data export", zap.Int64("user_id", userID))

	// Get current migration version
	version, err := s.getCurrentMigrationVersion(ctx)
	if err != nil {
		s.logger.Error("Failed to get migration version", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get schema version", "internal_error")
		return
	}

	backup := BackupData{
		Version:    version,
		ExportedAt: time.Now(),
		ExportedBy: userID,
		Tables:     TableData{},
	}

	// Export all tables
	tables := []struct {
		name   string
		target *[]map[string]interface{}
	}{
		{"users", &backup.Tables.Users},
		{"teams", &backup.Tables.Teams},
		{"team_members", &backup.Tables.TeamMembers},
		{"projects", &backup.Tables.Projects},
		{"project_members", &backup.Tables.ProjectMembers},
		{"project_invitations", &backup.Tables.ProjectInvitations},
		{"sprints", &backup.Tables.Sprints},
		{"swim_lanes", &backup.Tables.SwimLanes},
		{"tasks", &backup.Tables.Tasks},
		{"task_comments", &backup.Tables.TaskComments},
		{"task_attachments", &backup.Tables.TaskAttachments},
		{"tags", &backup.Tables.Tags},
		{"task_tags", &backup.Tables.TaskTags},
		{"invites", &backup.Tables.Invites},
		{"team_invitations", &backup.Tables.TeamInvitations},
		{"user_activity", &backup.Tables.UserActivity},
		{"api_keys", &backup.Tables.APIKeys},
		{"cloudinary_credentials", &backup.Tables.CloudinaryCredentials},
		{"email_provider", &backup.Tables.EmailProvider},
	}

	for _, table := range tables {
		data, err := s.exportTable(ctx, table.name)
		if err != nil {
			s.logger.Error("Failed to export table", zap.String("table", table.name), zap.Error(err))
			respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to export %s", table.name), "internal_error")
			return
		}
		*table.target = data
		s.logger.Info("Exported table", zap.String("table", table.name), zap.Int("rows", len(data)))
	}

	s.logger.Info("Export completed", zap.Int("version", version), zap.Int64("user_id", userID))

	// Set headers for download
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=taskai-backup-%s.json", time.Now().Format("20060102-150405")))

	respondJSON(w, http.StatusOK, backup)
}

// HandleImportData imports database data (admin only)
func (s *Server) HandleImportData(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated", "unauthorized")
		return
	}

	// Check if user is admin
	if !s.isAdmin(r.Context(), userID) {
		respondError(w, http.StatusForbidden, "admin access required", "forbidden")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 120*time.Second)
	defer cancel()

	var backup BackupData
	if err := json.NewDecoder(r.Body).Decode(&backup); err != nil {
		respondError(w, http.StatusBadRequest, "invalid backup file", "validation_error")
		return
	}

	// Verify migration version matches
	currentVersion, err := s.getCurrentMigrationVersion(ctx)
	if err != nil {
		s.logger.Error("Failed to get migration version", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get schema version", "internal_error")
		return
	}

	if backup.Version != currentVersion {
		s.logger.Warn("Migration version mismatch",
			zap.Int("backup_version", backup.Version),
			zap.Int("current_version", currentVersion))
		respondError(w, http.StatusBadRequest,
			fmt.Sprintf("migration version mismatch: backup is v%d, database is v%d", backup.Version, currentVersion),
			"version_mismatch")
		return
	}

	s.logger.Info("Starting data import",
		zap.Int64("user_id", userID),
		zap.Int("version", backup.Version),
		zap.Time("backup_date", backup.ExportedAt))

	// Import all tables in dependency order
	tables := []struct {
		name string
		data []map[string]interface{}
	}{
		{"users", backup.Tables.Users},
		{"teams", backup.Tables.Teams},
		{"team_members", backup.Tables.TeamMembers},
		{"projects", backup.Tables.Projects},
		{"project_members", backup.Tables.ProjectMembers},
		{"project_invitations", backup.Tables.ProjectInvitations},
		{"sprints", backup.Tables.Sprints},
		{"swim_lanes", backup.Tables.SwimLanes},
		{"tasks", backup.Tables.Tasks},
		{"task_comments", backup.Tables.TaskComments},
		{"task_attachments", backup.Tables.TaskAttachments},
		{"tags", backup.Tables.Tags},
		{"task_tags", backup.Tables.TaskTags},
		{"invites", backup.Tables.Invites},
		{"team_invitations", backup.Tables.TeamInvitations},
		{"user_activity", backup.Tables.UserActivity},
		{"api_keys", backup.Tables.APIKeys},
		{"cloudinary_credentials", backup.Tables.CloudinaryCredentials},
		{"email_provider", backup.Tables.EmailProvider},
	}

	for _, table := range tables {
		if len(table.data) > 0 {
			if err := s.importTable(ctx, table.name, table.data); err != nil {
				s.logger.Error("Failed to import table", zap.String("table", table.name), zap.Error(err))
				respondError(w, http.StatusInternalServerError, fmt.Sprintf("failed to import %s", table.name), "internal_error")
				return
			}
			s.logger.Info("Imported table", zap.String("table", table.name), zap.Int("rows", len(table.data)))
		}
	}

	s.logger.Info("Import completed", zap.Int64("user_id", userID))

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"message": "Data imported successfully",
		"version": backup.Version,
		"rows":    countTotalRows(backup.Tables),
	})
}

// exportTable exports all data from a table
func (s *Server) exportTable(ctx context.Context, tableName string) ([]map[string]interface{}, error) {
	query := fmt.Sprintf("SELECT * FROM %s", tableName)
	rows, err := s.db.DB.QueryContext(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	columns, err := rows.Columns()
	if err != nil {
		return nil, err
	}

	var result []map[string]interface{}
	for rows.Next() {
		values := make([]interface{}, len(columns))
		valuePtrs := make([]interface{}, len(columns))
		for i := range values {
			valuePtrs[i] = &values[i]
		}

		if err := rows.Scan(valuePtrs...); err != nil {
			return nil, err
		}

		row := make(map[string]interface{})
		for i, col := range columns {
			val := values[i]
			// Convert []byte to string for easier JSON handling
			if b, ok := val.([]byte); ok {
				row[col] = string(b)
			} else {
				row[col] = val
			}
		}
		result = append(result, row)
	}

	return result, rows.Err()
}

// importTable imports data into a table
func (s *Server) importTable(ctx context.Context, tableName string, data []map[string]interface{}) error {
	if len(data) == 0 {
		return nil
	}

	// Get column names from first row
	var columns []string
	for col := range data[0] {
		columns = append(columns, col)
	}

	// Build INSERT query
	placeholders := make([]string, len(columns))
	for i := range placeholders {
		placeholders[i] = "?"
	}

	query := fmt.Sprintf("INSERT OR REPLACE INTO %s (%s) VALUES (%s)",
		tableName,
		joinStrings(columns, ", "),
		joinStrings(placeholders, ", "))

	stmt, err := s.db.DB.PrepareContext(ctx, query)
	if err != nil {
		return err
	}
	defer stmt.Close()

	for _, row := range data {
		values := make([]interface{}, len(columns))
		for i, col := range columns {
			values[i] = row[col]
		}
		if _, err := stmt.ExecContext(ctx, values...); err != nil {
			return err
		}
	}

	return nil
}

// getCurrentMigrationVersion gets the current migration version from schema_migrations
func (s *Server) getCurrentMigrationVersion(ctx context.Context) (int, error) {
	var count int
	query := "SELECT COUNT(*) FROM schema_migrations"
	err := s.db.DB.QueryRowContext(ctx, query).Scan(&count)
	return count, err
}

// Helper functions
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

func countTotalRows(tables TableData) int {
	return len(tables.Users) +
		len(tables.Teams) +
		len(tables.TeamMembers) +
		len(tables.Projects) +
		len(tables.ProjectMembers) +
		len(tables.ProjectInvitations) +
		len(tables.Sprints) +
		len(tables.SwimLanes) +
		len(tables.Tasks) +
		len(tables.TaskComments) +
		len(tables.TaskAttachments) +
		len(tables.Tags) +
		len(tables.TaskTags) +
		len(tables.Invites) +
		len(tables.TeamInvitations) +
		len(tables.UserActivity) +
		len(tables.APIKeys) +
		len(tables.CloudinaryCredentials) +
		len(tables.EmailProvider)
}
