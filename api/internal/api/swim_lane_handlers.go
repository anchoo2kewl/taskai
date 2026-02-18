package api

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"

	"taskai/ent"
	"taskai/ent/swimlane"
)

type SwimLane struct {
	ID             int64     `json:"id"`
	ProjectID      int64     `json:"project_id"`
	Name           string    `json:"name"`
	Color          string    `json:"color"`
	Position       int       `json:"position"`
	StatusCategory string    `json:"status_category"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type CreateSwimLaneRequest struct {
	Name           string `json:"name"`
	Color          string `json:"color"`
	Position       int    `json:"position"`
	StatusCategory string `json:"status_category"`
}

type UpdateSwimLaneRequest struct {
	Name           *string `json:"name,omitempty"`
	Color          *string `json:"color,omitempty"`
	Position       *int    `json:"position,omitempty"`
	StatusCategory *string `json:"status_category,omitempty"`
}

// HandleListSwimLanes returns all swim lanes for a project
func (s *Server) HandleListSwimLanes(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)
	projectID, err := strconv.ParseInt(chi.URLParam(r, "projectId"), 10, 64)
	if err != nil {
		s.logger.Warn("Invalid project ID", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid project ID", "invalid_input")
		return
	}

	// Verify user has access to this project
	hasAccess, err := s.checkProjectAccess(ctx, userID, projectID)
	if err != nil {
		s.logger.Error("Failed to verify project access", zap.Error(err), zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to verify project access", "internal_error")
		return
	}
	if !hasAccess {
		s.logger.Warn("Access denied to project", zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusForbidden, "access denied", "forbidden")
		return
	}

	entSwimLanes, err := s.db.Client.SwimLane.Query().
		Where(swimlane.ProjectID(projectID)).
		Order(ent.Asc(swimlane.FieldPosition)).
		All(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch swim lanes", zap.Error(err), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to fetch swim lanes", "internal_error")
		return
	}

	swimLanes := make([]SwimLane, 0, len(entSwimLanes))
	for _, esl := range entSwimLanes {
		swimLanes = append(swimLanes, SwimLane{
			ID:             esl.ID,
			ProjectID:      esl.ProjectID,
			Name:           esl.Name,
			Color:          esl.Color,
			Position:       esl.Position,
			StatusCategory: esl.StatusCategory,
			CreatedAt:      esl.CreatedAt,
			UpdatedAt:      esl.UpdatedAt,
		})
	}

	respondJSON(w, http.StatusOK, swimLanes)
}

// HandleCreateSwimLane creates a new swim lane
func (s *Server) HandleCreateSwimLane(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)
	projectID, err := strconv.ParseInt(chi.URLParam(r, "projectId"), 10, 64)
	if err != nil {
		s.logger.Warn("Invalid project ID", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid project ID", "invalid_input")
		return
	}

	// Verify user has access to this project
	hasAccess, err := s.checkProjectAccess(ctx, userID, projectID)
	if err != nil {
		s.logger.Error("Failed to verify project access", zap.Error(err), zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to verify project access", "internal_error")
		return
	}
	if !hasAccess {
		s.logger.Warn("Access denied to project", zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusForbidden, "access denied", "forbidden")
		return
	}

	var req CreateSwimLaneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Invalid request body", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid request body", "invalid_input")
		return
	}

	// Validation
	if req.Name == "" {
		respondError(w, http.StatusBadRequest, "swim lane name is required", "invalid_input")
		return
	}
	if len(req.Name) > 50 {
		respondError(w, http.StatusBadRequest, "swim lane name is too long (max 50 characters)", "invalid_input")
		return
	}
	if req.Color == "" {
		req.Color = "#6B7280" // default gray
	}

	// Validate status_category
	if req.StatusCategory == "" {
		respondError(w, http.StatusBadRequest, "status_category is required (must be: todo, in_progress, or done)", "invalid_input")
		return
	}
	if req.StatusCategory != "todo" && req.StatusCategory != "in_progress" && req.StatusCategory != "done" {
		respondError(w, http.StatusBadRequest, "invalid status_category (must be: todo, in_progress, or done)", "invalid_input")
		return
	}

	// Check swim lane count limit (max 6)
	count, err := s.db.Client.SwimLane.Query().
		Where(swimlane.ProjectID(projectID)).
		Count(ctx)
	if err != nil {
		s.logger.Error("Failed to count swim lanes", zap.Error(err), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to count swim lanes", "internal_error")
		return
	}
	if count >= 6 {
		respondError(w, http.StatusBadRequest, "maximum 6 swim lanes allowed per project", "max_limit_reached")
		return
	}

	// Check minimum (need at least 2)
	if req.Position < 0 {
		req.Position = 0
	}

	newSwimLane, err := s.db.Client.SwimLane.Create().
		SetProjectID(projectID).
		SetName(req.Name).
		SetColor(req.Color).
		SetPosition(req.Position).
		SetStatusCategory(req.StatusCategory).
		Save(ctx)
	if err != nil {
		s.logger.Error("Failed to create swim lane", zap.Error(err), zap.Int64("projectID", projectID), zap.String("name", req.Name))
		respondError(w, http.StatusInternalServerError, "failed to create swim lane", "internal_error")
		return
	}

	sl := SwimLane{
		ID:             newSwimLane.ID,
		ProjectID:      newSwimLane.ProjectID,
		Name:           newSwimLane.Name,
		Color:          newSwimLane.Color,
		Position:       newSwimLane.Position,
		StatusCategory: newSwimLane.StatusCategory,
		CreatedAt:      newSwimLane.CreatedAt,
		UpdatedAt:      newSwimLane.UpdatedAt,
	}

	s.logger.Info("Swim lane created", zap.Int64("swimLaneID", sl.ID), zap.Int64("projectID", projectID), zap.String("name", req.Name))
	respondJSON(w, http.StatusCreated, sl)
}

// HandleUpdateSwimLane updates an existing swim lane
func (s *Server) HandleUpdateSwimLane(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)
	swimLaneID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.logger.Warn("Invalid swim lane ID", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid swim lane ID", "invalid_input")
		return
	}

	var req UpdateSwimLaneRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		s.logger.Warn("Invalid request body", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid request body", "invalid_input")
		return
	}

	// Get swim lane and verify user has access
	swimLaneEntity, err := s.db.Client.SwimLane.Get(ctx, swimLaneID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "swim lane not found", "not_found")
			return
		}
		s.logger.Error("Failed to get swim lane", zap.Error(err), zap.Int64("swimLaneID", swimLaneID))
		respondError(w, http.StatusInternalServerError, "failed to get swim lane", "internal_error")
		return
	}

	projectID := swimLaneEntity.ProjectID

	// Verify user has access to the project
	hasAccess, err := s.checkProjectAccess(ctx, userID, projectID)
	if err != nil {
		s.logger.Error("Failed to verify project access", zap.Error(err), zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to verify project access", "internal_error")
		return
	}
	if !hasAccess {
		s.logger.Warn("Access denied to project", zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusForbidden, "access denied", "forbidden")
		return
	}

	// Build update using Ent
	updateBuilder := s.db.Client.SwimLane.UpdateOneID(swimLaneID)

	if req.Name != nil {
		if *req.Name == "" {
			respondError(w, http.StatusBadRequest, "swim lane name cannot be empty", "invalid_input")
			return
		}
		if len(*req.Name) > 50 {
			respondError(w, http.StatusBadRequest, "swim lane name is too long (max 50 characters)", "invalid_input")
			return
		}
		updateBuilder.SetName(*req.Name)
	}

	if req.Color != nil {
		updateBuilder.SetColor(*req.Color)
	}

	if req.Position != nil {
		if *req.Position < 0 {
			respondError(w, http.StatusBadRequest, "position cannot be negative", "invalid_input")
			return
		}
		updateBuilder.SetPosition(*req.Position)
	}

	if req.StatusCategory != nil {
		if *req.StatusCategory != "todo" && *req.StatusCategory != "in_progress" && *req.StatusCategory != "done" {
			respondError(w, http.StatusBadRequest, "invalid status_category (must be: todo, in_progress, or done)", "invalid_input")
			return
		}
		updateBuilder.SetStatusCategory(*req.StatusCategory)
	}

	updatedSwimLane, err := updateBuilder.Save(ctx)
	if err != nil {
		s.logger.Error("Failed to update swim lane", zap.Error(err), zap.Int64("swimLaneID", swimLaneID))
		respondError(w, http.StatusInternalServerError, "failed to update swim lane", "internal_error")
		return
	}

	sl := SwimLane{
		ID:             updatedSwimLane.ID,
		ProjectID:      updatedSwimLane.ProjectID,
		Name:           updatedSwimLane.Name,
		Color:          updatedSwimLane.Color,
		Position:       updatedSwimLane.Position,
		StatusCategory: updatedSwimLane.StatusCategory,
		CreatedAt:      updatedSwimLane.CreatedAt,
		UpdatedAt:      updatedSwimLane.UpdatedAt,
	}

	s.logger.Info("Swim lane updated", zap.Int64("swimLaneID", swimLaneID), zap.Int64("projectID", projectID))
	respondJSON(w, http.StatusOK, sl)
}

// HandleDeleteSwimLane deletes a swim lane
func (s *Server) HandleDeleteSwimLane(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)
	swimLaneID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		s.logger.Warn("Invalid swim lane ID", zap.Error(err))
		respondError(w, http.StatusBadRequest, "invalid swim lane ID", "invalid_input")
		return
	}

	// Get swim lane and verify user has access
	swimLaneEntity, err := s.db.Client.SwimLane.Get(ctx, swimLaneID)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "swim lane not found", "not_found")
			return
		}
		s.logger.Error("Failed to get swim lane", zap.Error(err), zap.Int64("swimLaneID", swimLaneID))
		respondError(w, http.StatusInternalServerError, "failed to get swim lane", "internal_error")
		return
	}

	projectID := swimLaneEntity.ProjectID

	// Verify user has access to the project
	hasAccess, err := s.checkProjectAccess(ctx, userID, projectID)
	if err != nil {
		s.logger.Error("Failed to verify project access", zap.Error(err), zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to verify project access", "internal_error")
		return
	}
	if !hasAccess {
		s.logger.Warn("Access denied to project", zap.Int64("userID", userID), zap.Int64("projectID", projectID))
		respondError(w, http.StatusForbidden, "access denied", "forbidden")
		return
	}

	// Check minimum swim lanes (need at least 2)
	count, err := s.db.Client.SwimLane.Query().
		Where(swimlane.ProjectID(projectID)).
		Count(ctx)
	if err != nil {
		s.logger.Error("Failed to count swim lanes", zap.Error(err), zap.Int64("projectID", projectID))
		respondError(w, http.StatusInternalServerError, "failed to count swim lanes", "internal_error")
		return
	}
	if count <= 2 {
		respondError(w, http.StatusBadRequest, "minimum 2 swim lanes required per project", "min_limit_reached")
		return
	}

	err = s.db.Client.SwimLane.DeleteOneID(swimLaneID).Exec(ctx)
	if err != nil {
		s.logger.Error("Failed to delete swim lane", zap.Error(err), zap.Int64("swimLaneID", swimLaneID))
		respondError(w, http.StatusInternalServerError, "failed to delete swim lane", "internal_error")
		return
	}

	s.logger.Info("Swim lane deleted", zap.Int64("swimLaneID", swimLaneID), zap.Int64("projectID", projectID))
	w.WriteHeader(http.StatusNoContent)
}
