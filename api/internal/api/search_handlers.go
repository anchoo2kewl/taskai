package api

import (
	"context"
	"encoding/json"
	"net/http"
	"time"

	"go.uber.org/zap"

	"taskai/ent"
	"taskai/ent/project"
	"taskai/ent/task"
	"taskai/ent/wikiblock"
	"taskai/ent/wikipage"
)

// GlobalSearchRequest represents a global search request
type GlobalSearchRequest struct {
	Query     string   `json:"query"`
	ProjectID *int64   `json:"project_id,omitempty"`
	Types     []string `json:"types,omitempty"`
	Limit     int      `json:"limit,omitempty"`
}

// SearchTaskResult represents a task in global search results
type SearchTaskResult struct {
	ID          int64  `json:"id"`
	ProjectID   int64  `json:"project_id"`
	ProjectName string `json:"project_name"`
	TaskNumber  int    `json:"task_number"`
	Title       string `json:"title"`
	Snippet     string `json:"snippet"`
	Status      string `json:"status"`
	Priority    string `json:"priority"`
}

// GlobalSearchWikiResult represents a wiki page in global search results
type GlobalSearchWikiResult struct {
	PageID       int64  `json:"page_id"`
	PageTitle    string `json:"page_title"`
	PageSlug     string `json:"page_slug"`
	ProjectID    int64  `json:"project_id"`
	ProjectName  string `json:"project_name"`
	Snippet      string `json:"snippet"`
	HeadingsPath string `json:"headings_path,omitempty"`
}

// GlobalSearchResponse represents the global search response
type GlobalSearchResponse struct {
	Tasks []SearchTaskResult       `json:"tasks"`
	Wiki  []GlobalSearchWikiResult `json:"wiki"`
}

// HandleGlobalSearch performs search across tasks and wiki pages
func (s *Server) HandleGlobalSearch(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID, ok := ctx.Value(UserIDKey).(int64)
	if !ok {
		respondError(w, http.StatusUnauthorized, "unauthorized", "unauthorized")
		return
	}

	var req GlobalSearchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", "invalid_request")
		return
	}

	if req.Query == "" {
		respondError(w, http.StatusBadRequest, "query parameter is required", "invalid_request")
		return
	}

	// Set defaults
	if req.Limit == 0 {
		req.Limit = 10
	}
	if req.Limit > 50 {
		req.Limit = 50
	}

	// Determine which types to search
	searchTasks := true
	searchWiki := true
	if len(req.Types) > 0 {
		searchTasks = false
		searchWiki = false
		for _, t := range req.Types {
			switch t {
			case "tasks":
				searchTasks = true
			case "wiki":
				searchWiki = true
			}
		}
	}

	s.logger.Debug("Global search request",
		zap.String("query", req.Query),
		zap.Int64("user_id", userID),
		zap.Bool("search_tasks", searchTasks),
		zap.Bool("search_wiki", searchWiki),
	)

	// Get user's accessible project IDs
	accessibleProjects, err := s.getUserAccessibleProjects(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get accessible projects", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to search", "internal_error")
		return
	}

	if len(accessibleProjects) == 0 {
		respondJSON(w, http.StatusOK, GlobalSearchResponse{
			Tasks: []SearchTaskResult{},
			Wiki:  []GlobalSearchWikiResult{},
		})
		return
	}

	// Build project name lookup map
	projectNameMap, err := s.buildProjectNameMap(ctx, accessibleProjects)
	if err != nil {
		s.logger.Error("Failed to load project names", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to search", "internal_error")
		return
	}

	response := GlobalSearchResponse{
		Tasks: []SearchTaskResult{},
		Wiki:  []GlobalSearchWikiResult{},
	}

	// Search tasks
	if searchTasks {
		taskResults, err := s.searchTasks(ctx, req, accessibleProjects, projectNameMap)
		if err != nil {
			s.logger.Error("Failed to search tasks", zap.Error(err), zap.String("query", req.Query))
			respondError(w, http.StatusInternalServerError, "failed to search tasks", "internal_error")
			return
		}
		response.Tasks = taskResults
	}

	// Search wiki
	if searchWiki {
		wikiResults, err := s.searchWikiForGlobal(ctx, req, accessibleProjects, projectNameMap)
		if err != nil {
			s.logger.Error("Failed to search wiki", zap.Error(err), zap.String("query", req.Query))
			respondError(w, http.StatusInternalServerError, "failed to search wiki", "internal_error")
			return
		}
		response.Wiki = wikiResults
	}

	respondJSON(w, http.StatusOK, response)
}

