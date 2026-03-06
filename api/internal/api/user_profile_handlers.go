package api

import (
	"context"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

// UserProfileInfo holds basic user info for a profile page.
type UserProfileInfo struct {
	ID        int64     `json:"id"`
	Name      *string   `json:"name,omitempty"`
	FirstName *string   `json:"first_name,omitempty"`
	LastName  *string   `json:"last_name,omitempty"`
	Email     string    `json:"email"`
	JoinedAt  time.Time `json:"joined_at"`
}

// UserActivityItem represents a single activity entry.
type UserActivityItem struct {
	Type        string    `json:"type"`
	EntityID    int64     `json:"entity_id"`
	EntityTitle string    `json:"entity_title"`
	ProjectID   int64     `json:"project_id"`
	ProjectName string    `json:"project_name"`
	Link        string    `json:"link"`
	CreatedAt   time.Time `json:"created_at"`
}

// UserProfileResponse is the full user profile payload.
type UserProfileResponse struct {
	User           UserProfileInfo    `json:"user"`
	RecentActivity []UserActivityItem `json:"recent_activity"`
}

// HandleGetUserProfile returns a user's public profile with recent activity.
// Only accessible to members of shared projects.
func (s *Server) HandleGetUserProfile(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	viewerID := r.Context().Value(UserIDKey).(int64)
	targetUserID, err := strconv.ParseInt(chi.URLParam(r, "userId"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid user ID", "invalid_input")
		return
	}

	// Viewer can always view their own profile; otherwise check shared project
	if viewerID != targetUserID {
		var sharedCount int
		if err := s.db.QueryRowContext(ctx, `
			SELECT COUNT(*) FROM project_members pm1
			JOIN project_members pm2 ON pm2.project_id = pm1.project_id AND pm2.user_id = $2
			WHERE pm1.user_id = $1
		`, viewerID, targetUserID).Scan(&sharedCount); err != nil || sharedCount == 0 {
			respondError(w, http.StatusForbidden, "access denied", "forbidden")
			return
		}
	}

	// Fetch user info including created_at
	var u UserProfileInfo
	if err := s.db.QueryRowContext(ctx, `
		SELECT id, name, first_name, last_name, email, created_at FROM users WHERE id = $1 AND deleted_at IS NULL
	`, targetUserID).Scan(&u.ID, &u.Name, &u.FirstName, &u.LastName, &u.Email, &u.JoinedAt); err != nil {
		respondError(w, http.StatusNotFound, "user not found", "not_found")
		return
	}

	// Collect recent activity (scoped to projects shared between viewer and target)
	activity, err := s.fetchUserActivity(ctx, viewerID, targetUserID)
	if err != nil {
		s.logger.Error("Failed to fetch user activity", zap.Error(err), zap.Int64("target", targetUserID))
		activity = []UserActivityItem{}
	}

	respondJSON(w, http.StatusOK, UserProfileResponse{
		User:           u,
		RecentActivity: activity,
	})
}

// sharedProjectsSubquery returns a SQL fragment for projects shared between viewer and target.
// Both must be current members. Used as: WHERE t.project_id IN (<subquery>)
const sharedProjectsSubquery = `
	SELECT pm1.project_id
	FROM project_members pm1
	JOIN project_members pm2 ON pm2.project_id = pm1.project_id AND pm2.user_id = $2
	WHERE pm1.user_id = $1
`

// fetchUserActivity returns recent activity for a user scoped to projects
// shared between viewer and target (both must currently be members).
func (s *Server) fetchUserActivity(ctx context.Context, viewerID, targetUserID int64) ([]UserActivityItem, error) {
	items := []UserActivityItem{}

	// Task comments
	rows, err := s.db.QueryContext(ctx, `
		SELECT tc.id, t.title, t.project_id, p.name, t.task_number, tc.created_at
		FROM task_comments tc
		JOIN tasks t ON t.id = tc.task_id
		JOIN projects p ON p.id = t.project_id
		WHERE tc.user_id = $2
		  AND t.project_id IN (`+sharedProjectsSubquery+`)
		ORDER BY tc.created_at DESC
		LIMIT 20
	`, viewerID, targetUserID)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var item UserActivityItem
			var taskNumber int64
			if rows.Scan(&item.EntityID, &item.EntityTitle, &item.ProjectID, &item.ProjectName, &taskNumber, &item.CreatedAt) == nil {
				item.Type = "task_comment"
				item.Link = "/app/projects/" + int64ToStr(item.ProjectID) + "/tasks/" + int64ToStr(taskNumber)
				items = append(items, item)
			}
		}
	}

	// Wiki pages created
	wikiRows, err := s.db.QueryContext(ctx, `
		SELECT wp.id, wp.title, wp.project_id, p.name, wp.created_at
		FROM wiki_pages wp
		JOIN projects p ON p.id = wp.project_id
		WHERE wp.created_by = $2
		  AND wp.project_id IN (`+sharedProjectsSubquery+`)
		ORDER BY wp.created_at DESC
		LIMIT 20
	`, viewerID, targetUserID)
	if err == nil {
		defer wikiRows.Close()
		for wikiRows.Next() {
			var item UserActivityItem
			if wikiRows.Scan(&item.EntityID, &item.EntityTitle, &item.ProjectID, &item.ProjectName, &item.CreatedAt) == nil {
				item.Type = "wiki_page"
				item.Link = "/app/projects/" + int64ToStr(item.ProjectID) + "/wiki?page=" + int64ToStr(item.EntityID)
				items = append(items, item)
			}
		}
	}

	// Annotation comments
	annRows, err := s.db.QueryContext(ctx, `
		SELECT wac.id, wp.title, wp.project_id, p.name, wp.id, wa.id, wac.created_at
		FROM wiki_annotation_comments wac
		JOIN wiki_annotations wa ON wa.id = wac.annotation_id
		JOIN wiki_pages wp ON wp.id = wa.wiki_page_id
		JOIN projects p ON p.id = wp.project_id
		WHERE wac.author_id = $2
		  AND wp.project_id IN (`+sharedProjectsSubquery+`)
		ORDER BY wac.created_at DESC
		LIMIT 20
	`, viewerID, targetUserID)
	if err == nil {
		defer annRows.Close()
		for annRows.Next() {
			var item UserActivityItem
			var pageID, annotationID int64
			if annRows.Scan(&item.EntityID, &item.EntityTitle, &item.ProjectID, &item.ProjectName, &pageID, &annotationID, &item.CreatedAt) == nil {
				item.Type = "annotation_comment"
				item.Link = "/app/projects/" + int64ToStr(item.ProjectID) + "/wiki?page=" + int64ToStr(pageID) + "&annotation=" + int64ToStr(annotationID)
				items = append(items, item)
			}
		}
	}

	// Tasks created
	taskCreatedRows, err := s.db.QueryContext(ctx, `
		SELECT t.id, t.title, t.project_id, p.name, t.task_number, t.created_at
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE t.created_by = $2
		  AND t.project_id IN (`+sharedProjectsSubquery+`)
		ORDER BY t.created_at DESC
		LIMIT 20
	`, viewerID, targetUserID)
	if err == nil {
		defer taskCreatedRows.Close()
		for taskCreatedRows.Next() {
			var item UserActivityItem
			var taskNumber int64
			if taskCreatedRows.Scan(&item.EntityID, &item.EntityTitle, &item.ProjectID, &item.ProjectName, &taskNumber, &item.CreatedAt) == nil {
				item.Type = "task_created"
				item.Link = "/app/projects/" + int64ToStr(item.ProjectID) + "/tasks/" + int64ToStr(taskNumber)
				items = append(items, item)
			}
		}
	}

	// Wiki annotations created
	waRows, err := s.db.QueryContext(ctx, `
		SELECT wa.id, wp.title, wp.project_id, p.name, wp.id, wa.created_at
		FROM wiki_annotations wa
		JOIN wiki_pages wp ON wp.id = wa.wiki_page_id
		JOIN projects p ON p.id = wp.project_id
		WHERE wa.author_id = $2
		  AND wp.project_id IN (`+sharedProjectsSubquery+`)
		ORDER BY wa.created_at DESC
		LIMIT 20
	`, viewerID, targetUserID)
	if err == nil {
		defer waRows.Close()
		for waRows.Next() {
			var item UserActivityItem
			var pageID int64
			if waRows.Scan(&item.EntityID, &item.EntityTitle, &item.ProjectID, &item.ProjectName, &pageID, &item.CreatedAt) == nil {
				item.Type = "annotation_created"
				item.Link = "/app/projects/" + int64ToStr(item.ProjectID) + "/wiki?page=" + int64ToStr(pageID)
				items = append(items, item)
			}
		}
	}

	// Wiki edits (most recent edit per page)
	weRows, err := s.db.QueryContext(ctx, `
		SELECT yu.page_id, wp.title, wp.project_id, p.name, MAX(yu.created_at)
		FROM yjs_updates yu
		JOIN wiki_pages wp ON wp.id = yu.page_id
		JOIN projects p ON p.id = wp.project_id
		WHERE yu.created_by = $2
		  AND wp.project_id IN (`+sharedProjectsSubquery+`)
		GROUP BY yu.page_id, wp.title, wp.project_id, p.name
		ORDER BY MAX(yu.created_at) DESC
		LIMIT 20
	`, viewerID, targetUserID)
	if err == nil {
		defer weRows.Close()
		for weRows.Next() {
			var item UserActivityItem
			if weRows.Scan(&item.EntityID, &item.EntityTitle, &item.ProjectID, &item.ProjectName, &item.CreatedAt) == nil {
				item.Type = "wiki_edit"
				item.Link = "/app/projects/" + int64ToStr(item.ProjectID) + "/wiki?page=" + int64ToStr(item.EntityID)
				items = append(items, item)
			}
		}
	}

	// Sort by created_at descending and take top 50
	sortActivityItems(items)
	if len(items) > 50 {
		items = items[:50]
	}
	return items, nil
}

// sortActivityItems sorts by CreatedAt descending.
func sortActivityItems(items []UserActivityItem) {
	for i := 1; i < len(items); i++ {
		for j := i; j > 0 && items[j].CreatedAt.After(items[j-1].CreatedAt); j-- {
			items[j], items[j-1] = items[j-1], items[j]
		}
	}
}
