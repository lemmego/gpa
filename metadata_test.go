package gpa

import (
	"reflect"
	"testing"
)

func TestEntityInfo(t *testing.T) {
	info := EntityInfo{
		Name:      "User",
		TableName: "users",
		Fields: []FieldInfo{
			{
				Name:            "ID",
				Type:            reflect.TypeOf(int64(0)),
				DatabaseType:    "bigint",
				Tag:             `json:"id" db:"id"`,
				IsPrimaryKey:    true,
				IsAutoIncrement: true,
			},
			{
				Name:         "Email",
				Type:         reflect.TypeOf(""),
				DatabaseType: "varchar",
				Tag:          `json:"email" db:"email"`,
				MaxLength:    255,
			},
		},
		PrimaryKey: []string{"ID"},
		Indexes: []IndexInfo{
			{
				Name:   "idx_users_email",
				Fields: []string{"email"},
				IsUnique: true,
				Type:   IndexTypeUnique,
			},
		},
	}

	if info.Name != "User" {
		t.Errorf("Expected name 'User', got '%s'", info.Name)
	}
	if info.TableName != "users" {
		t.Errorf("Expected table name 'users', got '%s'", info.TableName)
	}
	if len(info.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(info.Fields))
	}
	if len(info.PrimaryKey) != 1 {
		t.Errorf("Expected 1 primary key field, got %d", len(info.PrimaryKey))
	}
	if info.PrimaryKey[0] != "ID" {
		t.Errorf("Expected primary key 'ID', got '%s'", info.PrimaryKey[0])
	}
}

func TestFieldInfo(t *testing.T) {
	field := FieldInfo{
		Name:            "CreatedAt",
		Type:            reflect.TypeOf(""),
		DatabaseType:    "timestamp",
		Tag:             `json:"created_at" db:"created_at"`,
		IsPrimaryKey:    false,
		IsNullable:      false,
		IsAutoIncrement: false,
		DefaultValue:    "CURRENT_TIMESTAMP",
		MaxLength:       0,
		Precision:       0,
		Scale:           0,
	}

	if field.Name != "CreatedAt" {
		t.Errorf("Expected name 'CreatedAt', got '%s'", field.Name)
	}
	if field.DatabaseType != "timestamp" {
		t.Errorf("Expected database type 'timestamp', got '%s'", field.DatabaseType)
	}
	if field.IsPrimaryKey {
		t.Error("Expected field not to be primary key")
	}
	if field.IsNullable {
		t.Error("Expected field not to be nullable")
	}
	if field.DefaultValue != "CURRENT_TIMESTAMP" {
		t.Errorf("Expected default value 'CURRENT_TIMESTAMP', got '%v'", field.DefaultValue)
	}
}

func TestIndexInfo(t *testing.T) {
	index := IndexInfo{
		Name:   "idx_users_email_status",
		Fields: []string{"email", "status"},
		IsUnique: false,
		Type:   IndexTypeComposite,
	}

	if index.Name != "idx_users_email_status" {
		t.Errorf("Expected name 'idx_users_email_status', got '%s'", index.Name)
	}
	if len(index.Fields) != 2 {
		t.Errorf("Expected 2 fields, got %d", len(index.Fields))
	}
	if index.Fields[0] != "email" || index.Fields[1] != "status" {
		t.Errorf("Expected fields [email, status], got %v", index.Fields)
	}
	if index.IsUnique {
		t.Error("Expected index not to be unique")
	}
	if index.Type != IndexTypeComposite {
		t.Errorf("Expected type composite, got %s", index.Type)
	}
}

func TestRelationInfo(t *testing.T) {
	relation := RelationInfo{
		Name:         "Posts",
		Type:         RelationOneToMany,
		TargetEntity: "Post",
		ForeignKey:   "user_id",
		References:   "id",
	}

	if relation.Name != "Posts" {
		t.Errorf("Expected name 'Posts', got '%s'", relation.Name)
	}
	if relation.Type != RelationOneToMany {
		t.Errorf("Expected type one-to-many, got %s", relation.Type)
	}
	if relation.TargetEntity != "Post" {
		t.Errorf("Expected target entity 'Post', got '%s'", relation.TargetEntity)
	}
	if relation.ForeignKey != "user_id" {
		t.Errorf("Expected foreign key 'user_id', got '%s'", relation.ForeignKey)
	}
	if relation.References != "id" {
		t.Errorf("Expected references 'id', got '%s'", relation.References)
	}
}

