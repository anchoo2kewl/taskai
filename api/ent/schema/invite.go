package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// Invite holds the schema definition for the Invite entity.
type Invite struct {
	ent.Schema
}

// Fields of the Invite.
func (Invite) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("code").Unique().NotEmpty(),
		field.Int64("inviter_id"),
		field.Int64("invitee_id").Optional().Nillable(),
		field.Time("used_at").Optional().Nillable(),
		field.Time("expires_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the Invite.
func (Invite) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("inviter", User.Type).Ref("invites_sent").Unique().Required().Field("inviter_id"),
		edge.From("invitee", User.Type).Ref("invites_received").Unique().Field("invitee_id"),
	}
}

// Indexes of the Invite.
func (Invite) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("code").Unique(),
		index.Fields("inviter_id"),
	}
}
