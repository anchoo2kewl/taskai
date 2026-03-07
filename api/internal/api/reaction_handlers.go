package api

import (
	"context"
	"database/sql"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap"
)

var validReactions = map[string]bool{
	"+1": true, "-1": true, "laugh": true, "hooray": true,
	"confused": true, "heart": true, "rocket": true, "eyes": true,
}

type ToggleReactionRequest struct {
	Reaction  string `json:"reaction"`
	CommentID int64  `json:"comment_id"`
}

type ToggleReactionResponse struct {
	Reaction    string `json:"reaction"`
	Count       int    `json:"count"`
	UserReacted bool   `json:"user_reacted"`
}

// HandleToggleReaction adds or removes a reaction on a task or task comment.
// POST /api/tasks/{taskId}/reactions
func (s *Server) HandleToggleReaction(w http.ResponseWriter, r *http.Request) {
	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	userID := r.Context().Value(UserIDKey).(int64)
	taskID, err := strconv.ParseInt(chi.URLParam(r, "taskId"), 10, 64)
	if err != nil {
		respondError(w, http.StatusBadRequest, "invalid task ID", "invalid_input")
		return
	}

	var req ToggleReactionRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", "invalid_input")
		return
	}
	if !validReactions[req.Reaction] {
		respondError(w, http.StatusBadRequest, "invalid reaction", "invalid_input")
		return
	}

	// Fetch task and verify access
	var projectID int64
	if err := s.db.QueryRowContext(ctx, `SELECT project_id FROM tasks WHERE id = $1`, taskID).Scan(&projectID); err != nil {
		if err == sql.ErrNoRows {
			respondError(w, http.StatusNotFound, "task not found", "not_found")
			return
		}
		respondError(w, http.StatusInternalServerError, "failed to get task", "internal_error")
		return
	}

	hasAccess, err := s.checkProjectAccess(ctx, userID, projectID)
	if err != nil {
		respondError(w, http.StatusInternalServerError, "failed to verify project access", "internal_error")
		return
	}
	if !hasAccess {
		respondError(w, http.StatusForbidden, "access denied", "forbidden")
		return
	}

	// Validate comment belongs to task (if a comment reaction)
	var commentID int64
	if req.CommentID > 0 {
		commentID = req.CommentID
		var commentTaskID int64
		if err := s.db.QueryRowContext(ctx, `SELECT task_id FROM task_comments WHERE id = $1`, commentID).Scan(&commentTaskID); err != nil {
			if err == sql.ErrNoRows {
				respondError(w, http.StatusNotFound, "comment not found", "not_found")
				return
			}
			respondError(w, http.StatusInternalServerError, "failed to get comment", "internal_error")
			return
		}
		if commentTaskID != taskID {
			respondError(w, http.StatusBadRequest, "comment does not belong to task", "invalid_input")
			return
		}
	}

	// Check existing user reaction
	var existingID int64
	if commentID > 0 {
		_ = s.db.QueryRowContext(ctx,
			`SELECT id FROM user_reactions WHERE user_id = $1 AND task_comment_id = $2 AND reaction = $3`,
			userID, commentID, req.Reaction,
		).Scan(&existingID)
	} else {
		_ = s.db.QueryRowContext(ctx,
			`SELECT id FROM user_reactions WHERE user_id = $1 AND task_id = $2 AND reaction = $3`,
			userID, taskID, req.Reaction,
		).Scan(&existingID)
	}

	var newCount int
	var userReacted bool

	if existingID == 0 {
		// ADD reaction
		if commentID > 0 {
			_, err = s.db.ExecContext(ctx,
				`INSERT INTO user_reactions (user_id, task_comment_id, reaction) VALUES ($1, $2, $3)`,
				userID, commentID, req.Reaction,
			)
		} else {
			_, err = s.db.ExecContext(ctx,
				`INSERT INTO user_reactions (user_id, task_id, reaction) VALUES ($1, $2, $3)`,
				userID, taskID, req.Reaction,
			)
		}
		if err != nil {
			s.logger.Error("Failed to insert user reaction", zap.Int64("user_id", userID), zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to add reaction", "internal_error")
			return
		}

		// Upsert github_reactions count
		if commentID > 0 {
			err = s.db.QueryRowContext(ctx, `
				INSERT INTO github_reactions (task_comment_id, reaction, count)
				VALUES ($1, $2, 1)
				ON CONFLICT (task_comment_id, reaction) WHERE task_comment_id IS NOT NULL
				DO UPDATE SET count = github_reactions.count + 1
				RETURNING count
			`, commentID, req.Reaction).Scan(&newCount)
		} else {
			err = s.db.QueryRowContext(ctx, `
				INSERT INTO github_reactions (task_id, reaction, count)
				VALUES ($1, $2, 1)
				ON CONFLICT (task_id, reaction) WHERE task_id IS NOT NULL
				DO UPDATE SET count = github_reactions.count + 1
				RETURNING count
			`, taskID, req.Reaction).Scan(&newCount)
		}
		if err != nil {
			s.logger.Error("Failed to upsert github_reactions", zap.Int64("task_id", taskID), zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to update reaction count", "internal_error")
			return
		}
		userReacted = true

		// Push to GitHub if this was the first reaction (count just became 1)
		if newCount == 1 {
			go s.tryPushReactionToGitHub(context.Background(), taskID, commentID, req.Reaction, true)
		}
	} else {
		// REMOVE reaction
		if commentID > 0 {
			_, err = s.db.ExecContext(ctx,
				`DELETE FROM user_reactions WHERE user_id = $1 AND task_comment_id = $2 AND reaction = $3`,
				userID, commentID, req.Reaction,
			)
		} else {
			_, err = s.db.ExecContext(ctx,
				`DELETE FROM user_reactions WHERE user_id = $1 AND task_id = $2 AND reaction = $3`,
				userID, taskID, req.Reaction,
			)
		}
		if err != nil {
			s.logger.Error("Failed to delete user reaction", zap.Int64("user_id", userID), zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to remove reaction", "internal_error")
			return
		}

		// Decrement count (CASE expression works in both SQLite and PostgreSQL)
		if commentID > 0 {
			err = s.db.QueryRowContext(ctx, `
				UPDATE github_reactions SET count = CASE WHEN count > 0 THEN count - 1 ELSE 0 END
				WHERE task_comment_id = $1 AND reaction = $2
				RETURNING count
			`, commentID, req.Reaction).Scan(&newCount)
		} else {
			err = s.db.QueryRowContext(ctx, `
				UPDATE github_reactions SET count = CASE WHEN count > 0 THEN count - 1 ELSE 0 END
				WHERE task_id = $1 AND reaction = $2
				RETURNING count
			`, taskID, req.Reaction).Scan(&newCount)
		}
		if err != nil {
			// No row = reaction record didn't exist; treat as 0
			newCount = 0
		}
		userReacted = false

		// Delete from GitHub if count reached zero
		if newCount == 0 {
			go s.tryPushReactionToGitHub(context.Background(), taskID, commentID, req.Reaction, false)
		}
	}

	respondJSON(w, http.StatusOK, ToggleReactionResponse{
		Reaction:    req.Reaction,
		Count:       newCount,
		UserReacted: userReacted,
	})
}

// tryPushReactionToGitHub is a best-effort goroutine to sync a reaction with GitHub.
// add=true → push; add=false → delete.
func (s *Server) tryPushReactionToGitHub(ctx context.Context, taskID, commentID int64, reaction string, add bool) {
	var (
		issueNumber    int64
		ghCommentID    sql.NullInt64
		owner, repo    string
		token          string
		pushEnabled    bool
	)

	err := s.db.QueryRowContext(ctx, `
		SELECT COALESCE(t.github_issue_number,0),
		       COALESCE(p.github_owner,''), COALESCE(p.github_repo_name,''),
		       COALESCE(p.github_token,''), p.github_push_enabled
		FROM tasks t
		JOIN projects p ON p.id = t.project_id
		WHERE t.id = $1
	`, taskID).Scan(&issueNumber, &owner, &repo, &token, &pushEnabled)
	if err != nil || !pushEnabled || owner == "" || token == "" {
		return
	}

	if commentID > 0 {
		// Fetch GitHub comment ID
		_ = s.db.QueryRowContext(ctx,
			`SELECT github_comment_id FROM task_comments WHERE id = $1`,
			commentID,
		).Scan(&ghCommentID)
		if !ghCommentID.Valid {
			return // comment not synced to GitHub
		}

		if add {
			ghReactionID, err := pushReactionToGitHub(ctx, token, owner, repo, ghCommentID.Int64, reaction, "comment")
			if err != nil {
				s.logger.Warn("Failed to push comment reaction to GitHub",
					zap.Int64("comment_id", commentID), zap.String("reaction", reaction), zap.Error(err))
				return
			}
			_, _ = s.db.ExecContext(ctx,
				`UPDATE github_reactions SET github_reaction_id = $1 WHERE task_comment_id = $2 AND reaction = $3`,
				ghReactionID, commentID, reaction,
			)
		} else {
			var ghReactionID sql.NullInt64
			_ = s.db.QueryRowContext(ctx,
				`SELECT github_reaction_id FROM github_reactions WHERE task_comment_id = $1 AND reaction = $2`,
				commentID, reaction,
			).Scan(&ghReactionID)
			if !ghReactionID.Valid {
				return
			}
			if err := deleteReactionFromGitHub(ctx, token, owner, repo, ghCommentID.Int64, ghReactionID.Int64, "comment"); err != nil {
				s.logger.Warn("Failed to delete comment reaction from GitHub",
					zap.Int64("comment_id", commentID), zap.String("reaction", reaction), zap.Error(err))
			}
		}
	} else {
		if issueNumber == 0 {
			return
		}
		if add {
			ghReactionID, err := pushReactionToGitHub(ctx, token, owner, repo, issueNumber, reaction, "issue")
			if err != nil {
				s.logger.Warn("Failed to push task reaction to GitHub",
					zap.Int64("task_id", taskID), zap.String("reaction", reaction), zap.Error(err))
				return
			}
			_, _ = s.db.ExecContext(ctx,
				`UPDATE github_reactions SET github_reaction_id = $1 WHERE task_id = $2 AND reaction = $3`,
				ghReactionID, taskID, reaction,
			)
		} else {
			var ghReactionID sql.NullInt64
			_ = s.db.QueryRowContext(ctx,
				`SELECT github_reaction_id FROM github_reactions WHERE task_id = $1 AND reaction = $2`,
				taskID, reaction,
			).Scan(&ghReactionID)
			if !ghReactionID.Valid {
				return
			}
			if err := deleteReactionFromGitHub(ctx, token, owner, repo, issueNumber, ghReactionID.Int64, "issue"); err != nil {
				s.logger.Warn("Failed to delete task reaction from GitHub",
					zap.Int64("task_id", taskID), zap.String("reaction", reaction), zap.Error(err))
			}
		}
	}
}
