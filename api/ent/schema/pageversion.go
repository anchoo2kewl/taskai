package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// PageVersion holds the schema definition for the PageVersion entity.
type PageVersion struct {
	ent.Schema
}

// Fields of the PageVersion.
func (PageVersion) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("page_id"),
		field.Int("version_number"),
		field.Bytes("yjs_state").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the PageVersion.
func (PageVersion) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("page", WikiPage.Type).Ref("versions").Unique().Required().Field("page_id"),
	}
}

// Indexes of the PageVersion.
func (PageVersion) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("page_id"),
		index.Fields("page_id", "version_number").Unique(),
	}
}
