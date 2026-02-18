package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
	"taskai/internal/version"
)

// HandleVersion returns version information about the API, database, and build
func (s *Server) HandleVersion(w http.ResponseWriter, r *http.Request) {
	// Get current database migration version
	dbVersion, err := s.db.GetMigrationVersion(r.Context())
	if err != nil {
		s.logger.Warn("Failed to get database version", zap.Error(err))
		dbVersion = 0
	}

	info := version.Get(s.config.Env, dbVersion)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(info)
}
