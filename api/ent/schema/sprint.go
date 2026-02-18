package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Sprint holds the schema definition for the Sprint entity.
type Sprint struct {
	ent.Schema
}

// Fields of the Sprint.
func (Sprint) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.Int64("team_id").Optional().Nillable(),
		field.String("name").NotEmpty(),
		field.String("goal").Optional().Nillable(),
		field.Time("start_date").Optional().Nillable(),
		field.Time("end_date").Optional().Nillable(),
		field.String("status").Default("planned"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Sprint.
func (Sprint) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("sprints").Unique().Required().Field("user_id"),
		edge.From("team", Team.Type).Ref("sprints").Unique().Field("team_id"),
		edge.To("tasks", Task.Type),
	}
}

// Indexes of the Sprint.
func (Sprint) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("team_id"),
		index.Fields("status"),
	}
}
