package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Team holds the schema definition for the Team entity.
type Team struct {
	ent.Schema
}

// Fields of the Team.
func (Team) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("name").NotEmpty(),
		field.Int64("owner_id"),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the Team.
func (Team) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("owner", User.Type).Ref("owned_teams").Unique().Required().Field("owner_id"),
		edge.To("members", TeamMember.Type),
		edge.To("invitations", TeamInvitation.Type),
		edge.To("projects", Project.Type),
		edge.To("sprints", Sprint.Type),
		edge.To("tags", Tag.Type),
	}
}

// Indexes of the Team.
func (Team) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("owner_id"),
	}
}
