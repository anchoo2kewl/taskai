package api

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"net/http"
	"time"

	"go.uber.org/zap"

	"taskai/ent"
	"taskai/ent/invite"
)

// Invite represents an invite record
type Invite struct {
	ID          int64   `json:"id"`
	Code        string  `json:"code"`
	InviterID   int64   `json:"inviter_id"`
	InviterName *string `json:"inviter_name,omitempty"`
	InviteeID   *int64  `json:"invitee_id,omitempty"`
	InviteeName *string `json:"invitee_name,omitempty"`
	UsedAt      *string `json:"used_at,omitempty"`
	ExpiresAt   *string `json:"expires_at,omitempty"`
	CreatedAt   string  `json:"created_at"`
}

// InviteStatus is returned for invite validation
type InviteStatus struct {
	Valid       bool   `json:"valid"`
	InviterName string `json:"inviter_name,omitempty"`
	Message     string `json:"message,omitempty"`
}

// generateInviteCode creates a random URL-safe invite code
func generateInviteCode() (string, error) {
	b := make([]byte, 18)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return base64.URLEncoding.WithPadding(base64.NoPadding).EncodeToString(b), nil
}

// HandleListInvites returns the current user's invites
func (s *Server) HandleListInvites(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated", "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Fetch invites with invitee details
	entInvites, err := s.db.Client.Invite.Query().
		Where(invite.InviterID(userID)).
		WithInvitee().
		Order(ent.Desc(invite.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		s.logger.Error("Failed to query invites", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to list invites", "internal_error")
		return
	}

	type InviteWithDetails struct {
		ID          int64   `json:"id"`
		Code        string  `json:"code"`
		InviterID   int64   `json:"inviter_id"`
		InviteeID   *int64  `json:"invitee_id,omitempty"`
		InviteeName *string `json:"invitee_name,omitempty"`
		UsedAt      *string `json:"used_at,omitempty"`
		ExpiresAt   *string `json:"expires_at,omitempty"`
		CreatedAt   string  `json:"created_at"`
	}

	invites := make([]InviteWithDetails, 0, len(entInvites))
	for _, ei := range entInvites {
		inv := InviteWithDetails{
			ID:        ei.ID,
			Code:      ei.Code,
			InviterID: ei.InviterID,
			InviteeID: ei.InviteeID,
			CreatedAt: ei.CreatedAt.Format(time.RFC3339),
		}
		if ei.UsedAt != nil {
			usedAtStr := ei.UsedAt.Format(time.RFC3339)
			inv.UsedAt = &usedAtStr
		}
		if ei.ExpiresAt != nil {
			expiresAtStr := ei.ExpiresAt.Format(time.RFC3339)
			inv.ExpiresAt = &expiresAtStr
		}
		if ei.Edges.Invitee != nil {
			if ei.Edges.Invitee.Name != nil && *ei.Edges.Invitee.Name != "" {
				inv.InviteeName = ei.Edges.Invitee.Name
			} else {
				inv.InviteeName = &ei.Edges.Invitee.Email
			}
		}
		invites = append(invites, inv)
	}

	// Also get the user's invite count
	userEntity, err := s.db.Client.User.Get(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to get invite count", "internal_error")
		return
	}

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"invites":      invites,
		"invite_count": userEntity.InviteCount,
		"is_admin":     userEntity.IsAdmin,
	})
}

