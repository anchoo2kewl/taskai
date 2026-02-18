package api

import (
	"context"
	"encoding/base64"
	"time"

	"go.uber.org/zap"

	"taskai/ent"
	"taskai/ent/pageversion"
	"taskai/ent/wikiblock"
	"taskai/ent/wikipage"
)

// StartIndexingWorker starts a background worker that periodically indexes wiki content
func (s *Server) StartIndexingWorker(ctx context.Context) {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()

	s.logger.Info("Starting wiki indexing worker",
		zap.Duration("interval", 2*time.Minute),
	)

	// Run immediately on startup
	s.indexPages(ctx)

	for {
		select {
		case <-ctx.Done():
			s.logger.Info("Indexing worker shutting down")
			return
		case <-ticker.C:
			s.indexPages(ctx)
		}
	}
}

// indexPages finds pages that need indexing and indexes them
func (s *Server) indexPages(parentCtx context.Context) {
	ctx, cancel := context.WithTimeout(parentCtx, 2*time.Minute)
	defer cancel()

	// Find pages that have been updated recently (within last 3 minutes)
	pages, err := s.db.Client.WikiPage.Query().
		Where(wikipage.UpdatedAtGTE(time.Now().Add(-3 * time.Minute))).
		All(ctx)
	if err != nil {
		s.logger.Error("Failed to fetch pages for indexing",
			zap.Error(err),
		)
		return
	}

	if len(pages) == 0 {
		s.logger.Debug("No pages need indexing")
		return
	}

	s.logger.Info("Indexing pages",
		zap.Int("page_count", len(pages)),
	)

	successCount := 0
	failCount := 0

	for _, page := range pages {
		if err := s.indexPage(ctx, page); err != nil {
			s.logger.Error("Failed to index page",
				zap.Int64("page_id", page.ID),
				zap.String("page_title", page.Title),
				zap.Error(err),
			)
			failCount++
		} else {
			successCount++
		}
	}

	s.logger.Info("Indexing completed",
		zap.Int("success", successCount),
		zap.Int("failed", failCount),
	)
}

// indexPage indexes a single wiki page
func (s *Server) indexPage(ctx context.Context, page *ent.WikiPage) error {
	// Get the latest snapshot for the page
	snapshot, err := s.db.Client.PageVersion.Query().
		Where(pageversion.PageID(page.ID)).
		Order(ent.Desc(pageversion.FieldVersionNumber)).
		First(ctx)
	if err != nil {
		if ent.IsNotFound(err) {
			s.logger.Debug("No snapshot available for indexing",
				zap.Int64("page_id", page.ID),
			)
			return nil
		}
		return err
	}

	// Convert state to base64 for Yjs processor
	stateBase64 := base64.StdEncoding.EncodeToString(snapshot.YjsState)

	// Extract blocks from the snapshot
	blocks, err := s.yjsClient.ExtractBlocks(ctx, stateBase64)
	if err != nil {
		return err
	}

	// Delete existing blocks for this page
	_, err = s.db.Client.WikiBlock.Delete().
		Where(wikiblock.PageID(page.ID)).
		Exec(ctx)
	if err != nil {
		return err
	}

	// Insert new blocks
	if len(blocks) > 0 {
		bulk := make([]*ent.WikiBlockCreate, len(blocks))
		for i, block := range blocks {
			bulk[i] = s.db.Client.WikiBlock.Create().
				SetPageID(page.ID).
				SetBlockType(block.Type).
				SetHeadingsPath(block.HeadingsPath).
				SetPlainText(block.PlainText).
				SetPosition(block.Position)

			// Set optional level for headings
			if block.Level != nil {
				bulk[i].SetLevel(*block.Level)
			}

			// Store canonical JSON as string
			if block.CanonicalJSON != "" {
				bulk[i].SetCanonicalJSON(block.CanonicalJSON)
			}
		}

		_, err = s.db.Client.WikiBlock.CreateBulk(bulk...).Save(ctx)
		if err != nil {
			return err
		}
	}

	s.logger.Info("Indexed page",
		zap.Int64("page_id", page.ID),
		zap.String("page_title", page.Title),
		zap.Int("block_count", len(blocks)),
	)

	return nil
}
