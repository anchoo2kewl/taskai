package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TaskAttachment holds the schema definition for the TaskAttachment entity.
type TaskAttachment struct {
	ent.Schema
}

// Fields of the TaskAttachment.
func (TaskAttachment) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("task_id"),
		field.Int64("project_id"),
		field.Int64("user_id"),
		field.String("filename").NotEmpty(),
		field.String("file_type").NotEmpty(),
		field.String("content_type").NotEmpty(),
		field.Int("file_size"),
		field.String("cloudinary_url").NotEmpty(),
		field.String("cloudinary_public_id").NotEmpty(),
		field.String("alt_name").Default(""),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the TaskAttachment.
func (TaskAttachment) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("task", Task.Type).Ref("attachments").Unique().Required().Field("task_id"),
		edge.From("project", Project.Type).Ref("attachments").Unique().Required().Field("project_id"),
		edge.From("user", User.Type).Ref("task_attachments").Unique().Required().Field("user_id"),
	}
}

// Indexes of the TaskAttachment.
func (TaskAttachment) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("task_id"),
		index.Fields("user_id"),
		index.Fields("project_id"),
	}
}
