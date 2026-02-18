package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// TeamInvitation holds the schema definition for the TeamInvitation entity.
type TeamInvitation struct {
	ent.Schema
}

// Fields of the TeamInvitation.
func (TeamInvitation) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("team_id"),
		field.Int64("inviter_id"),
		field.String("invitee_email").NotEmpty(),
		field.Int64("invitee_id").Optional().Nillable(),
		field.String("status").Default("pending"),
		field.String("acceptance_token").Optional().Nillable(),
		field.String("invite_code").Optional().Nillable(),
		field.Time("token_expires_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("responded_at").Optional().Nillable(),
	}
}

// Edges of the TeamInvitation.
func (TeamInvitation) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("team", Team.Type).Ref("invitations").Unique().Required().Field("team_id"),
		edge.From("inviter", User.Type).Ref("team_invitations_sent").Unique().Required().Field("inviter_id"),
		edge.From("invitee", User.Type).Ref("team_invitations_received").Unique().Field("invitee_id"),
	}
}

// Indexes of the TeamInvitation.
func (TeamInvitation) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("team_id"),
		index.Fields("invitee_email"),
		index.Fields("invitee_id"),
		index.Fields("status"),
		index.Fields("acceptance_token").Unique(),
	}
}
