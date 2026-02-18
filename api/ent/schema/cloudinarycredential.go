package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// CloudinaryCredential holds the schema definition for the CloudinaryCredential entity.
type CloudinaryCredential struct {
	ent.Schema
}

// Fields of the CloudinaryCredential.
func (CloudinaryCredential) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("user_id").Unique(),
		field.String("cloud_name").NotEmpty().Sensitive(),
		field.String("api_key").NotEmpty().Sensitive(),
		field.String("api_secret").NotEmpty().Sensitive(),
		field.Int("max_file_size_mb").Default(10),
		field.String("status").Default("unknown"),
		field.Time("last_checked_at").Optional().Nillable(),
		field.String("last_error").Default(""),
		field.Int("consecutive_failures").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the CloudinaryCredential.
func (CloudinaryCredential) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("user", User.Type).Ref("cloudinary_credentials").Unique().Required().Field("user_id"),
	}
}

// Indexes of the CloudinaryCredential.
func (CloudinaryCredential) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("user_id").Unique(),
	}
}
