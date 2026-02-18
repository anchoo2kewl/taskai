package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/field"
)

// EmailProvider holds the schema definition for the EmailProvider entity (singleton).
type EmailProvider struct {
	ent.Schema
}

// Fields of the EmailProvider.
func (EmailProvider) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.String("provider").Default("brevo"),
		field.String("api_key").NotEmpty().Sensitive(),
		field.String("sender_email").NotEmpty(),
		field.String("sender_name").NotEmpty(),
		field.String("status").Default("unknown"),
		field.Time("last_checked_at").Optional().Nillable(),
		field.String("last_error").Default(""),
		field.Int("consecutive_failures").Default(0),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Time("updated_at").Default(time.Now).UpdateDefault(time.Now),
	}
}

// Edges of the EmailProvider.
func (EmailProvider) Edges() []ent.Edge {
	return nil
}
