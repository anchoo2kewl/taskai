package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// UserActivity holds the schema definition for the UserActivity entity.
type UserActivity struct {
	ent.Schema
}

// Fields of the UserActivity.
func (UserActivity) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.String("activity_type").NotEmpty(),
		field.String("ip_address").Optional().Nillable(),
		field.String("user_agent").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
	}
}

// Edges of the UserActivity.
func (UserActivity) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("user_activities").Unique().Required().Field("user_id"),
	}
}

// Indexes of the UserActivity.
func (UserActivity) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("activity_type"),
		index.Fields("created_at"),
	}
}
