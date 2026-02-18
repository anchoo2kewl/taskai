package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"taskai/ent/enttest"
	"taskai/ent/wikipage"
	"taskai/internal/db"

	"github.com/go-chi/chi/v5"
	"go.uber.org/zap/zaptest"
	_ "modernc.org/sqlite"
)

func TestWikiHandlers(t *testing.T) {
	t.Skip("Wiki handlers use ent which requires Postgres - skipping until migration complete")
	ctx := context.Background()
	logger := zaptest.NewLogger(t)
	client := enttest.Open(t, "sqlite", "file:ent?mode=memory&cache=shared&_fk=1")
	defer client.Close()

	database := &db.DB{Client: client}
	server := &Server{
		db:     database,
		logger: logger,
	}

	// Create test user
	user, err := client.User.Create().
		SetEmail("test@example.com").
		SetPasswordHash("$2a$10$test").
		Save(ctx)
	if err != nil {
		t.Fatalf("Failed to create test user: %v", err)
	}

	// Create test team
	team, err := client.Team.Create().
		SetName("Test Team").
		SetOwnerID(user.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("Failed to create test team: %v", err)
	}

	// Create test project
	project, err := client.Project.Create().
		SetName("Test Project").
		SetOwnerID(user.ID).
		SetTeamID(team.ID).
		Save(ctx)
	if err != nil {
		t.Fatalf("Failed to create test project: %v", err)
	}

	// Add user as project member
	_, err = client.ProjectMember.Create().
		SetUserID(user.ID).
		SetProjectID(project.ID).
		SetRole("owner").
		Save(ctx)
	if err != nil {
		t.Fatalf("Failed to create project member: %v", err)
	}

	t.Run("CreateWikiPage", func(t *testing.T) {
		reqBody := CreateWikiPageRequest{
			Title: "Getting Started Guide",
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/api/projects/"+strconv.FormatInt(project.ID, 10)+"/wiki/pages", bytes.NewReader(body))
		reqCtx := context.WithValue(req.Context(), UserIDKey, user.ID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("projectId", strconv.FormatInt(project.ID, 10))
		reqCtx = context.WithValue(reqCtx, chi.RouteCtxKey, rctx)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		server.HandleCreateWikiPage(rr, req)

		if status := rr.Code; status != http.StatusCreated {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusCreated, status, rr.Body.String())
		}

		var response WikiPageResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Title != "Getting Started Guide" {
			t.Errorf("Expected title 'Getting Started Guide', got %q", response.Title)
		}
		if response.Slug != "getting-started-guide" {
			t.Errorf("Expected slug 'getting-started-guide', got %q", response.Slug)
		}
	})

	t.Run("ListWikiPages", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "/api/projects/"+strconv.FormatInt(project.ID, 10)+"/wiki/pages", nil)
		reqCtx := context.WithValue(req.Context(), UserIDKey, user.ID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("projectId", strconv.FormatInt(project.ID, 10))
		reqCtx = context.WithValue(reqCtx, chi.RouteCtxKey, rctx)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		server.HandleListWikiPages(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
		}

		var response []WikiPageResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if len(response) < 1 {
			t.Errorf("Expected at least 1 wiki page, got %d", len(response))
		}
	})

	t.Run("GetWikiPage", func(t *testing.T) {
		// Get existing page from previous test
		page, err := client.WikiPage.Query().
			Where(wikipage.ProjectID(project.ID)).
			First(ctx)
		if err != nil {
			t.Fatalf("Failed to get wiki page: %v", err)
		}

		req := httptest.NewRequest(http.MethodGet, "/api/wiki/pages/"+strconv.FormatInt(page.ID, 10), nil)
		reqCtx := context.WithValue(req.Context(), UserIDKey, user.ID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("pageId", strconv.FormatInt(page.ID, 10))
		reqCtx = context.WithValue(reqCtx, chi.RouteCtxKey, rctx)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		server.HandleGetWikiPage(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
		}

		var response WikiPageResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.ID != page.ID {
			t.Errorf("Expected page ID %d, got %d", page.ID, response.ID)
		}
	})

	t.Run("UpdateWikiPage", func(t *testing.T) {
		// Create a new page for this test
		page, err := client.WikiPage.Create().
			SetProjectID(project.ID).
			SetTitle("Old Title").
			SetSlug("old-title").
			SetCreatedBy(user.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wiki page: %v", err)
		}

		newTitle := "Updated Title"
		reqBody := UpdateWikiPageRequest{
			Title: &newTitle,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPatch, "/api/wiki/pages/"+strconv.FormatInt(page.ID, 10), bytes.NewReader(body))
		reqCtx := context.WithValue(req.Context(), UserIDKey, user.ID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("pageId", strconv.FormatInt(page.ID, 10))
		reqCtx = context.WithValue(reqCtx, chi.RouteCtxKey, rctx)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		server.HandleUpdateWikiPage(rr, req)

		if status := rr.Code; status != http.StatusOK {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusOK, status, rr.Body.String())
		}

		var response WikiPageResponse
		if err := json.NewDecoder(rr.Body).Decode(&response); err != nil {
			t.Fatalf("Failed to decode response: %v", err)
		}

		if response.Title != "Updated Title" {
			t.Errorf("Expected title 'Updated Title', got %q", response.Title)
		}
	})

	t.Run("DeleteWikiPage", func(t *testing.T) {
		// Create a page to delete
		page, err := client.WikiPage.Create().
			SetProjectID(project.ID).
			SetTitle("To Delete").
			SetSlug("to-delete").
			SetCreatedBy(user.ID).
			Save(ctx)
		if err != nil {
			t.Fatalf("Failed to create test wiki page: %v", err)
		}

		req := httptest.NewRequest(http.MethodDelete, "/api/wiki/pages/"+strconv.FormatInt(page.ID, 10), nil)
		reqCtx := context.WithValue(req.Context(), UserIDKey, user.ID)
		rctx := chi.NewRouteContext()
		rctx.URLParams.Add("pageId", strconv.FormatInt(page.ID, 10))
		reqCtx = context.WithValue(reqCtx, chi.RouteCtxKey, rctx)
		req = req.WithContext(reqCtx)

		rr := httptest.NewRecorder()
		server.HandleDeleteWikiPage(rr, req)

		if status := rr.Code; status != http.StatusNoContent {
			t.Errorf("Expected status %d, got %d. Body: %s", http.StatusNoContent, status, rr.Body.String())
		}

		// Verify page is deleted
		exists, err := client.WikiPage.Query().
			Where(wikipage.ID(page.ID)).
			Exist(ctx)
		if err != nil {
			t.Fatalf("Failed to check if page exists: %v", err)
		}
		if exists {
			t.Errorf("Page should be deleted but still exists")
		}
	})
}