func TestManyToManyRelation(t *testing.T) {
	relation := RelationInfo{
		Name:         "Roles",
		Type:         RelationManyToMany,
		TargetEntity: "Role",
		ForeignKey:   "user_id",
		References:   "id",
	}

	if relation.Type != RelationManyToMany {
		t.Errorf("Expected type many-to-many, got %s", relation.Type)
	}
	if relation.TargetEntity != "Role" {
		t.Errorf("Expected target entity 'Role', got '%s'", relation.TargetEntity)
	}
	if relation.ForeignKey != "user_id" {
		t.Errorf("Expected foreign key 'user_id', got '%s'", relation.ForeignKey)
	}
	if relation.References != "id" {
		t.Errorf("Expected references 'id', got '%s'", relation.References)
	}
}


func TestCompleteEntityInfo(t *testing.T) {
	// Test a complete entity with all metadata
	info := EntityInfo{
		Name:      "Post",
		TableName: "posts",
		Fields: []FieldInfo{
			{
				Name:            "ID",
				Type:            reflect.TypeOf(int64(0)),
				DatabaseType:    "bigint",
				IsPrimaryKey:    true,
				IsAutoIncrement: true,
			},
			{
				Name:         "Title",
				Type:         reflect.TypeOf(""),
				DatabaseType: "varchar",
				MaxLength:    255,
				IsNullable:   false,
			},
			{
				Name:         "Content",
				Type:         reflect.TypeOf(""),
				DatabaseType: "text",
				IsNullable:   true,
			},
			{
				Name:         "UserID",
				Type:         reflect.TypeOf(int64(0)),
				DatabaseType: "bigint",
				IsNullable:   false,
			},
			{
				Name:         "Score",
				Type:         reflect.TypeOf(float64(0)),
				DatabaseType: "decimal",
				Precision:    10,
				Scale:        2,
				DefaultValue: 0.0,
			},
		},
		PrimaryKey: []string{"ID"},
		Indexes: []IndexInfo{
			{
				Name:   "idx_posts_user_id",
				Fields: []string{"user_id"},
				Type:   IndexTypeStandard,
			},
			{
				Name:   "idx_posts_title",
				Fields: []string{"title"},
				Type:   IndexTypeFullText,
			},
		},
		Relations: []RelationInfo{
			{
				Name:          "User",
				Type:          RelationManyToOne,
				TargetEntity: "User",
				ForeignKey:   "user_id",
				References:   "user_id",
			},
			{
				Name:          "Comments",
				Type:          RelationOneToMany,
				TargetEntity: "Comment",
				ForeignKey:   "post_id",
				References:   "id",
			},
		},
	}

	// Test basic info
	if info.Name != "Post" {
		t.Errorf("Expected entity name 'Post', got '%s'", info.Name)
	}
	if len(info.Fields) != 5 {
		t.Errorf("Expected 5 fields, got %d", len(info.Fields))
	}
	if len(info.Indexes) != 2 {
		t.Errorf("Expected 2 indexes, got %d", len(info.Indexes))
	}
	if len(info.Relations) != 2 {
		t.Errorf("Expected 2 relations, got %d", len(info.Relations))
	}

	// Test field details
	scoreField := info.Fields[4]
	if scoreField.Precision != 10 {
		t.Errorf("Expected precision 10, got %d", scoreField.Precision)
	}
	if scoreField.Scale != 2 {
		t.Errorf("Expected scale 2, got %d", scoreField.Scale)
	}

	// Test index types
	if info.Indexes[0].Type != IndexTypeStandard {
		t.Errorf("Expected standard index, got %s", info.Indexes[0].Type)
	}
	if info.Indexes[1].Type != IndexTypeFullText {
		t.Errorf("Expected fulltext index, got %s", info.Indexes[1].Type)
	}

	// Test relations
	userRelation := info.Relations[0]
	if userRelation.Type != RelationManyToOne {
		t.Errorf("Expected many-to-one relation, got %s", userRelation.Type)
	}
	
	commentsRelation := info.Relations[1]
	if commentsRelation.Type != RelationOneToMany {
		t.Errorf("Expected one-to-many relation, got %s", commentsRelation.Type)
	}
}

func TestFieldInfoDefaults(t *testing.T) {
	// Test field with minimal information
	field := FieldInfo{
		Name: "SimpleField",
		Type: reflect.TypeOf(""),
	}

	if field.IsPrimaryKey {
		t.Error("Expected field not to be primary key by default")
	}
	if field.IsAutoIncrement {
		t.Error("Expected field not to be auto increment by default")
	}
	// Field uniqueness is tested through indexes, not individual fields
	if field.IsNullable {
		t.Error("Expected field to not be nullable by default")
	}
	if field.MaxLength != 0 {
		t.Error("Expected max length to be 0 by default")
	}
}

func TestIndexInfoDefaults(t *testing.T) {
	// Test index with minimal information
	index := IndexInfo{
		Name:   "simple_index",
		Fields: []string{"field1"},
	}

	if index.IsUnique {
		t.Error("Expected index not to be unique by default")
	}
	if index.Type != "" {
		t.Error("Expected empty index type by default")
	}
}