// HandleCreateInvite creates a new invite code and optionally sends an email
func (s *Server) HandleCreateInvite(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated", "unauthorized")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Parse optional email from request body
	var req struct {
		Email string `json:"email"`
	}
	// Body may be empty (backwards compatible)
	_ = decodeJSON(r, &req)

	// Check invite count (admins have unlimited)
	userEntity, err := s.db.Client.User.Get(ctx, userID)
	if err != nil {
		s.logger.Error("Failed to get user", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create invite", "internal_error")
		return
	}

	if !userEntity.IsAdmin && userEntity.InviteCount <= 0 {
		respondError(w, http.StatusForbidden, "no invites remaining", "no_invites")
		return
	}

	// Generate invite code
	code, err := generateInviteCode()
	if err != nil {
		s.logger.Error("Failed to generate invite code", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create invite", "internal_error")
		return
	}

	// Set expiry to 7 days
	expiresAt := time.Now().Add(7 * 24 * time.Hour)

	// Use Ent transaction
	tx, err := s.db.Client.Tx(ctx)
	if err != nil {
		s.logger.Error("Failed to begin transaction", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create invite", "internal_error")
		return
	}
	defer tx.Rollback()

	// Insert invite
	_, err = tx.Invite.Create().
		SetCode(code).
		SetInviterID(userID).
		SetExpiresAt(expiresAt).
		Save(ctx)
	if err != nil {
		s.logger.Error("Failed to insert invite", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create invite", "internal_error")
		return
	}

	// Decrement invite count (only for non-admins)
	if !userEntity.IsAdmin {
		_, err = tx.User.UpdateOneID(userID).
			SetInviteCount(userEntity.InviteCount - 1).
			Save(ctx)
		if err != nil {
			s.logger.Error("Failed to decrement invite count", zap.Error(err))
			respondError(w, http.StatusInternalServerError, "failed to create invite", "internal_error")
			return
		}
	}

	if err := tx.Commit(); err != nil {
		s.logger.Error("Failed to commit transaction", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to create invite", "internal_error")
		return
	}

	s.logger.Info("Invite created", zap.Int64("user_id", userID), zap.String("code", code[:8]+"..."))

	// Send email if requested and email service is available
	inviterName := userEntity.Email
	if userEntity.Name != nil && *userEntity.Name != "" {
		inviterName = *userEntity.Name
	}

	emailSent := false
	if req.Email != "" {
		if emailSvc := s.GetEmailService(); emailSvc != nil {
			appURL := s.getAppURL()
			if err := emailSvc.SendUserInvite(ctx, req.Email, inviterName, code, appURL); err != nil {
				s.logger.Warn("Failed to send invite email",
					zap.String("to", req.Email),
					zap.Error(err),
				)
			} else {
				emailSent = true
			}
		}
	}

	respondJSON(w, http.StatusCreated, map[string]interface{}{
		"code":       code,
		"expires_at": expiresAt.Format(time.RFC3339),
		"email_sent": emailSent,
	})
}

// HandleValidateInvite checks if an invite code is valid (public endpoint)
func (s *Server) HandleValidateInvite(w http.ResponseWriter, r *http.Request) {
	code := r.URL.Query().Get("code")
	if code == "" {
		respondJSON(w, http.StatusOK, InviteStatus{Valid: false, Message: "invite code is required"})
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	inviteEntity, err := s.db.Client.Invite.Query().
		Where(invite.Code(code)).
		WithInviter().
		Only(ctx)

	if err != nil {
		if ent.IsNotFound(err) {
			respondJSON(w, http.StatusOK, InviteStatus{Valid: false, Message: "invalid invite code"})
			return
		}
		s.logger.Error("Failed to validate invite", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to validate invite", "internal_error")
		return
	}

	if inviteEntity.UsedAt != nil {
		respondJSON(w, http.StatusOK, InviteStatus{Valid: false, Message: "this invite has already been used"})
		return
	}

	if inviteEntity.ExpiresAt != nil && time.Now().After(*inviteEntity.ExpiresAt) {
		respondJSON(w, http.StatusOK, InviteStatus{Valid: false, Message: "this invite has expired"})
		return
	}

	name := ""
	if inviteEntity.Edges.Inviter != nil {
		if inviteEntity.Edges.Inviter.Name != nil && *inviteEntity.Edges.Inviter.Name != "" {
			name = *inviteEntity.Edges.Inviter.Name
		}
	}

	respondJSON(w, http.StatusOK, InviteStatus{Valid: true, InviterName: name})
}

// HandleAdminBoostInvites allows admins to set a user's invite count
func (s *Server) HandleAdminBoostInvites(w http.ResponseWriter, r *http.Request) {
	userID, ok := GetUserID(r)
	if !ok {
		respondError(w, http.StatusUnauthorized, "user not authenticated", "unauthorized")
		return
	}

	if !s.isAdmin(r.Context(), userID) {
		respondError(w, http.StatusForbidden, "admin access required", "forbidden")
		return
	}

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
		InviteCount int `json:"invite_count"`
	}
	if err := decodeJSON(r, &req); err != nil {
		respondError(w, http.StatusBadRequest, "invalid request body", "invalid_request")
		return
	}

	if req.InviteCount < 0 {
		respondError(w, http.StatusBadRequest, "invite count must be non-negative", "validation_error")
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 5*time.Second)
	defer cancel()

	// Update the user
	err := s.db.Client.User.UpdateOneID(targetUserID).
		SetInviteCount(req.InviteCount).
		Exec(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			respondError(w, http.StatusNotFound, "user not found", "not_found")
			return
		}
		s.logger.Error("Failed to update invite count", zap.Error(err))
		respondError(w, http.StatusInternalServerError, "failed to update invite count", "internal_error")
		return
	}

	s.logger.Info("Admin boosted invites",
		zap.Int64("admin_id", userID),
		zap.Int64("target_user_id", targetUserID),
		zap.Int("invite_count", req.InviteCount),
	)

	respondJSON(w, http.StatusOK, map[string]interface{}{
		"id":           targetUserID,
		"invite_count": req.InviteCount,
	})
}
