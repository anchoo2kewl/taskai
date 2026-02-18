package api

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"time"

	"go.uber.org/zap"

	"taskai/ent"
	"taskai/ent/user"
	"taskai/ent/useractivity"
)

// UserWithStats represents a user with admin and activity stats
type UserWithStats struct {
	ID             int64     `json:"id"`
	Email          string    `json:"email"`
	IsAdmin        bool      `json:"is_admin"`
	CreatedAt      time.Time `json:"created_at"`
	LoginCount     int       `json:"login_count"`
	LastLoginAt    *string   `json:"last_login_at"`
	LastLoginIP    *string   `json:"last_login_ip"`
	FailedAttempts int       `json:"failed_attempts"`
	InviteCount    int       `json:"invite_count"`
}

// UserActivity represents a user activity log entry
type UserActivity struct {
	ID           int64   `json:"id"`
	UserID       int64   `json:"user_id"`
	ActivityType string  `json:"activity_type"` // 'login', 'logout', 'failed_login'
	IPAddress    *string `json:"ip_address"`
	UserAgent    *string `json:"user_agent"`
	CreatedAt    string  `json:"created_at"`
}

// HandleGetUsers returns all users (admin only)
func (s *Server) HandleGetUsers(w http.ResponseWriter, r *http.Request) {
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

	ctx, cancel := context.WithTimeout(r.Context(), 10*time.Second)
	defer cancel()

	// Get all users
	entUsers, err := s.db.Client.User.Query().
		Order(ent.Desc(user.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Error("Failed to query users", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get users", "internal_error")
		return
	}

	users := make([]UserWithStats, 0, len(entUsers))
	for _, u := range entUsers {
		userStats := UserWithStats{
			ID:          u.ID,
			Email:       u.Email,
			IsAdmin:     u.IsAdmin,
			CreatedAt:   u.CreatedAt,
			InviteCount: u.InviteCount,
		}

		// Get login count and last login
		loginActivities, err := s.db.Client.UserActivity.Query().
			Where(
				useractivity.UserID(u.ID),
				useractivity.ActivityType("login"),
			).
			Order(ent.Desc(useractivity.FieldCreatedAt)).
			Limit(1).
			All(ctx)
		if err == nil {
			// Count all logins
			loginCount, _ := s.db.Client.UserActivity.Query().
				Where(
					useractivity.UserID(u.ID),
					useractivity.ActivityType("login"),
				).
				Count(ctx)
			userStats.LoginCount = loginCount

			// Get last login details
			if len(loginActivities) > 0 {
				lastLogin := loginActivities[0]
				lastLoginStr := lastLogin.CreatedAt.Format(time.RFC3339)
				userStats.LastLoginAt = &lastLoginStr
				userStats.LastLoginIP = lastLogin.IPAddress
			}
		}

		// Get failed login count
		failedCount, err := s.db.Client.UserActivity.Query().
			Where(
				useractivity.UserID(u.ID),
				useractivity.ActivityType("failed_login"),
			).
			Count(ctx)
		if err == nil {
			userStats.FailedAttempts = failedCount
		}

		users = append(users, userStats)
	}

	respondJSON(w, http.StatusOK, users)
}

// HandleGetUserActivity returns activity log for a specific user (admin only)
func (s *Server) HandleGetUserActivity(w http.ResponseWriter, r *http.Request) {
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

	// Get target user ID from path
	targetUserIDStr := r.PathValue("id")
	if targetUserIDStr == "" {
		respondError(w, http.StatusBadRequest, "user id required", "validation_error")
		return
	}

	var targetUserID int64
	if _, err := fmt.Sscanf(targetUserIDStr, "%d", &targetUserID); err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id", "validation_error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	entActivities, err := s.db.Client.UserActivity.Query().
		Where(useractivity.UserID(targetUserID)).
		Order(ent.Desc(useractivity.FieldCreatedAt)).
		Limit(100).
		All(ctx)
	if err != nil {
		s.logger.Error("Failed to query user activity", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get activity", "internal_error")
		return
	}

	activities := make([]UserActivity, 0, len(entActivities))
	for _, ea := range entActivities {
		activities = append(activities, UserActivity{
			ID:           ea.ID,
			UserID:       ea.UserID,
			ActivityType: ea.ActivityType,
			IPAddress:    ea.IPAddress,
			UserAgent:    ea.UserAgent,
			CreatedAt:    ea.CreatedAt.Format(time.RFC3339),
		})
	}

	respondJSON(w, http.StatusOK, activities)
}

// HandleUpdateUserAdmin updates admin status of a user (admin only)
func (s *Server) HandleUpdateUserAdmin(w http.ResponseWriter, r *http.Request) {
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

	// Get target user ID from path
	targetUserIDStr := r.PathValue("id")
	if targetUserIDStr == "" {
		respondError(w, http.StatusBadRequest, "user id required", "validation_error")
		return
	}

	var targetUserID int64
	if _, err := fmt.Sscanf(targetUserIDStr, "%d", &targetUserID); err != nil {
		respondError(w, http.StatusBadRequest, "invalid user id", "validation_error")
		return
	}

	var req struct {
		IsAdmin bool `json:"is_admin"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", "invalid_request")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	err := s.db.Client.User.UpdateOneID(targetUserID).
		SetIsAdmin(req.IsAdmin).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "user not found", "not_found")
			return
		}
		s.logger.Error("Failed to update user admin status", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to update user", "internal_error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":       targetUserID,
		"is_admin": req.IsAdmin,
	})
}

// isAdmin checks if a user is an admin
func (s *Server) isAdmin(ctx context.Context, userID int64) bool {
	userEntity, err := s.db.Client.User.Get(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to check admin status", zap.Error(err))
		return false
	}
	return userEntity.IsAdmin
}

// logUserActivity logs a user activity event using Ent
func (s *Server) logUserActivity(ctx context.Context, userID int64, activityType, ipAddress, userAgent string) error {
	creator := s.db.Client.UserActivity.Create().
		SetUserID(userID).
		SetActivityType(activityType)

	// Set optional fields only if not empty
	if ipAddress != "" {
		creator.SetIPAddress(ipAddress)
	}
	if userAgent != "" {
		creator.SetUserAgent(userAgent)
	}

	_, err := creator.Save(ctx)
	if err != nil {
		s.logger.Error("Failed to log user activity", zap.Error(err))
	}
	return err
}

// getClientIP extracts the client IP address from the request
func getClientIP(r *http.Request) string {
	// Check X-Forwarded-For header first (for proxies/load balancers)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// XFF can contain multiple IPs, get the first one
		if idx := strings.Index(xff, ","); idx != -1 {
			return strings.TrimSpace(xff[:idx])
		}
		return strings.TrimSpace(xff)
	}

	// Check X-Real-IP header
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	return r.RemoteAddr
}
