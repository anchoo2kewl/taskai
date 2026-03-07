package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"regexp"
	"strings"
	"time"

	"go.uber.org/zap"
)

var figmaFileKeyRe = regexp.MustCompile(`figma\.com/(?:file|design|proto)/([A-Za-z0-9]+)`)

type FigmaCredentialsStatusResponse struct {
	Configured bool `json:"configured"`
}

type SaveFigmaCredentialsRequest struct {
	AccessToken string `json:"access_token"`
}

type FigmaEmbedResponse struct {
	EmbedURL     string  `json:"embed_url"`
	Name         *string `json:"name,omitempty"`
	ThumbnailURL *string `json:"thumbnail_url,omitempty"`
	Configured   bool    `json:"configured"`
}

// HandleGetFigmaCredentials returns whether the current user has a Figma PAT configured.
func (s *Server) HandleGetFigmaCredentials(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)

	var count int
	query := convertToPostgresQuery(`SELECT COUNT(*) FROM figma_credentials WHERE user_id = ?`)
	err := s.db.QueryRowContext(ctx, query, userID).Scan(&count)
	if err != nil {
		s.logger.Error("Failed to check figma credentials", zap.Error(err), zap.Int64("user_id", userID))
		respondError(w, http.StatusInternalServerError, "failed to check credentials", "internal_error")
		return
	}

	respondJSON(w, http.StatusOK, FigmaCredentialsStatusResponse{Configured: count > 0})
}

// HandleSaveFigmaCredentials creates or updates the current user's Figma PAT.
func (s *Server) HandleSaveFigmaCredentials(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)

	var req SaveFigmaCredentialsRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", "bad_request")
		return
	}

	req.AccessToken = strings.TrimSpace(req.AccessToken)
	if req.AccessToken == "" {
		respondError(w, http.StatusBadRequest, "access_token is required", "validation_error")
		return
	}
	if !strings.HasPrefix(req.AccessToken, "figd_") {
		respondError(w, http.StatusBadRequest, "access_token must start with figd_", "validation_error")
		return
	}

	query := convertToPostgresQuery(`
		INSERT INTO figma_credentials (user_id, access_token, updated_at)
		VALUES (?, ?, CURRENT_TIMESTAMP)
		ON CONFLICT(user_id) DO UPDATE SET
		  access_token = excluded.access_token,
		  updated_at = CURRENT_TIMESTAMP`)
	if _, err := s.db.ExecContext(ctx, query, userID, req.AccessToken); err != nil {
		s.logger.Error("Failed to save figma credentials", zap.Error(err), zap.Int64("user_id", userID))
		respondError(w, http.StatusInternalServerError, "failed to save credentials", "internal_error")
		return
	}

	s.logger.Info("Figma credentials saved", zap.Int64("user_id", userID))
	respondJSON(w, http.StatusOK, FigmaCredentialsStatusResponse{Configured: true})
}

// HandleDeleteFigmaCredentials removes the current user's Figma PAT.
func (s *Server) HandleDeleteFigmaCredentials(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)

	query := convertToPostgresQuery(`DELETE FROM figma_credentials WHERE user_id = ?`)
	if _, err := s.db.ExecContext(ctx, query, userID); err != nil {
		s.logger.Error("Failed to delete figma credentials", zap.Error(err), zap.Int64("user_id", userID))
		respondError(w, http.StatusInternalServerError, "failed to delete credentials", "internal_error")
		return
	}

	s.logger.Info("Figma credentials deleted", zap.Int64("user_id", userID))
	respondJSON(w, http.StatusOK, FigmaCredentialsStatusResponse{Configured: false})
}

// HandleFigmaEmbed fetches Figma file metadata and returns embed info.
func (s *Server) HandleFigmaEmbed(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)

	figmaURL := strings.TrimSpace(r.URL.Query().Get("url"))
	if figmaURL == "" {
		respondError(w, http.StatusBadRequest, "url parameter is required", "bad_request")
		return
	}

	// Validate it's a figma.com URL
	parsed, err := url.Parse(figmaURL)
	if err != nil || !strings.Contains(parsed.Hostname(), "figma.com") {
		respondError(w, http.StatusBadRequest, "url must be a figma.com URL", "validation_error")
		return
	}

	embedURL := fmt.Sprintf("https://www.figma.com/embed?embed_host=taskai.cc&url=%s", url.QueryEscape(figmaURL))

	// Look up user's Figma token
	var accessToken string
	tokenQuery := convertToPostgresQuery(`SELECT access_token FROM figma_credentials WHERE user_id = ?`)
	err = s.db.QueryRowContext(ctx, tokenQuery, userID).Scan(&accessToken)
	if err == sql.ErrNoRows {
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: false})
		return
	}
	if err != nil {
		s.logger.Error("Failed to fetch figma credentials", zap.Error(err), zap.Int64("user_id", userID))
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: false})
		return
	}

	// Extract file key from URL
	keyMatch := figmaFileKeyRe.FindStringSubmatch(figmaURL)
	if keyMatch == nil {
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: true})
		return
	}
	fileKey := keyMatch[1]

	// Call Figma API
	apiURL := fmt.Sprintf("https://api.figma.com/v1/files/%s?depth=1", fileKey)
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, apiURL, nil)
	if err != nil {
		s.logger.Error("Failed to build figma API request", zap.Error(err))
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: true})
		return
	}
	req.Header.Set("X-Figma-Token", accessToken)

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		s.logger.Warn("Figma API request failed", zap.Error(err), zap.String("file_key", fileKey))
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: true})
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 512))
		s.logger.Warn("Figma API returned non-200", zap.Int("status", resp.StatusCode), zap.String("body", string(body)))
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: true})
		return
	}

	var figmaResp struct {
		Name        string `json:"name"`
		ThumbnailURL string `json:"thumbnailUrl"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&figmaResp); err != nil {
		s.logger.Warn("Failed to decode figma API response", zap.Error(err))
		respondJSON(w, http.StatusOK, FigmaEmbedResponse{EmbedURL: embedURL, Configured: true})
		return
	}

	result := FigmaEmbedResponse{
		EmbedURL:   embedURL,
		Configured: true,
	}
	if figmaResp.Name != "" {
		result.Name = &figmaResp.Name
	}
	if figmaResp.ThumbnailURL != "" {
		result.ThumbnailURL = &figmaResp.ThumbnailURL
	}

	respondJSON(w, http.StatusOK, result)
}
