package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// ProjectMember holds the schema definition for the ProjectMember entity.
type ProjectMember struct {
	ent.Schema
}

// Fields of the ProjectMember.
func (ProjectMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("project_id"),
		field.Int64("user_id"),
		field.String("role").Default("member"),
		field.Int64("granted_by"),
		field.Time("granted_at").Default(time.Now).Immutable(),
	}
}

// Edges of the ProjectMember.
func (ProjectMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("project", Project.Type).Ref("members").Unique().Required().Field("project_id"),
		edge.From("user", User.Type).Ref("project_memberships").Unique().Required().Field("user_id"),
		edge.From("granter", User.Type).Ref("project_memberships_granted").Unique().Required().Field("granted_by"),
	}
}

// Indexes of the ProjectMember.
func (ProjectMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("project_id"),
		index.Fields("user_id"),
		index.Fields("role"),
		index.Fields("project_id", "user_id").Unique(),
	}
}
