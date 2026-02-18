package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TaskTag holds the schema definition for the TaskTag entity (junction table).
type TaskTag struct {
	ent.Schema
}

// Fields of the TaskTag.
func (TaskTag) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("task_id"),
		field.Int64("tag_id"),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the TaskTag.
func (TaskTag) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).Ref("task_tags").Unique().Required().Field("task_id"),
		edge.From("tag", Tag.Type).Ref("task_tags").Unique().Required().Field("tag_id"),
	}
}

// Indexes of the TaskTag.
func (TaskTag) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("tag_id"),
		index.Fields("task_id", "tag_id").Unique(),
	}
}
