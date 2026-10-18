package gpa_test

import (
	"testing"

	"github.com/lemmego/gpa"
)

// Compile guards for the Go examples on the docs site's database pages.
//
// These exist because three separate blocks documented an API that does not
// compile, and nothing noticed: a docs example is only prose until something
// builds it.

type docsOrder struct{ ID int }

// database/provider-registry.md — "Lifecycle Management".
//
// The page called all five of these as package-level functions. They are
// methods on *ProviderRegistry, reached through gpa.Registry(), and
// ListInstances/Remove/RemoveAll each report an error the page dropped.
func TestDocsProviderRegistryLifecycle(t *testing.T) {
	t.Skip("compile-only: RemoveAll would empty the shared registry")

	r := gpa.Registry()

	var types []string = r.ListTypes()
	_ = types

	instances, err := r.ListInstances("gorm")
	_, _ = instances, err

	err = r.Remove("gorm", "primary")
	_ = err

	err = r.RemoveAll()
	_ = err

	var health map[string]map[string]error = r.HealthCheck()
	_ = health
}

// database/query-system.md — "Full Builder API".
//
// The page listed qb.InnerJoin(table, on) and qb.LeftJoin(table, on), neither
// of which exists, and gave Join the wrong shape: it takes the join type
// first. InnerJoin and LeftJoin are package-level QueryOptions, not methods.
func TestDocsQueryBuilderAPI(t *testing.T) {
	qb := gpa.NewQueryBuilder[docsOrder]()

	qb.Where("total", gpa.OpGreaterThan, 100)
	qb.OrderBy("id", gpa.OrderDesc)
	qb.Limit(10)
	qb.Offset(0)
	qb.Select("id")
	qb.Join(gpa.JoinInner, "users", "users.id = orders.user_id")
	qb.GroupBy("user_id")
	qb.Having("total", gpa.OpGreaterThan, 0)
	qb.Distinct()
	qb.Lock(gpa.LockForUpdate)
	qb.Preload("User")

	if q := qb.Build(); q == nil {
		t.Fatal("Build returned nil")
	}
}

// database/query-system.md — "Subqueries".
//
// The page passed a SQL string to all three. They take a *Query, and
// CorrelatedSubQuery takes four arguments, not one.
func TestDocsSubqueries(t *testing.T) {
	highValue := gpa.NewQueryBuilder[docsOrder]().
		Select("user_id").
		Where("total", gpa.OpGreaterThan, 100).
		Build()

	for _, opt := range []gpa.QueryOption{
		gpa.InSubQuery("id", highValue),
		gpa.ExistsSubQuery(highValue),
		gpa.CorrelatedSubQuery("score", gpa.OpGreaterThan, highValue, "product_id"),
		gpa.InnerJoin("orders", "orders.user_id = users.id"),
		gpa.LeftJoin("comments", "comments.user_id = users.id"),
	} {
		if opt == nil {
			t.Fatal("nil QueryOption")
		}
	}
}
