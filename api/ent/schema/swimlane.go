package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// SwimLane holds the schema definition for the SwimLane entity.
type SwimLane struct {
	ent.Schema
}

// Fields of the SwimLane.
func (SwimLane) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("project_id"),
		field.String("name").NotEmpty(),
		field.String("color").NotEmpty(),
		field.Int("position"),
		field.String("status_category").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the SwimLane.
func (SwimLane) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("swim_lanes").Unique().Required().Field("project_id"),
		edge.To("tasks", Task.Type),
	}
}

// Indexes of the SwimLane.
func (SwimLane) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("project_id", "position").Unique(),
	}
}
