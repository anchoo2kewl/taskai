package schema

import (
	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// WikiBlock holds the schema definition for the WikiBlock entity.
type WikiBlock struct {
	ent.Schema
}

// Fields of the WikiBlock.
func (WikiBlock) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("page_id"),
		field.String("block_type").NotEmpty().MaxLen(50),
		field.Int("level").Optional().Nillable(),
		field.String("headings_path").Optional().Nillable(),
		field.String("canonical_json").Optional().Nillable(),
		field.Text("plain_text").Optional().Nillable(),
		field.Int("position"),
		field.String("search_text").Optional().Nillable(), // For SQLite
	}
}

// Edges of the WikiBlock.
func (WikiBlock) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("page", WikiPage.Type).Ref("blocks").Unique().Required().Field("page_id"),
	}
}

// Indexes of the WikiBlock.
func (WikiBlock) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("page_id"),
	}
}
