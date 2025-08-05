package gpa

import "reflect"

// =====================================
// Entity Metadata
// =====================================

// EntityInfo contains metadata about an entity type
type EntityInfo struct {
	Name       string
	TableName  string
	Fields     []FieldInfo
	PrimaryKey []string
	Indexes    []IndexInfo
	Relations  []RelationInfo
}

// FieldInfo contains metadata about a field
type FieldInfo struct {
	Name            string
	Type            reflect.Type
	DatabaseType    string
	Tag             string
	IsPrimaryKey    bool
	IsNullable      bool
	IsAutoIncrement bool
	DefaultValue    interface{}
	MaxLength       int
	Precision       int
	Scale           int
}

// IndexInfo contains metadata about an index
type IndexInfo struct {
	Name     string
	Fields   []string
	IsUnique bool
	Type     IndexType
}

// RelationInfo contains metadata about rel
type RelationInfo struct {
	Name         string
	Type         RelationType
	TargetEntity string
	ForeignKey   string
	References   string
}

// =====================================
// Events and Hooks
// =====================================

// EventHook represents a hook that can be called during repository operations
type EventHook interface {
	// BeforeCreate is called before creating an entity
	BeforeCreate(ctx interface{}, entity interface{}) error

	// AfterCreate is called after creating an entity
	AfterCreate(ctx interface{}, entity interface{}) error

	// BeforeUpdate is called before updating an entity
	BeforeUpdate(ctx interface{}, entity interface{}) error

	// AfterUpdate is called after updating an entity
	AfterUpdate(ctx interface{}, entity interface{}) error

	// BeforeDelete is called before deleting an entity
	BeforeDelete(ctx interface{}, id interface{}) error

	// AfterDelete is called after deleting an entity
	AfterDelete(ctx interface{}, id interface{}) error
}
