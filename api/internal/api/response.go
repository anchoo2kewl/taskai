package api

import (
	"encoding/json"
	"net/http"

	"go.uber.org/zap"
)

// pkgLogger is the package-level logger for response helpers.
var pkgLogger *zap.Logger = zap.NewNop()

// SetLogger sets the package-level logger for response helpers.
func SetLogger(l *zap.Logger) {
	pkgLogger = l
}

// ErrorResponse represents an error response
type ErrorResponse struct {
	Error string `json:"error"`
	Code  string `json:"code,omitempty"`
}

// respondJSON sends a JSON response
func respondJSON(w http.ResponseWriter, statusCode int, data interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)

	if data != nil {
		encoder := json.NewEncoder(w)
		encoder.SetEscapeHTML(false) // Don't escape HTML characters like < and >
		if err := encoder.Encode(data); err != nil {
			pkgLogger.Error("Error encoding JSON response", zap.Error(err))
		}
	}
}

// respondError sends an error response
func respondError(w http.ResponseWriter, statusCode int, message, code string) {
	respondJSON(w, statusCode, ErrorResponse{
		Error: message,
		Code:  code,
	})
}

// decodeJSON decodes a JSON request body
func decodeJSON(r *http.Request, v interface{}) error {
	if r.Body == nil {
		return http.ErrBodyNotAllowed
	}
	defer r.Body.Close()
	return json.NewDecoder(r.Body).Decode(v)
}