// searchTasks searches for tasks matching the query
func (s *Server) searchTasks(ctx context.Context, req GlobalSearchRequest, accessibleProjects []int64, projectNameMap map[int64]string) ([]SearchTaskResult, error) {
	query := s.db.Client.Task.Query().
		Where(
			task.Or(
				task.TitleContainsFold(req.Query),
				task.DescriptionContainsFold(req.Query),
			),
		)

	// Filter by project
	if req.ProjectID != nil {
		query = query.Where(task.ProjectID(*req.ProjectID))
	} else {
		query = query.Where(task.ProjectIDIn(accessibleProjects...))
	}

	tasks, err := query.
		Limit(req.Limit).
		Order(ent.Desc(task.FieldUpdatedAt)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]SearchTaskResult, 0, len(tasks))
	for _, t := range tasks {
		snippet := ""
		if t.Description != nil {
			snippet = *t.Description
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
		}

		taskNumber := 0
		if t.TaskNumber != nil {
			taskNumber = *t.TaskNumber
		}

		results = append(results, SearchTaskResult{
			ID:          t.ID,
			ProjectID:   t.ProjectID,
			ProjectName: projectNameMap[t.ProjectID],
			TaskNumber:  taskNumber,
			Title:       t.Title,
			Snippet:     snippet,
			Status:      t.Status,
			Priority:    t.Priority,
		})
	}

	return results, nil
}

// searchWikiForGlobal searches wiki blocks and returns results with project info
func (s *Server) searchWikiForGlobal(ctx context.Context, req GlobalSearchRequest, accessibleProjects []int64, projectNameMap map[int64]string) ([]GlobalSearchWikiResult, error) {
	query := s.db.Client.WikiBlock.Query().
		WithPage(func(q *ent.WikiPageQuery) {
			q.Select(wikipage.FieldID, wikipage.FieldTitle, wikipage.FieldSlug, wikipage.FieldProjectID)
		})

	// Filter by project
	if req.ProjectID != nil {
		query = query.Where(wikiblock.HasPageWith(wikipage.ProjectID(*req.ProjectID)))
	} else {
		query = query.Where(wikiblock.HasPageWith(wikipage.ProjectIDIn(accessibleProjects...)))
	}

	// Apply search filter
	query = query.Where(wikiblock.Or(
		wikiblock.PlainTextContains(req.Query),
		wikiblock.HeadingsPathContains(req.Query),
	))

	blocks, err := query.
		Limit(req.Limit).
		All(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]GlobalSearchWikiResult, 0, len(blocks))
	for _, block := range blocks {
		page := block.Edges.Page
		if page == nil {
			continue
		}

		snippet := ""
		if block.PlainText != nil {
			snippet = *block.PlainText
			if len(snippet) > 200 {
				snippet = snippet[:200] + "..."
			}
		}

		headingsPath := ""
		if block.HeadingsPath != nil {
			headingsPath = *block.HeadingsPath
		}

		results = append(results, GlobalSearchWikiResult{
			PageID:       page.ID,
			PageTitle:    page.Title,
			PageSlug:     page.Slug,
			ProjectID:    page.ProjectID,
			ProjectName:  projectNameMap[page.ProjectID],
			Snippet:      snippet,
			HeadingsPath: headingsPath,
		})
	}

	return results, nil
}

// buildProjectNameMap loads project names for the given IDs into a map
func (s *Server) buildProjectNameMap(ctx context.Context, projectIDs []int64) (map[int64]string, error) {
	projects, err := s.db.Client.Project.Query().
		Where(project.IDIn(projectIDs...)).
		All(ctx)
	if err != nil {
		return nil, err
	}

	nameMap := make(map[int64]string, len(projects))
	for _, p := range projects {
		nameMap[p.ID] = p.Name
	}

	return nameMap, nil
}
