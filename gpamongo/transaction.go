package gpamongo

import (
	"context"

	"github.com/lemmego/gpa"
	"go.mongodb.org/mongo-driver/mongo"
)

// =====================================
// Transaction Implementation
// =====================================

// Transaction implements gpa.Transaction for MongoDB
type Transaction struct {
	*Repository
	session mongo.Session
}

// Commit commits the transaction
func (t *Transaction) Commit() error {
	// MongoDB transactions are committed in the Transaction method
	// This is a no-op as the commit happens automatically
	return nil
}

// Rollback rolls back the transaction
func (t *Transaction) Rollback() error {
	// MongoDB transactions are rolled back automatically on error
	// This is a no-op as the rollback happens automatically
	return nil
}

// Create creates a new entity within the transaction
func (t *Transaction) Create(ctx context.Context, entity interface{}) error {
	// Use the session context for the transaction
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.Create(sc, entity)
	}
	return t.Repository.Create(ctx, entity)
}

// CreateBatch creates multiple entities within the transaction
func (t *Transaction) CreateBatch(ctx context.Context, entities interface{}) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.CreateBatch(sc, entities)
	}
	return t.Repository.CreateBatch(ctx, entities)
}

// FindByID finds an entity by ID within the transaction
func (t *Transaction) FindByID(ctx context.Context, id interface{}, dest interface{}) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.FindByID(sc, id, dest)
	}
	return t.Repository.FindByID(ctx, id, dest)
}

// FindAll finds all entities within the transaction
func (t *Transaction) FindAll(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.FindAll(sc, dest, opts...)
	}
	return t.Repository.FindAll(ctx, dest, opts...)
}

// Update updates an entity within the transaction
func (t *Transaction) Update(ctx context.Context, entity interface{}) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.Update(sc, entity)
	}
	return t.Repository.Update(ctx, entity)
}

// UpdatePartial updates specific fields within the transaction
func (t *Transaction) UpdatePartial(ctx context.Context, id interface{}, updates map[string]interface{}) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.UpdatePartial(sc, id, updates)
	}
	return t.Repository.UpdatePartial(ctx, id, updates)
}

// Delete deletes an entity within the transaction
func (t *Transaction) Delete(ctx context.Context, id interface{}) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.Delete(sc, id)
	}
	return t.Repository.Delete(ctx, id)
}

// DeleteByCondition deletes entities by condition within the transaction
func (t *Transaction) DeleteByCondition(ctx context.Context, condition gpa.Condition) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.DeleteByCondition(sc, condition)
	}
	return t.Repository.DeleteByCondition(ctx, condition)
}

// Query executes a query within the transaction
func (t *Transaction) Query(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.Query(sc, dest, opts...)
	}
	return t.Repository.Query(ctx, dest, opts...)
}

// QueryOne executes a query for one result within the transaction
func (t *Transaction) QueryOne(ctx context.Context, dest interface{}, opts ...gpa.QueryOption) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.QueryOne(sc, dest, opts...)
	}
	return t.Repository.QueryOne(ctx, dest, opts...)
}

// Count counts entities within the transaction
func (t *Transaction) Count(ctx context.Context, opts ...gpa.QueryOption) (int64, error) {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.Count(sc, opts...)
	}
	return t.Repository.Count(ctx, opts...)
}

// Exists checks if entities exist within the transaction
func (t *Transaction) Exists(ctx context.Context, opts ...gpa.QueryOption) (bool, error) {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.Exists(sc, opts...)
	}
	return t.Repository.Exists(ctx, opts...)
}

// Transaction executes a nested transaction (not supported in MongoDB)
func (t *Transaction) Transaction(ctx context.Context, fn gpa.TransactionFunc) error {
	return gpa.GPAError{
		Type:    gpa.ErrorTypeUnsupported,
		Message: "nested transactions are not supported in MongoDB",
	}
}

// RawQuery executes a raw query within the transaction
func (t *Transaction) RawQuery(ctx context.Context, query string, args []interface{}, dest interface{}) error {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.RawQuery(sc, query, args, dest)
	}
	return t.Repository.RawQuery(ctx, query, args, dest)
}

// RawExec executes a raw operation within the transaction
func (t *Transaction) RawExec(ctx context.Context, query string, args []interface{}) (gpa.Result, error) {
	if sc, ok := ctx.(mongo.SessionContext); ok {
		return t.Repository.RawExec(sc, query, args)
	}
	return t.Repository.RawExec(ctx, query, args)
}

// GetEntityInfo returns entity info within the transaction
func (t *Transaction) GetEntityInfo(entity interface{}) (*gpa.EntityInfo, error) {
	return t.Repository.GetEntityInfo(entity)
}

// Close closes the transaction (no-op)
func (t *Transaction) Close() error {
	return nil
}

// =====================================
// Result Implementation
// =====================================

// Result implements gpa.Result for MongoDB operations
type Result struct {
	lastInsertId int64
	rowsAffected int64
}

// LastInsertId returns the last insert ID (not applicable for MongoDB)
func (r *Result) LastInsertId() (int64, error) {
	return r.lastInsertId, nil
}

// RowsAffected returns the number of affected rows/documents
func (r *Result) RowsAffected() (int64, error) {
	return r.rowsAffected, nil
}