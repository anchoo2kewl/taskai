package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TeamMember holds the schema definition for the TeamMember entity.
type TeamMember struct {
	ent.Schema
}

// Fields of the TeamMember.
func (TeamMember) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("team_id"),
		field.Int64("user_id"),
		field.String("role").Default("member"),
		field.String("status").Default("active"),
		field.Time("joined_at").Default(time.Now).Immutable(),
	}
}

// Edges of the TeamMember.
func (TeamMember) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).Ref("members").Unique().Required().Field("team_id"),
		edge.From("user", User.Type).Ref("team_memberships").Unique().Required().Field("user_id"),
	}
}

// Indexes of the TeamMember.
func (TeamMember) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("team_id"),
		index.Fields("user_id"),
		index.Fields("team_id", "user_id").Unique(),
	}
}
