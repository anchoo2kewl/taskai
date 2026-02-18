package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TaskComment holds the schema definition for the TaskComment entity.
type TaskComment struct {
	ent.Schema
}

// Fields of the TaskComment.
func (TaskComment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("task_id"),
		field.Int64("user_id"),
		field.String("comment").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the TaskComment.
func (TaskComment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).Ref("comments").Unique().Required().Field("task_id"),
		edge.From("user", User.Type).Ref("task_comments").Unique().Required().Field("user_id"),
	}
}

// Indexes of the TaskComment.
func (TaskComment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("user_id"),
	}
}
