package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// --- GitHub API response types ---

type ghMilestone struct {
	Number int    `json:"number"`
	Title  string `json:"title"`
	State  string `json:"state"`
	DueOn  string `json:"due_on"`
}

type ghLabel struct {
	Name  string `json:"name"`
	Color string `json:"color"`
}

type ghUser struct {
	Login string `json:"login"`
	Name  string `json:"name"`
}

type ghIssue struct {
	Number      int          `json:"number"`
	Title       string       `json:"title"`
	Body        string       `json:"body"`
	State       string       `json:"state"`
	Assignee    *ghUser      `json:"assignee"`
	Assignees   []ghUser     `json:"assignees"`
	Labels      []ghLabel    `json:"labels"`
	Milestone   *ghMilestone `json:"milestone"`
	PullRequest *struct{}    `json:"pull_request"` // non-nil means this is a PR, not an issue
}

// --- GitHub Projects V2 GraphQL types ---

type ghProjectV2 struct {
	ID     string `json:"id"`
	Title  string `json:"title"`
	Fields struct {
		Nodes []ghProjectField `json:"nodes"`
	} `json:"fields"`
}

type ghProjectField struct {
	ID      string            `json:"id"`
	Name    string            `json:"name"`
	Options []ghProjectOption `json:"options"` // only present for single-select fields
}

type ghProjectOption struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type ghProjectItem struct {
	Content     *ghProjectItemContent `json:"content"`
	FieldValues struct {
		Nodes []ghProjectFieldValue `json:"nodes"`
	} `json:"fieldValues"`
}

type ghProjectItemContent struct {
	Number int `json:"number"` // issue number
}

type ghProjectFieldValue struct {
	Name  string `json:"name"`  // selected option name (for single-select fields)
	Field struct {
		Name string `json:"name"` // field name (e.g. "Status")
	} `json:"field"`
}

// --- Request / Response types for our handlers ---

// GitHubPreviewRequest allows overriding the stored token for a preview fetch.
type GitHubPreviewRequest struct {
	Token string `json:"token"`
}

// GitHubUserMatch represents a GitHub user with optional TaskAI mapping.
type GitHubUserMatch struct {
	Login         string  `json:"login"`
	Name          string  `json:"name"`
	MatchedUserID *int64  `json:"matched_user_id"`
	MatchedName   string  `json:"matched_name"`
}

// GitHubColumnMatch represents a GitHub Projects V2 status column with optional swim lane mapping.
type GitHubColumnMatch struct {
	Name          string `json:"name"`
	MatchedLaneID *int64 `json:"matched_lane_id"`
	MatchedName   string `json:"matched_name"`
}

// GitHubPreviewResponse is returned by HandleGitHubPreview.
type GitHubPreviewResponse struct {
	MilestoneCount int                 `json:"milestone_count"`
	LabelCount     int                 `json:"label_count"`
	IssueCount     int                 `json:"issue_count"`
	GitHubUsers    []GitHubUserMatch   `json:"github_users"`
	ProjectColumns []GitHubColumnMatch `json:"project_columns"` // GitHub Projects V2 status columns
}

// GitHubPullRequest is the body for HandleGitHubPull / HandleGitHubSync.
type GitHubPullRequest struct {
	Token             string           `json:"token"`
	PullSprints       bool             `json:"pull_sprints"`
	PullTags          bool             `json:"pull_tags"`
	PullTasks         bool             `json:"pull_tasks"`
	UserAssignments   map[string]int64 `json:"user_assignments"`   // login → TaskAI user_id (0 = unassigned)
	ColumnAssignments map[string]int64 `json:"column_assignments"` // GitHub column name → TaskAI swim_lane_id (0 = use default)
}

// GitHubPullResponse is returned by HandleGitHubPull / HandleGitHubSync.
type GitHubPullResponse struct {
	CreatedSprints int `json:"created_sprints"`
	CreatedTags    int `json:"created_tags"`
	CreatedTasks   int `json:"created_tasks"`
	SkippedTasks   int `json:"skipped_tasks"`
}

