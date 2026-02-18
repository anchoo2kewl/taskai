package schema

import (
	"time"

	"entgo.io/ent"
	"entgo.io/ent/schema/edge"
	"entgo.io/ent/schema/field"
	"entgo.io/ent/schema/index"
)

// YjsUpdate holds the schema definition for the YjsUpdate entity.
type YjsUpdate struct {
	ent.Schema
}

// Fields of the YjsUpdate.
func (YjsUpdate) Fields() []ent.Field {
	return []ent.Field{
		field.Int64("id"),
		field.Int64("page_id"),
		field.Bytes("update_data").NotEmpty(),
		field.Time("created_at").Default(time.Now).Immutable(),
		field.Int64("created_by").Optional().Nillable(),
	}
}

// Edges of the YjsUpdate.
func (YjsUpdate) Edges() []ent.Edge {
	return []ent.Edge{
		edge.From("page", WikiPage.Type).Ref("yjs_updates").Unique().Required().Field("page_id"),
		edge.From("creator", User.Type).Ref("yjs_updates").Unique().Field("created_by"),
	}
}

// Indexes of the YjsUpdate.
func (YjsUpdate) Indexes() []ent.Index {
	return []ent.Index{
		index.Fields("page_id"),
	}
}
