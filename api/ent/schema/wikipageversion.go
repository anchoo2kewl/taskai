package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WikiPageVersion holds the schema definition for the WikiPageVersion entity.
type WikiPageVersion struct {
	ent.Schema
}

// Fields of the WikiPageVersion.
func (WikiPageVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("wiki_page_id"),
		field.Int("version_number"),
		field.Text("content").Default(""),
		field.String("content_hash").MaxLen(64),
		field.Int64("created_by"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the WikiPageVersion.
func (WikiPageVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("wiki_page", WikiPage.Type).Ref("wiki_page_versions").Unique().Required().Field("wiki_page_id"),
		edge.From("creator", User.Type).Ref("wiki_page_versions_created").Unique().Required().Field("created_by"),
	}
}

// Indexes of the WikiPageVersion.
func (WikiPageVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("wiki_page_id"),
		index.Fields("wiki_page_id", "version_number").Unique(),
	}
}
