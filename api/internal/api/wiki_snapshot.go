package api

import (
	"context"
	"encoding/base64"
	"time"

	"go.uber.org/zap"

	"taskai/ent"
	"taskai/ent/pageversion"
	"taskai/ent/wikipage"
	"taskai/ent/yjsupdate"
)

// StartSnapshotWorker starts a background worker that periodically creates snapshots of wiki pages
func (s *Server) StartSnapshotWorker(ctx context.Context) {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	s.logger.Info("Starting wiki snapshot worker",
		zap.Duration("interval", 5*time.Minute),
	)

	// Run immediately on startup
	s.generateSnapshots(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Snapshot worker shutting down")
			return
		case <-ticker.C:
			s.generateSnapshots(ctx)
		}
	}
}

// generateSnapshots finds pages that need snapshots and generates them
func (s *Server) generateSnapshots(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
	defer cancel()

	// Find pages that have been updated since their last snapshot
	pages, err := s.db.Client.WikiPage.Query().
		Where(wikipage.UpdatedAtGTE(time.Now().Add(-6 * time.Minute))).
		All(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch pages for snapshot",
			zap.Error(err),
		)
		return
	}

	if len(pages) == 0 {
		s.logger.Debug("No pages need snapshots")
		return
	}

	s.logger.Info("Generating snapshots",
		zap.Int("page_count", len(pages)),
	)

	successCount := 0
	failCount := 0

	for _, page := range pages {
		if err := s.generatePageSnapshot(ctx, page); err != nil {
			s.logger.Error("Failed to generate snapshot",
				zap.Int64("page_id", page.ID),
				zap.String("page_title", page.Title),
				zap.Error(err),
			)
			failCount++
		} else {
			successCount++
		}
	}

	s.logger.Info("Snapshot generation completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
	)
}

// generatePageSnapshot generates a snapshot for a single wiki page
func (s *Server) generatePageSnapshot(ctx context.Context, page *ent.WikiPage) error {
	// Fetch all Yjs updates for the page
	updates, err := s.db.Client.YjsUpdate.Query().
		Where(yjsupdate.PageID(page.ID)).
		Order(ent.Asc(yjsupdate.FieldCreatedAt)).
		All(ctx)
	if err != nil {
		return err
	}

	if len(updates) == 0 {
		s.logger.Debug("No updates to snapshot",
			zap.Int64("page_id", page.ID),
		)
		return nil
	}

	// Convert updates to base64 strings
	updateStrings := make([]string, len(updates))
	for i, update := range updates {
		updateStrings[i] = base64.StdEncoding.EncodeToString(update.UpdateData)
	}

	// Call Yjs processor to apply all updates
	state, err := s.yjsClient.ApplyUpdates(ctx, updateStrings)
	if err != nil {
		return err
	}

	// Decode the state back to binary
	stateBytes, err := base64.StdEncoding.DecodeString(state)
	if err != nil {
		return err
	}

	// Get the current version number
	lastVersion, err := s.db.Client.PageVersion.Query().
		Where(pageversion.PageID(page.ID)).
		Order(ent.Desc(pageversion.FieldVersionNumber)).
		First(ctx)

	versionNumber := 1
	if err == nil {
		// Check if the state has changed
		if string(lastVersion.YjsState) == string(stateBytes) {
			s.logger.Debug("State unchanged, skipping snapshot",
				zap.Int64("page_id", page.ID),
			)
			return nil
		}
		versionNumber = lastVersion.VersionNumber + 1
	}

	// Create new snapshot
	_, err = s.db.Client.PageVersion.Create().
		SetPageID(page.ID).
		SetVersionNumber(versionNumber).
		SetYjsState(stateBytes).
		Save(ctx)
	if err != nil {
		return err
	}

	s.logger.Info("Generated page snapshot",
		zap.Int64("page_id", page.ID),
		zap.String("page_title", page.Title),
		zap.Int("version", versionNumber),
		zap.Int("update_count", len(updates)),
	)

	return nil
}