// --- Helper ---

func fetchGitHubJSON(ctx context.Context, token, url string, dest interface{}) error {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/vnd.github+json")
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github api error %d: %s", resp.StatusCode, strings.TrimSpace(string(body)))
	}

	return json.NewDecoder(resp.Body).Decode(dest)
}

// fetchGitHubGraphQL sends a GraphQL query to the GitHub API.
func fetchGitHubGraphQL(ctx context.Context, token, query string, variables map[string]interface{}, dest interface{}) error {
	payload := map[string]interface{}{"query": query}
	if variables != nil {
		payload["variables"] = variables
	}
	body, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, "https://api.github.com/graphql", bytes.NewReader(body))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+token)
	req.Header.Set("X-GitHub-Api-Version", "2022-11-28")

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		b, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("github graphql error %d: %s", resp.StatusCode, strings.TrimSpace(string(b)))
	}
	return json.NewDecoder(resp.Body).Decode(dest)
}

// fetchProjectStatusColumns fetches GitHub Projects V2 status columns for a repo.
// Returns columns from the first project found, or nil if none.
func fetchProjectStatusColumns(ctx context.Context, token, owner, repo string) ([]ghProjectOption, string, error) {
	if token == "" {
		return nil, "", nil
	}
	const q = `
query($owner: String!, $repo: String!) {
  repository(owner: $owner, name: $repo) {
    projectsV2(first: 5) {
      nodes {
        id
        title
        fields(first: 20) {
          nodes {
            ... on ProjectV2SingleSelectField {
              id
              name
              options { id name }
            }
          }
        }
      }
    }
  }
}`
	var result struct {
		Data struct {
			Repository struct {
				ProjectsV2 struct {
					Nodes []ghProjectV2 `json:"nodes"`
				} `json:"projectsV2"`
			} `json:"repository"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}
	if err := fetchGitHubGraphQL(ctx, token, q, map[string]interface{}{"owner": owner, "repo": repo}, &result); err != nil {
		return nil, "", err
	}
	if len(result.Errors) > 0 {
		return nil, "", fmt.Errorf("graphql: %s", result.Errors[0].Message)
	}
	for _, proj := range result.Data.Repository.ProjectsV2.Nodes {
		for _, field := range proj.Fields.Nodes {
			if strings.EqualFold(field.Name, "status") && len(field.Options) > 0 {
				return field.Options, proj.ID, nil
			}
		}
	}
	return nil, "", nil
}

// fetchProjectIssueStatuses builds a map of issue_number → status column name
// by paginating through all items of the given project.
func fetchProjectIssueStatuses(ctx context.Context, token, projectID string) (map[int]string, error) {
	result := map[int]string{}
	if token == "" || projectID == "" {
		return result, nil
	}
	const q = `
query($projectId: ID!, $cursor: String) {
  node(id: $projectId) {
    ... on ProjectV2 {
      items(first: 100, after: $cursor) {
        pageInfo { hasNextPage endCursor }
        nodes {
          content {
            ... on Issue { number }
          }
          fieldValues(first: 10) {
            nodes {
              ... on ProjectV2ItemFieldSingleSelectValue {
                name
                field { ... on ProjectV2SingleSelectField { name } }
              }
            }
          }
        }
      }
    }
  }
}`
	type pageResult struct {
		Data struct {
			Node struct {
				Items struct {
					PageInfo struct {
						HasNextPage bool   `json:"hasNextPage"`
						EndCursor   string `json:"endCursor"`
					} `json:"pageInfo"`
					Nodes []ghProjectItem `json:"nodes"`
				} `json:"items"`
			} `json:"node"`
		} `json:"data"`
		Errors []struct {
			Message string `json:"message"`
		} `json:"errors"`
	}

	var cursor *string
	for page := 0; page < 20; page++ {
		vars := map[string]interface{}{"projectId": projectID}
		if cursor != nil {
			vars["cursor"] = *cursor
		}
		var pr pageResult
		if err := fetchGitHubGraphQL(ctx, token, q, vars, &pr); err != nil {
			return result, err
		}
		if len(pr.Errors) > 0 {
			return result, fmt.Errorf("graphql: %s", pr.Errors[0].Message)
		}
		for _, item := range pr.Data.Node.Items.Nodes {
			if item.Content == nil || item.Content.Number == 0 {
				continue
			}
			for _, fv := range item.FieldValues.Nodes {
				if strings.EqualFold(fv.Field.Name, "status") && fv.Name != "" {
					result[item.Content.Number] = fv.Name
					break
				}
			}
		}
		if !pr.Data.Node.Items.PageInfo.HasNextPage {
			break
		}
		c := pr.Data.Node.Items.PageInfo.EndCursor
		cursor = &c
	}
	return result, nil
}

// fuzzyMatchColumn fuzzy-matches a GitHub column name to the best swim lane.
// Returns the swim_lane_id or 0 if no match found.
func fuzzyMatchColumn(columnName string, lanes []swimLaneInfo) (int64, string) {
	col := strings.ToLower(strings.TrimSpace(columnName))
	// Exact match first
	for _, l := range lanes {
		if strings.ToLower(l.Name) == col {
			return l.ID, l.Name
		}
	}
	// Substring: column name contains lane name or vice versa
	for _, l := range lanes {
		lname := strings.ToLower(l.Name)
		if strings.Contains(col, lname) || strings.Contains(lname, col) {
			return l.ID, l.Name
		}
	}
	// Keyword-based category match
	statusCat := ""
	switch {
	case strings.Contains(col, "todo") || strings.Contains(col, "to do") || col == "backlog" || strings.Contains(col, "triage"):
		statusCat = "todo"
	case strings.Contains(col, "progress") || strings.Contains(col, "doing") || strings.Contains(col, "review") || strings.Contains(col, "test") || strings.Contains(col, "qa"):
		statusCat = "in_progress"
	case strings.Contains(col, "done") || strings.Contains(col, "complete") || strings.Contains(col, "finish") || strings.Contains(col, "closed") || strings.Contains(col, "ship") || strings.Contains(col, "release"):
		statusCat = "done"
	}
	if statusCat != "" {
		for _, l := range lanes {
			if l.StatusCategory == statusCat {
				return l.ID, l.Name
			}
		}
	}
	return 0, ""
}

type swimLaneInfo struct {
	ID             int64
	Name           string
	StatusCategory string
}

// loadSwimLaneInfos loads all swim lanes for a project as swimLaneInfo.
func (s *Server) loadSwimLaneInfos(ctx context.Context, projectID int) ([]swimLaneInfo, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, name, status_category FROM swim_lanes WHERE project_id = $1 ORDER BY position ASC
	`, projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var lanes []swimLaneInfo
	for rows.Next() {
		var l swimLaneInfo
		if err := rows.Scan(&l.ID, &l.Name, &l.StatusCategory); err != nil {
			continue
		}
		lanes = append(lanes, l)
	}
	return lanes, nil
}

// loadGitHubConfig loads owner, repo, and token for a project.
func (s *Server) loadGitHubConfig(projectID int) (owner, repo, token string, err error) {
	var tokenNull sql.NullString
	err = s.db.QueryRow(`
		SELECT COALESCE(github_owner,''), COALESCE(github_repo_name,''), github_token
		FROM projects WHERE id = $1
	`, projectID).Scan(&owner, &repo, &tokenNull)
	if tokenNull.Valid {
		token = tokenNull.String
	}
	return
}

// --- Handlers ---

// HandleGitHubPreview fetches GitHub data without importing anything.
// POST /api/projects/{id}/github/preview
func (s *Server) HandleGitHubPreview(w http.ResponseWriter, r *http.Request) {
	projectID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	hasAccess, err := s.userHasProjectAccess(int(userID), projectID)
	if err != nil || !hasAccess {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req GitHubPreviewRequest
	_ = json.NewDecoder(r.Body).Decode(&req)

	owner, repo, storedToken, err := s.loadGitHubConfig(projectID)
	if err != nil {
		http.Error(w, "Failed to load project config", http.StatusInternalServerError)
		return
	}
	if owner == "" || repo == "" {
		respondError(w, http.StatusBadRequest, "GitHub owner and repo name must be configured first", "missing_config")
		return
	}
	token := storedToken
	if req.Token != "" {
		token = req.Token
	}

	ctx := r.Context()
	base := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	// Fetch milestones
	var milestones []ghMilestone
	if err := fetchGitHubJSON(ctx, token, base+"/milestones?state=all&per_page=100", &milestones); err != nil {
		s.logger.Error("Failed to fetch GitHub milestones", zap.Error(err))
		respondError(w, http.StatusBadGateway, "Failed to fetch milestones from GitHub: "+err.Error(), "github_error")
		return
	}

	// Fetch labels
	var labels []ghLabel
	if err := fetchGitHubJSON(ctx, token, base+"/labels?per_page=100", &labels); err != nil {
		s.logger.Error("Failed to fetch GitHub labels", zap.Error(err))
		respondError(w, http.StatusBadGateway, "Failed to fetch labels from GitHub: "+err.Error(), "github_error")
		return
	}

	// Fetch issues (paginate up to 10 pages)
	var allIssues []ghIssue
	for page := 1; page <= 10; page++ {
		var pageIssues []ghIssue
		url := fmt.Sprintf("%s/issues?state=all&per_page=100&page=%d", base, page)
		if err := fetchGitHubJSON(ctx, token, url, &pageIssues); err != nil {
			s.logger.Error("Failed to fetch GitHub issues", zap.Int("page", page), zap.Error(err))
			respondError(w, http.StatusBadGateway, "Failed to fetch issues from GitHub: "+err.Error(), "github_error")
			return
		}
		if len(pageIssues) == 0 {
			break
		}
		allIssues = append(allIssues, pageIssues...)
	}

	// Collect unique assignee logins
	loginSet := map[string]ghUser{}
	for _, issue := range allIssues {
		for _, a := range issue.Assignees {
			if a.Login != "" {
				loginSet[a.Login] = a
			}
		}
		if issue.Assignee != nil && issue.Assignee.Login != "" {
			loginSet[issue.Assignee.Login] = *issue.Assignee
		}
	}

	// Load all team members for auto-matching (not just project members)
	type memberInfo struct {
		UserID    int64
		Email     string
		Name      string
		FirstName string
		LastName  string
	}
	rows, err := s.db.QueryContext(ctx, `
		SELECT DISTINCT u.id, u.email, COALESCE(u.name,''), COALESCE(u.first_name,''), COALESCE(u.last_name,'')
		FROM users u
		JOIN team_members tm ON tm.user_id = u.id
		WHERE tm.team_id = (SELECT team_id FROM projects WHERE id = $1)
	`, projectID)
	if err != nil {
		http.Error(w, "Failed to load team members", http.StatusInternalServerError)
		return
	}
	defer rows.Close()

	var members []memberInfo
	for rows.Next() {
		var m memberInfo
		if err := rows.Scan(&m.UserID, &m.Email, &m.Name, &m.FirstName, &m.LastName); err != nil {
			continue
		}
		members = append(members, m)
	}
	rows.Close()

	// Auto-match GitHub users to TaskAI members
	ghUsers := make([]GitHubUserMatch, 0, len(loginSet))
	for login, ghU := range loginSet {
		match := GitHubUserMatch{Login: login, Name: ghU.Name}
		loginLower := strings.ToLower(login)
		nameLower := strings.ToLower(ghU.Name)
		for _, m := range members {
			emailUser := strings.ToLower(strings.Split(m.Email, "@")[0])
			fullName := strings.ToLower(strings.TrimSpace(m.FirstName + " " + m.LastName))
			if fullName == " " {
				fullName = ""
			}
			nameLowerM := strings.ToLower(m.Name)
			if loginLower == emailUser ||
				(nameLower != "" && (nameLower == nameLowerM || nameLower == fullName)) {
				uid := m.UserID
				match.MatchedUserID = &uid
				if m.FirstName != "" || m.LastName != "" {
					match.MatchedName = strings.TrimSpace(m.FirstName + " " + m.LastName)
				} else {
					match.MatchedName = m.Name
				}
				break
			}
		}
		ghUsers = append(ghUsers, match)
	}

	// Fetch GitHub Projects V2 status columns and fuzzy-match to swim lanes
	var projectColumns []GitHubColumnMatch
	if colOptions, _, err := fetchProjectStatusColumns(ctx, token, owner, repo); err == nil && len(colOptions) > 0 {
		lanes, _ := s.loadSwimLaneInfos(ctx, projectID)
		for _, opt := range colOptions {
			col := GitHubColumnMatch{Name: opt.Name}
			if laneID, laneName := fuzzyMatchColumn(opt.Name, lanes); laneID != 0 {
				col.MatchedLaneID = &laneID
				col.MatchedName = laneName
			}
			projectColumns = append(projectColumns, col)
		}
	}

	respondJSON(w, http.StatusOK, GitHubPreviewResponse{
		MilestoneCount: len(milestones),
		LabelCount:     len(labels),
		IssueCount:     len(allIssues),
		GitHubUsers:    ghUsers,
		ProjectColumns: projectColumns,
	})
}

// HandleGitHubPull imports GitHub data, skipping already-imported items.
// POST /api/projects/{id}/github/pull
func (s *Server) HandleGitHubPull(w http.ResponseWriter, r *http.Request) {
	s.handleGitHubImport(w, r, false)
}

// HandleGitHubSync imports GitHub data, updating already-imported items.
// POST /api/projects/{id}/github/sync
func (s *Server) HandleGitHubSync(w http.ResponseWriter, r *http.Request) {
	s.handleGitHubImport(w, r, true)
}

func (s *Server) handleGitHubImport(w http.ResponseWriter, r *http.Request, doUpdate bool) {
	projectID, err := strconv.Atoi(chi.URLParam(r, "id"))
	if err != nil {
		http.Error(w, "Invalid project ID", http.StatusBadRequest)
		return
	}

	userID, ok := GetUserID(r)
	if !ok {
		http.Error(w, "Unauthorized", http.StatusUnauthorized)
		return
	}

	isOwnerOrAdmin, err := s.userIsProjectOwnerOrAdmin(int(userID), projectID)
	if err != nil || !isOwnerOrAdmin {
		http.Error(w, "Forbidden", http.StatusForbidden)
		return
	}

	var req GitHubPullRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	owner, repo, storedToken, err := s.loadGitHubConfig(projectID)
	if err != nil {
		http.Error(w, "Failed to load project config", http.StatusInternalServerError)
		return
	}
	if owner == "" || repo == "" {
		respondError(w, http.StatusBadRequest, "GitHub owner and repo name must be configured first", "missing_config")
		return
	}

	token := storedToken
	if req.Token != "" {
		// Save the new token
		token = req.Token
		if _, err := s.db.ExecContext(r.Context(), `UPDATE projects SET github_token = $1 WHERE id = $2`, token, projectID); err != nil {
			s.logger.Warn("Failed to save GitHub token", zap.Error(err))
		}
	}

	ctx := r.Context()
	base := fmt.Sprintf("https://api.github.com/repos/%s/%s", owner, repo)

	var result GitHubPullResponse

	// --- Import Sprints from Milestones ---
	if req.PullSprints {
		var milestones []ghMilestone
		if err := fetchGitHubJSON(ctx, token, base+"/milestones?state=all&per_page=100", &milestones); err != nil {
			s.logger.Error("Failed to fetch GitHub milestones", zap.Error(err))
			respondError(w, http.StatusBadGateway, "Failed to fetch milestones: "+err.Error(), "github_error")
			return
		}

		for _, m := range milestones {
			status := "active"
			if m.State == "closed" {
				status = "completed"
			}

			var dueDate *string
			if m.DueOn != "" {
				t, err := time.Parse(time.RFC3339, m.DueOn)
				if err == nil {
					s := t.Format("2006-01-02")
					dueDate = &s
				}
			}

			if doUpdate {
				var existingID int64
				err := s.db.QueryRowContext(ctx, `
					SELECT id FROM sprints WHERE project_id = $1 AND github_milestone_number = $2
				`, projectID, m.Number).Scan(&existingID)

				if err == sql.ErrNoRows {
					// Insert new
					err = s.db.QueryRowContext(ctx, `
						INSERT INTO sprints (user_id, project_id, name, status, end_date, github_milestone_number)
						VALUES ($1, $2, $3, $4, $5, $6)
						ON CONFLICT (project_id, github_milestone_number) DO NOTHING
						RETURNING id
					`, userID, projectID, m.Title, status, dueDate, m.Number).Scan(&existingID)
					if err == nil {
						result.CreatedSprints++
					}
				} else if err == nil {
					// Update existing
					_, _ = s.db.ExecContext(ctx, `
						UPDATE sprints SET name = $1, status = $2, end_date = $3 WHERE id = $4
					`, m.Title, status, dueDate, existingID)
				}
			} else {
				var newID int64
				err := s.db.QueryRowContext(ctx, `
					INSERT INTO sprints (user_id, project_id, name, status, end_date, github_milestone_number)
					VALUES ($1, $2, $3, $4, $5, $6)
					ON CONFLICT (project_id, github_milestone_number) DO NOTHING
					RETURNING id
				`, userID, projectID, m.Title, status, dueDate, m.Number).Scan(&newID)
				if err == nil {
					result.CreatedSprints++
				}
			}
		}
	}

	// --- Import Tags from Labels ---
	if req.PullTags {
		var labels []ghLabel
		if err := fetchGitHubJSON(ctx, token, base+"/labels?per_page=100", &labels); err != nil {
			s.logger.Error("Failed to fetch GitHub labels", zap.Error(err))
			respondError(w, http.StatusBadGateway, "Failed to fetch labels: "+err.Error(), "github_error")
			return
		}

		for _, l := range labels {
			color := "#" + l.Color
			if l.Color == "" {
				color = "#6B7280"
			}

			if doUpdate {
				var existingID int64
				err := s.db.QueryRowContext(ctx, `
					SELECT id FROM tags WHERE project_id = $1 AND github_label_name = $2
				`, projectID, l.Name).Scan(&existingID)

				if err == sql.ErrNoRows {
					err = s.db.QueryRowContext(ctx, `
						INSERT INTO tags (user_id, project_id, name, color, github_label_name)
						VALUES ($1, $2, $3, $4, $5)
						ON CONFLICT (project_id, github_label_name) DO NOTHING
						RETURNING id
					`, userID, projectID, l.Name, color, l.Name).Scan(&existingID)
					if err == nil {
						result.CreatedTags++
					}
				} else if err == nil {
					_, _ = s.db.ExecContext(ctx, `UPDATE tags SET color = $1 WHERE id = $2`, color, existingID)
				}
			} else {
				var newID int64
				err := s.db.QueryRowContext(ctx, `
					INSERT INTO tags (user_id, project_id, name, color, github_label_name)
					VALUES ($1, $2, $3, $4, $5)
					ON CONFLICT (project_id, github_label_name) DO NOTHING
					RETURNING id
				`, userID, projectID, l.Name, color, l.Name).Scan(&newID)
				if err == nil {
					result.CreatedTags++
				}
			}
		}
	}

	// --- Import Tasks from Issues ---
	if req.PullTasks {
		var allIssues []ghIssue
		for page := 1; page <= 10; page++ {
			var pageIssues []ghIssue
			url := fmt.Sprintf("%s/issues?state=all&per_page=100&page=%d", base, page)
			if err := fetchGitHubJSON(ctx, token, url, &pageIssues); err != nil {
				s.logger.Error("Failed to fetch GitHub issues", zap.Int("page", page), zap.Error(err))
				respondError(w, http.StatusBadGateway, "Failed to fetch issues: "+err.Error(), "github_error")
				return
			}
			if len(pageIssues) == 0 {
				break
			}
			allIssues = append(allIssues, pageIssues...)
		}

		// Build a label→tag_id map from newly imported tags
		labelToTagID := map[string]int64{}
		if req.PullTags {
			rows, err := s.db.QueryContext(ctx, `
				SELECT github_label_name, id FROM tags
				WHERE project_id = $1 AND github_label_name IS NOT NULL
			`, projectID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var lname string
					var tid int64
					if err := rows.Scan(&lname, &tid); err == nil {
						labelToTagID[lname] = tid
					}
				}
				rows.Close()
			}
		}

		// Build a milestone_number→sprint_id map
		milestoneToSprintID := map[int]int64{}
		if req.PullSprints {
			rows, err := s.db.QueryContext(ctx, `
				SELECT github_milestone_number, id FROM sprints
				WHERE project_id = $1 AND github_milestone_number IS NOT NULL
			`, projectID)
			if err == nil {
				defer rows.Close()
				for rows.Next() {
					var mnum int
					var sid int64
					if err := rows.Scan(&mnum, &sid); err == nil {
						milestoneToSprintID[mnum] = sid
					}
				}
				rows.Close()
			}
		}

		// Build status_category → swim_lane_id map for this project (fallback)
		swimLaneByCategory := map[string]int64{}
		slRows, err := s.db.QueryContext(ctx, `
			SELECT status_category, id FROM swim_lanes WHERE project_id = $1 ORDER BY position ASC
		`, projectID)
		if err == nil {
			defer slRows.Close()
			for slRows.Next() {
				var cat string
				var slID int64
				if err := slRows.Scan(&cat, &slID); err == nil {
					if _, exists := swimLaneByCategory[cat]; !exists {
						swimLaneByCategory[cat] = slID
					}
				}
			}
			slRows.Close()
		}

		// Fetch GitHub Projects V2 issue→column map (best-effort; ignore errors)
		issueColumnMap := map[int]string{}
		if _, projID, err := fetchProjectStatusColumns(ctx, token, owner, repo); err == nil && projID != "" {
			if m, err := fetchProjectIssueStatuses(ctx, token, projID); err == nil {
				issueColumnMap = m
			}
		}

		// Get next task_number baseline
		var maxNumber sql.NullInt64
		_ = s.db.QueryRowContext(ctx, `SELECT MAX(task_number) FROM tasks WHERE project_id = $1`, projectID).Scan(&maxNumber)
		nextNumber := int64(1)
		if maxNumber.Valid {
			nextNumber = maxNumber.Int64 + 1
		}

		for _, issue := range allIssues {
			taskStatus := "todo"
			if issue.State == "closed" {
				taskStatus = "done"
			}

			// Resolve assignee
			var assigneeID *int64
			primaryLogin := ""
			if issue.Assignee != nil {
				primaryLogin = issue.Assignee.Login
			} else if len(issue.Assignees) > 0 {
				primaryLogin = issue.Assignees[0].Login
			}
			if primaryLogin != "" {
				if uid, ok := req.UserAssignments[primaryLogin]; ok && uid != 0 {
					assigneeID = &uid
				}
			}

			// Resolve sprint
			var sprintID *int64
			if issue.Milestone != nil {
				if sid, ok := milestoneToSprintID[issue.Milestone.Number]; ok {
					sprintID = &sid
				}
			}

			description := issue.Body

			// Resolve swim lane: prefer GitHub Projects V2 column mapping, fall back to status_category
			var swimLaneID *int64
			if colName, ok := issueColumnMap[issue.Number]; ok && colName != "" {
				if laneID := req.ColumnAssignments[colName]; laneID != 0 {
					swimLaneID = &laneID
				}
			}
			if swimLaneID == nil {
				if slID, ok := swimLaneByCategory[taskStatus]; ok {
					swimLaneID = &slID
				}
			}

			if doUpdate {
				var existingID int64
				err := s.db.QueryRowContext(ctx, `
					SELECT id FROM tasks WHERE project_id = $1 AND github_issue_number = $2
				`, projectID, issue.Number).Scan(&existingID)

				if err == sql.ErrNoRows {
					// Insert new task
					err = s.db.QueryRowContext(ctx, `
						INSERT INTO tasks (project_id, task_number, title, description, status, priority, assignee_id, sprint_id, github_issue_number, swim_lane_id)
						VALUES ($1, $2, $3, $4, $5, 'medium', $6, $7, $8, $9)
						ON CONFLICT (project_id, github_issue_number) DO NOTHING
						RETURNING id
					`, projectID, nextNumber, issue.Title, description, taskStatus, assigneeID, sprintID, issue.Number, swimLaneID).Scan(&existingID)
					if err == nil {
						nextNumber++
						result.CreatedTasks++
						s.insertTaskTags(ctx, existingID, issue.Labels, labelToTagID)
					} else {
						result.SkippedTasks++
					}
				} else if err == nil {
					// Update existing
					_, _ = s.db.ExecContext(ctx, `
						UPDATE tasks SET title = $1, description = $2, status = $3, assignee_id = $4, sprint_id = $5, swim_lane_id = $6
						WHERE id = $7
					`, issue.Title, description, taskStatus, assigneeID, sprintID, swimLaneID, existingID)
					// Refresh tags
					_, _ = s.db.ExecContext(ctx, `DELETE FROM task_tags WHERE task_id = $1`, existingID)
					s.insertTaskTags(ctx, existingID, issue.Labels, labelToTagID)
				}
			} else {
				var newID int64
				err := s.db.QueryRowContext(ctx, `
					INSERT INTO tasks (project_id, task_number, title, description, status, priority, assignee_id, sprint_id, github_issue_number, swim_lane_id)
					VALUES ($1, $2, $3, $4, $5, 'medium', $6, $7, $8, $9)
					ON CONFLICT (project_id, github_issue_number) DO NOTHING
					RETURNING id
				`, projectID, nextNumber, issue.Title, description, taskStatus, assigneeID, sprintID, issue.Number, swimLaneID).Scan(&newID)
				if err == nil {
					nextNumber++
					result.CreatedTasks++
					s.insertTaskTags(ctx, newID, issue.Labels, labelToTagID)
				} else {
					result.SkippedTasks++
				}
			}
		}
	}

	// Update last sync timestamp
	_, _ = s.db.ExecContext(ctx, `UPDATE projects SET github_last_sync = $1 WHERE id = $2`, time.Now(), projectID)

	respondJSON(w, http.StatusOK, result)
}

// insertTaskTags inserts tag associations for a task based on issue labels.
func (s *Server) insertTaskTags(ctx context.Context, taskID int64, labels []ghLabel, labelToTagID map[string]int64) {
	for _, lbl := range labels {
		tagID, ok := labelToTagID[lbl.Name]
		if !ok {
			continue
		}
		_, _ = s.db.ExecContext(ctx, `
			INSERT INTO task_tags (task_id, tag_id) VALUES ($1, $2)
			ON CONFLICT DO NOTHING
		`, taskID, tagID)
	}
}
