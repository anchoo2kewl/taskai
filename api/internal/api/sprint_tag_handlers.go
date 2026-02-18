package api

import (
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"

	"taskai/ent"
	"taskai/ent/sprint"
	"taskai/ent/tag"
)

// Sprint represents a sprint
type Sprint struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	Goal      string    `json:"goal,omitempty"`
	StartDate string    `json:"start_date,omitempty"`
	EndDate   string    `json:"end_date,omitempty"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

// Tag represents a tag
type Tag struct {
	ID        int       `json:"id"`
	UserID    int       `json:"user_id"`
	Name      string    `json:"name"`
	Color     string    `json:"color"`
	CreatedAt time.Time `json:"created_at"`
}

// CreateSprintRequest represents a request to create a sprint
type CreateSprintRequest struct {
	Name      string `json:"name"`
	Goal      string `json:"goal,omitempty"`
	StartDate string `json:"start_date,omitempty"`
	EndDate   string `json:"end_date,omitempty"`
	Status    string `json:"status,omitempty"`
}

// UpdateSprintRequest represents a request to update a sprint
type UpdateSprintRequest struct {
	Name      *string `json:"name,omitempty"`
	Goal      *string `json:"goal,omitempty"`
	StartDate *string `json:"start_date,omitempty"`
	EndDate   *string `json:"end_date,omitempty"`
	Status    *string `json:"status,omitempty"`
}

// CreateTagRequest represents a request to create a tag
type CreateTagRequest struct {
	Name  string `json:"name"`
	Color string `json:"color,omitempty"`
}

// UpdateTagRequest represents a request to update a tag
type UpdateTagRequest struct {
	Name  *string `json:"name,omitempty"`
	Color *string `json:"color,omitempty"`
}

// HandleListSprints returns all sprints for the current user's team
func (s *Server) HandleListSprints(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	// Query sprints that belong to the user's team
	// We'll sort by status priority in Go after fetching
	entSprints, err := s.db.Client.Sprint.Query().
		Where(sprint.TeamID(teamID)).
		Order(ent.Desc(sprint.FieldStartDate)).
		Order(ent.Desc(sprint.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch sprints", http.StatusInternalServerError)
		return
	}

	sprints := make([]Sprint, 0, len(entSprints))
	for _, es := range entSprints {
		sp := Sprint{
			ID:        int(es.ID),
			UserID:    int(es.UserID),
			Name:      es.Name,
			Status:    es.Status,
			CreatedAt: es.CreatedAt,
			UpdatedAt: es.UpdatedAt,
		}

		if es.Goal != nil {
			sp.Goal = *es.Goal
		}
		if es.StartDate != nil {
			sp.StartDate = es.StartDate.Format("2006-01-02")
		}
		if es.EndDate != nil {
			sp.EndDate = es.EndDate.Format("2006-01-02")
		}

		sprints = append(sprints, sp)
	}

	respondJSON(w, http.StatusOK, sprints)
}

// HandleCreateSprint creates a new sprint
func (s *Server) HandleCreateSprint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	var req CreateSprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Sprint name is required", http.StatusBadRequest)
		return
	}

	status := req.Status
	if status == "" {
		status = "planned"
	}

	// Validate status
	if status != "planned" && status != "active" && status != "completed" {
		http.Error(w, "Invalid status. Must be planned, active, or completed", http.StatusBadRequest)
		return
	}

	// Create sprint using Ent
	builder := s.db.Client.Sprint.Create().
		SetUserID(userID).
		SetTeamID(teamID).
		SetName(req.Name).
		SetStatus(status)

	if req.Goal != "" {
		builder.SetGoal(req.Goal)
	}

	// Parse dates if provided
	if req.StartDate != "" {
		startDate, err := time.Parse("2006-01-02", req.StartDate)
		if err == nil {
			builder.SetStartDate(startDate)
		}
	}
	if req.EndDate != "" {
		endDate, err := time.Parse("2006-01-02", req.EndDate)
		if err == nil {
			builder.SetEndDate(endDate)
		}
	}

	newSprint, err := builder.Save(ctx)
	if err != nil {
		http.Error(w, "Failed to create sprint", http.StatusInternalServerError)
		return
	}

	// Convert to API sprint
	sprint := Sprint{
		ID:        int(newSprint.ID),
		UserID:    int(newSprint.UserID),
		Name:      newSprint.Name,
		Status:    newSprint.Status,
		CreatedAt: newSprint.CreatedAt,
		UpdatedAt: newSprint.UpdatedAt,
	}

	if newSprint.Goal != nil {
		sprint.Goal = *newSprint.Goal
	}
	if newSprint.StartDate != nil {
		sprint.StartDate = newSprint.StartDate.Format("2006-01-02")
	}
	if newSprint.EndDate != nil {
		sprint.EndDate = newSprint.EndDate.Format("2006-01-02")
	}

	respondJSON(w, http.StatusCreated, sprint)
}

// HandleUpdateSprint updates a sprint
func (s *Server) HandleUpdateSprint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sprintID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid sprint ID", http.StatusBadRequest)
		return
	}

	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	// Check if sprint belongs to user's team
	sprintEntity, err := s.db.Client.Sprint.Get(ctx, sprintID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Sprint not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sprintEntity.TeamID == nil || *sprintEntity.TeamID != teamID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req UpdateSprintRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build update using Ent
	updateBuilder := s.db.Client.Sprint.UpdateOneID(sprintID)
	hasUpdates := false

	if req.Name != nil {
		updateBuilder.SetName(*req.Name)
		hasUpdates = true
	}
	if req.Goal != nil {
		updateBuilder.SetGoal(*req.Goal)
		hasUpdates = true
	}
	if req.StartDate != nil {
		if *req.StartDate != "" {
			startDate, err := time.Parse("2006-01-02", *req.StartDate)
			if err == nil {
				updateBuilder.SetStartDate(startDate)
				hasUpdates = true
			}
		}
	}
	if req.EndDate != nil {
		if *req.EndDate != "" {
			endDate, err := time.Parse("2006-01-02", *req.EndDate)
			if err == nil {
				updateBuilder.SetEndDate(endDate)
				hasUpdates = true
			}
		}
	}
	if req.Status != nil {
		// Validate status
		if *req.Status != "planned" && *req.Status != "active" && *req.Status != "completed" {
			http.Error(w, "Invalid status", http.StatusBadRequest)
			return
		}
		updateBuilder.SetStatus(*req.Status)
		hasUpdates = true
	}

	if !hasUpdates {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	updatedSprint, err := updateBuilder.Save(ctx)
	if err != nil {
		http.Error(w, "Failed to update sprint", http.StatusInternalServerError)
		return
	}

	// Convert to API sprint
	sprint := Sprint{
		ID:        int(updatedSprint.ID),
		UserID:    int(updatedSprint.UserID),
		Name:      updatedSprint.Name,
		Status:    updatedSprint.Status,
		CreatedAt: updatedSprint.CreatedAt,
		UpdatedAt: updatedSprint.UpdatedAt,
	}

	if updatedSprint.Goal != nil {
		sprint.Goal = *updatedSprint.Goal
	}
	if updatedSprint.StartDate != nil {
		sprint.StartDate = updatedSprint.StartDate.Format("2006-01-02")
	}
	if updatedSprint.EndDate != nil {
		sprint.EndDate = updatedSprint.EndDate.Format("2006-01-02")
	}

	respondJSON(w, http.StatusOK, sprint)
}

// HandleDeleteSprint deletes a sprint
func (s *Server) HandleDeleteSprint(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	sprintID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid sprint ID", http.StatusBadRequest)
		return
	}

	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	// Check if sprint belongs to user's team
	sprintEntity, err := s.db.Client.Sprint.Get(ctx, sprintID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Sprint not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if sprintEntity.TeamID == nil || *sprintEntity.TeamID != teamID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	err = s.db.Client.Sprint.DeleteOneID(sprintID).Exec(ctx)
	if err != nil {
		http.Error(w, "Failed to delete sprint", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Sprint deleted successfully"})
}

// HandleListTags returns all tags for the current user's team
func (s *Server) HandleListTags(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	entTags, err := s.db.Client.Tag.Query().
		Where(tag.TeamID(teamID)).
		Order(ent.Asc(tag.FieldName)).
		All(ctx)
	if err != nil {
		http.Error(w, "Failed to fetch tags", http.StatusInternalServerError)
		return
	}

	tags := make([]Tag, 0, len(entTags))
	for _, et := range entTags {
		tags = append(tags, Tag{
			ID:        int(et.ID),
			UserID:    int(et.UserID),
			Name:      et.Name,
			Color:     et.Color,
			CreatedAt: et.CreatedAt,
		})
	}

	respondJSON(w, http.StatusOK, tags)
}

// HandleCreateTag creates a new tag
func (s *Server) HandleCreateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	var req CreateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Tag name is required", http.StatusBadRequest)
		return
	}

	color := req.Color
	if color == "" {
		color = "#3B82F6"
	}

	newTag, err := s.db.Client.Tag.Create().
		SetUserID(userID).
		SetTeamID(teamID).
		SetName(req.Name).
		SetColor(color).
		Save(ctx)

	if err != nil {
		if ent.IsConstraintError(err) {
			http.Error(w, "Failed to create tag. Tag name must be unique.", http.StatusConflict)
			return
		}
		http.Error(w, "Failed to create tag", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusCreated, Tag{
		ID:        int(newTag.ID),
		UserID:    int(newTag.UserID),
		Name:      newTag.Name,
		Color:     newTag.Color,
		CreatedAt: newTag.CreatedAt,
	})
}

// HandleUpdateTag updates a tag
func (s *Server) HandleUpdateTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tagID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	// Check if tag belongs to user's team
	tagEntity, err := s.db.Client.Tag.Get(ctx, tagID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Tag not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if tagEntity.TeamID == nil || *tagEntity.TeamID != teamID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req UpdateTagRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build update using Ent
	updateBuilder := s.db.Client.Tag.UpdateOneID(tagID)
	hasUpdates := false

	if req.Name != nil {
		updateBuilder.SetName(*req.Name)
		hasUpdates = true
	}
	if req.Color != nil {
		updateBuilder.SetColor(*req.Color)
		hasUpdates = true
	}

	if !hasUpdates {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	updatedTag, err := updateBuilder.Save(ctx)
	if err != nil {
		http.Error(w, "Failed to update tag", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, Tag{
		ID:        int(updatedTag.ID),
		UserID:    int(updatedTag.UserID),
		Name:      updatedTag.Name,
		Color:     updatedTag.Color,
		CreatedAt: updatedTag.CreatedAt,
	})
}

// HandleDeleteTag deletes a tag
func (s *Server) HandleDeleteTag(w http.ResponseWriter, r *http.Request) {
	ctx := r.Context()
	tagID, err := strconv.ParseInt(chi.URLParam(r, "id"), 10, 64)
	if err != nil {
		http.Error(w, "Invalid tag ID", http.StatusBadRequest)
		return
	}

	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	// Get user's team ID
	teamID, err := s.getUserTeamID(ctx, userID)
	if err != nil {
		http.Error(w, "Failed to get user team", http.StatusInternalServerError)
		return
	}

	// Check if tag belongs to user's team
	tagEntity, err := s.db.Client.Tag.Get(ctx, tagID)
	if err != nil {
		if ent.IsNotFound(err) {
			http.Error(w, "Tag not found", http.StatusNotFound)
			return
		}
		http.Error(w, "Internal server error", http.StatusInternalServerError)
		return
	}
	if tagEntity.TeamID == nil || *tagEntity.TeamID != teamID {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	err = s.db.Client.Tag.DeleteOneID(tagID).Exec(ctx)
	if err != nil {
		http.Error(w, "Failed to delete tag", http.StatusInternalServerError)
		return
	}

	respondJSON(w, http.StatusOK, map[string]string{"message": "Tag deleted successfully"})
}
