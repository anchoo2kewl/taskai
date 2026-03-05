package api

import (
	"testing"
)

func TestParseGraphLinks(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    []graphLinkRef
	}{
		{
			name:    "empty content",
			content: "",
			want:    []graphLinkRef{},
		},
		{
			name:    "no links",
			content: "Just some text without any links.",
			want:    []graphLinkRef{},
		},
		{
			name:    "wiki link without label",
			content: "See [[wiki:123]] for details.",
			want:    []graphLinkRef{{EntityType: "wiki", EntityID: 123}},
		},
		{
			name:    "wiki link with label",
			content: "See [[wiki:123|Architecture]] for details.",
			want:    []graphLinkRef{{EntityType: "wiki", EntityID: 123}},
		},
		{
			name:    "task link without label",
			content: "Blocked by [[task:456]].",
			want:    []graphLinkRef{{EntityType: "task", EntityID: 456}},
		},
		{
			name:    "task link with label",
			content: "Blocked by [[task:456|Fix the bug]].",
			want:    []graphLinkRef{{EntityType: "task", EntityID: 456}},
		},
		{
			name:    "multiple links",
			content: "See [[wiki:10]] and [[task:20|Sprint task]] and [[wiki:30|Docs]].",
			want: []graphLinkRef{
				{EntityType: "wiki", EntityID: 10},
				{EntityType: "task", EntityID: 20},
				{EntityType: "wiki", EntityID: 30},
			},
		},
		{
			name:    "duplicate links are deduplicated",
			content: "[[wiki:5]] appears twice: [[wiki:5]].",
			want:    []graphLinkRef{{EntityType: "wiki", EntityID: 5}},
		},
		{
			name:    "different entity types are not deduplicated",
			content: "[[wiki:7]] and [[task:7]].",
			want: []graphLinkRef{
				{EntityType: "wiki", EntityID: 7},
				{EntityType: "task", EntityID: 7},
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseGraphLinks(tt.content)
			if len(got) != len(tt.want) {
				t.Errorf("parseGraphLinks() returned %d refs, want %d; got %+v", len(got), len(tt.want), got)
				return
			}
			for i, ref := range got {
				if ref.EntityType != tt.want[i].EntityType || ref.EntityID != tt.want[i].EntityID {
					t.Errorf("ref[%d] = %+v, want %+v", i, ref, tt.want[i])
				}
			}
		})
	}
}

func TestPreprocessGraphLinksForPreview(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantContains []string
	}{
		{
			name:         "wiki link gets data attributes",
			content:      "See [[wiki:42|Docs]].",
			wantContains: []string{`data-graph-type="wiki"`, `data-entity-id="42"`, "Docs", "📄"},
		},
		{
			name:         "task link gets data attributes",
			content:      "Fix [[task:99]].",
			wantContains: []string{`data-graph-type="task"`, `data-entity-id="99"`, "Task #99", "✅"},
		},
		{
			name:         "no links unchanged",
			content:      "plain text",
			wantContains: []string{"plain text"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := preprocessGraphLinksForPreview(tt.content)
			for _, want := range tt.wantContains {
				if !containsStr(got, want) {
					t.Errorf("preprocessGraphLinksForPreview() = %q, want to contain %q", got, want)
				}
			}
		})
	}
}

func containsStr(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstr(s, sub))
}

func findSubstr(s, sub string) bool {
	for i := 0; i <= len(s)-len(sub); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
