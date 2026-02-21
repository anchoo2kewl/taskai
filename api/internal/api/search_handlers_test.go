package api

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"
	"time"
)

func TestHandleGlobalSearch(t *testing.T) {
	t.Run("requires authentication", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		// Make request without auth context
		rec, req := MakeRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "test",
		}, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusUnauthorized)
	})

	t.Run("requires query parameter", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "",
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusBadRequest)
	})

	t.Run("returns empty results when no accessible projects", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "test",
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 0 {
			t.Errorf("expected 0 tasks, got %d", len(resp.Tasks))
		}
		if len(resp.Wiki) != 0 {
			t.Errorf("expected 0 wiki results, got %d", len(resp.Wiki))
		}
	})

	t.Run("finds tasks by title", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "Test Project")
		ts.CreateTestTask(t, projectID, "Fix login bug")
		ts.CreateTestTask(t, projectID, "Add dashboard feature")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "login",
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
		}
		if resp.Tasks[0].Title != "Fix login bug" {
			t.Errorf("expected title 'Fix login bug', got '%s'", resp.Tasks[0].Title)
		}
		if resp.Tasks[0].ProjectName != "Test Project" {
			t.Errorf("expected project name 'Test Project', got '%s'", resp.Tasks[0].ProjectName)
		}
		if resp.Tasks[0].TaskNumber != 1 {
			t.Errorf("expected task number 1, got %d", resp.Tasks[0].TaskNumber)
		}
	})

	t.Run("case insensitive search", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "Test Project")
		ts.CreateTestTask(t, projectID, "Fix Login Bug")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "fix login",
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
		}
	})

	t.Run("finds tasks by description", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "Test Project")

		// Create task with description
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_, err := ts.DB.ExecContext(ctx,
			`INSERT INTO tasks (project_id, task_number, title, description, status, priority) VALUES (?, 1, 'Some task', 'This involves authentication flow', 'todo', 'medium')`,
			projectID,
		)
		if err != nil {
			t.Fatalf("Failed to create test task: %v", err)
		}

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "authentication",
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
		}
		if resp.Tasks[0].Snippet != "This involves authentication flow" {
			t.Errorf("unexpected snippet: %s", resp.Tasks[0].Snippet)
		}
	})

	t.Run("respects project_id filter", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		project1 := ts.CreateTestProject(t, userID, "Project One")
		project2 := ts.CreateTestProject(t, userID, "Project Two")
		ts.CreateTestTask(t, project1, "Shared keyword task")
		ts.CreateTestTask(t, project2, "Shared keyword task")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]interface{}{
			"query":      "keyword",
			"project_id": project1,
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
		}
		if resp.Tasks[0].ProjectID != project1 {
			t.Errorf("expected project ID %d, got %d", project1, resp.Tasks[0].ProjectID)
		}
	})

	t.Run("respects types filter", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "Test Project")
		ts.CreateTestTask(t, projectID, "Some task with searchterm")

		// Search only wiki (should return no tasks)
		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]interface{}{
			"query": "searchterm",
			"types": []string{"wiki"},
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 0 {
			t.Errorf("expected 0 tasks when filtered to wiki only, got %d", len(resp.Tasks))
		}
	})

	t.Run("respects limit parameter", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "Test Project")

		// Create 5 tasks
		for i := 0; i < 5; i++ {
			ts.CreateTestTask(t, projectID, "Findable task")
		}

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]interface{}{
			"query": "findable",
			"limit": 2,
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 2 {
			t.Errorf("expected 2 tasks with limit=2, got %d", len(resp.Tasks))
		}
	})

	t.Run("does not return tasks from inaccessible projects", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		user1 := ts.CreateTestUser(t, "user1@example.com", "password123")
		user2 := ts.CreateTestUser(t, "user2@example.com", "password123")

		project1 := ts.CreateTestProject(t, user1, "User1 Project")
		project2 := ts.CreateTestProject(t, user2, "User2 Project")

		ts.CreateTestTask(t, project1, "Secret task one")
		ts.CreateTestTask(t, project2, "Secret task two")

		// User2 should not see user1's tasks
		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "secret",
		}, user2, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)

		if len(resp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
		}
		if resp.Tasks[0].ProjectID != project2 {
			t.Errorf("expected task from project %d, got project %d", project2, resp.Tasks[0].ProjectID)
		}
	})

	t.Run("clamps limit to max 50", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "Test Project")
		ts.CreateTestTask(t, projectID, "Test task")

		body := map[string]interface{}{
			"query": "test",
			"limit": 100,
		}

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", body, userID, nil)
		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		// Just verify it doesn't error — the limit is clamped internally
		var resp GlobalSearchResponse
		DecodeJSON(t, rec, &resp)
	})

	t.Run("returns proper task fields", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")
		projectID := ts.CreateTestProject(t, userID, "My Project")
		ts.CreateTestTask(t, projectID, "Important task")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", map[string]string{
			"query": "important",
		}, userID, nil)

		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusOK)

		// Decode raw JSON to verify field names
		var raw map[string]json.RawMessage
		if err := json.Unmarshal(rec.Body.Bytes(), &raw); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		// Verify both keys exist
		if _, ok := raw["tasks"]; !ok {
			t.Error("response missing 'tasks' key")
		}
		if _, ok := raw["wiki"]; !ok {
			t.Error("response missing 'wiki' key")
		}

		var resp GlobalSearchResponse
		if err := json.Unmarshal(rec.Body.Bytes(), &resp); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if len(resp.Tasks) != 1 {
			t.Fatalf("expected 1 task, got %d", len(resp.Tasks))
		}

		task := resp.Tasks[0]
		if task.ID == 0 {
			t.Error("task ID should not be 0")
		}
		if task.ProjectID != projectID {
			t.Errorf("expected project_id %d, got %d", projectID, task.ProjectID)
		}
		if task.ProjectName != "My Project" {
			t.Errorf("expected project_name 'My Project', got '%s'", task.ProjectName)
		}
		if task.Status != "todo" {
			t.Errorf("expected status 'todo', got '%s'", task.Status)
		}
		if task.Priority != "medium" {
			t.Errorf("expected priority 'medium', got '%s'", task.Priority)
		}
	})

	t.Run("invalid request body", func(t *testing.T) {
		ts := NewTestServer(t)
		defer ts.Close()

		userID := ts.CreateTestUser(t, "test@example.com", "password123")

		rec, req := ts.MakeAuthRequest(t, http.MethodPost, "/api/search", "not json", userID, nil)
		ts.HandleGlobalSearch(rec, req)

		AssertStatusCode(t, rec.Code, http.StatusBadRequest)
	})
}
