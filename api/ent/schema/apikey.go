package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// APIKey holds the schema definition for the APIKey entity.
type APIKey struct {
	ent.Schema
}

// Fields of the APIKey.
func (APIKey) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id"),
		field.String("name").NotEmpty(),
		field.String("key_hash").NotEmpty().Unique().Sensitive(),
		field.String("key_prefix").NotEmpty(),
		field.Time("last_used_at").Optional().Nillable(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("expires_at").Optional().Nillable(),
	}
}

// Edges of the APIKey.
func (APIKey) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("api_keys").Unique().Required().Field("user_id"),
	}
}

// Indexes of the APIKey.
func (APIKey) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id"),
		index.Fields("key_hash").Unique(),
	}
}
