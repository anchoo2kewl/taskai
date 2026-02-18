package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Tag holds the schema definition for the Tag entity.
type Tag struct {
	ent.Schema
}

// Fields of the Tag.
func (Tag) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.Int64("team_id").Optional().Nillable(),
		field.String("name").NotEmpty(),
		field.String("color").Default("#3B82F6"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Tag.
func (Tag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("tags").Unique().Required().Field("user_id"),
		edge.From("team", Team.Type).Ref("tags").Unique().Field("team_id"),
		edge.To("task_tags", TaskTag.Type),
	}
}

// Indexes of the Tag.
func (Tag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("team_id"),
		index.Fields("user_id", "name").Unique(),
	}
}
