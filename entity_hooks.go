package gpa

import "context"

// =====================================
// Entity Hook Interfaces
// =====================================

// BeforeCreateHook is called before creating an entity
type BeforeCreateHook interface {
	BeforeCreate(ctx context.Context) error
}

// AfterCreateHook is called after successfully creating an entity
type AfterCreateHook interface {
	AfterCreate(ctx context.Context) error
}

// BeforeUpdateHook is called before updating an entity
type BeforeUpdateHook interface {
	BeforeUpdate(ctx context.Context) error
}

// AfterUpdateHook is called after successfully updating an entity
type AfterUpdateHook interface {
	AfterUpdate(ctx context.Context) error
}

// BeforeDeleteHook is called before deleting an entity
type BeforeDeleteHook interface {
	BeforeDelete(ctx context.Context) error
}

// AfterDeleteHook is called after successfully deleting an entity
type AfterDeleteHook interface {
	AfterDelete(ctx context.Context) error
}

// BeforeFindHook is called before finding an entity
type BeforeFindHook interface {
	BeforeFind(ctx context.Context) error
}

// AfterFindHook is called after successfully finding an entity
type AfterFindHook interface {
	AfterFind(ctx context.Context) error
}

// ValidationHook is called to validate an entity before create/update
type ValidationHook interface {
	Validate(ctx context.Context) error
}

