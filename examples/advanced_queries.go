package examples

import (
	"context"
	"fmt"
	"log"

	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpagorm"
)

// Order represents an e-commerce order
type Order struct {
	ID         uint      `gorm:"primaryKey"`
	CustomerID uint      `gorm:"not null;index"`
	Status     string    `gorm:"size:50;not null;index"`
	Total      float64   `gorm:"not null"`
	Items      int       `gorm:"not null"`
	CreatedAt  string    `gorm:"type:datetime;not null"`
	Customer   *Customer `gorm:"foreignKey:CustomerID"`
}

// Customer represents a customer
type Customer struct {
	ID       uint   `gorm:"primaryKey"`
	Name     string `gorm:"size:255;not null"`
	Email    string `gorm:"uniqueIndex;size:255;not null"`
	City     string `gorm:"size:100;index"`
	Country  string `gorm:"size:100;index"`
	Premium  bool   `gorm:"default:false;index"`
}

func RunAdvancedQueries() {
	fmt.Println("🔍 Advanced Queries Example")
	fmt.Println("Demonstrating complex query patterns and relationships")

	// Setup database
	config := gpa.Config{
		Driver:   "sqlite",
		Database: "advanced_example.db",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level": "info",
			},
		},
	}

	// Create providers for both entities
	customerProvider, err := gpagorm.NewTypeSafeProvider[Customer](config)
	if err != nil {
		log.Fatalf("Failed to create customer provider: %v", err)
	}
	defer customerProvider.Close()

	orderProvider, err := gpagorm.NewTypeSafeProvider[Order](config)
	if err != nil {
		log.Fatalf("Failed to create order provider: %v", err)
	}
	defer orderProvider.Close()

	customerRepo := customerProvider.Repository()
	orderRepo := orderProvider.Repository()

	// Migrate tables
	ctx := context.Background()
	if migratableCustomerRepo, ok := customerRepo.(gpa.MigratableRepository[Customer]); ok {
		err = migratableCustomerRepo.MigrateTable(ctx)
		if err != nil {
			log.Fatalf("Failed to migrate customers table: %v", err)
		}
	}

	if migratableOrderRepo, ok := orderRepo.(gpa.MigratableRepository[Order]); ok {
		err = migratableOrderRepo.MigrateTable(ctx)
		if err != nil {
			log.Fatalf("Failed to migrate orders table: %v", err)
		}
	}

	fmt.Println("✓ Tables migrated successfully")

	// Setup test data
	setupTestData(ctx, customerRepo, orderRepo)

	// ============================================
	// Basic Query Examples
	// ============================================
	fmt.Println("\n=== Basic Query Examples ===")

	// Single condition queries
	premiumCustomers, err := customerRepo.Query(ctx,
		gpa.Where("premium", gpa.OpEqual, true),
	)
	if err != nil {
		log.Printf("Failed to find premium customers: %v", err)
	} else {
		fmt.Printf("✓ Found %d premium customers\n", len(premiumCustomers))
	}

	// Multiple condition queries (AND)
	usCustomers, err := customerRepo.Query(ctx,
		gpa.Where("country", gpa.OpEqual, "USA"),
		gpa.Where("premium", gpa.OpEqual, true),
	)
	if err != nil {
		log.Printf("Failed to find US premium customers: %v", err)
	} else {
		fmt.Printf("✓ Found %d premium customers in USA\n", len(usCustomers))
	}

	// ============================================
	// Comparison Operators
	// ============================================
	fmt.Println("\n=== Comparison Operators ===")

	// Greater than
	highValueOrders, err := orderRepo.Query(ctx,
		gpa.Where("total", gpa.OpGreaterThan, 500.0),
		gpa.OrderBy("total", gpa.OrderDesc),
	)
	if err != nil {
		log.Printf("Failed to find high value orders: %v", err)
	} else {
		fmt.Printf("✓ Found %d orders > $500\n", len(highValueOrders))
		for _, order := range highValueOrders {
			fmt.Printf("  Order #%d: $%.2f\n", order.ID, order.Total)
		}
	}

	// Between (using two conditions)
	mediumOrders, err := orderRepo.Query(ctx,
		gpa.Where("total", gpa.OpGreaterThanOrEqual, 100.0),
		gpa.Where("total", gpa.OpLessThanOrEqual, 500.0),
		gpa.OrderBy("total", gpa.OrderAsc),
	)
	if err != nil {
		log.Printf("Failed to find medium value orders: %v", err)
	} else {
		fmt.Printf("✓ Found %d orders between $100-$500\n", len(mediumOrders))
	}

	// ============================================
	// IN and LIKE Operators
	// ============================================
	fmt.Println("\n=== IN and LIKE Operators ===")

	// IN operator
	specificStatuses := []interface{}{"pending", "shipped", "delivered"}
	activeOrders, err := orderRepo.Query(ctx,
		gpa.WhereIn("status", specificStatuses),
		gpa.OrderBy("created_at", gpa.OrderDesc),
	)
	if err != nil {
		log.Printf("Failed to find active orders: %v", err)
	} else {
		fmt.Printf("✓ Found %d orders with active statuses\n", len(activeOrders))
	}

	// LIKE operator
	johnCustomers, err := customerRepo.Query(ctx,
		gpa.WhereLike("name", "John%"),
	)
	if err != nil {
		log.Printf("Failed to find Johns: %v", err)
	} else {
		fmt.Printf("✓ Found %d customers named John*\n", len(johnCustomers))
	}

	// ============================================
	// NULL Checks
	// ============================================
	fmt.Println("\n=== NULL Checks ===")

	// NOT NULL (all customers should have emails)
	customersWithEmail, err := customerRepo.Query(ctx,
		gpa.WhereNotNull("email"),
	)
	if err != nil {
		log.Printf("Failed to find customers with email: %v", err)
	} else {
		fmt.Printf("✓ Found %d customers with email addresses\n", len(customersWithEmail))
	}

	// ============================================
	// Sorting and Limiting
	// ============================================
	fmt.Println("\n=== Sorting and Limiting ===")

	// Top 5 most expensive orders
	topOrders, err := orderRepo.Query(ctx,
		gpa.OrderBy("total", gpa.OrderDesc),
		gpa.Limit(5),
	)
	if err != nil {
		log.Printf("Failed to find top orders: %v", err)
	} else {
		fmt.Printf("✓ Top 5 most expensive orders:\n")
		for i, order := range topOrders {
			fmt.Printf("  %d. Order #%d: $%.2f (%s)\n", i+1, order.ID, order.Total, order.Status)
		}
	}

	// Pagination example
	page2Orders, err := orderRepo.Query(ctx,
		gpa.OrderBy("id", gpa.OrderAsc),
		gpa.Limit(3),
		gpa.Offset(3), // Skip first 3 (page 1)
	)
	if err != nil {
		log.Printf("Failed to get page 2: %v", err)
	} else {
		fmt.Printf("✓ Page 2 (orders 4-6): %d orders\n", len(page2Orders))
	}

	// ============================================
	// Field Selection
	// ============================================
	fmt.Println("\n=== Field Selection ===")

	// Select only specific fields
	customerNames, err := customerRepo.Query(ctx,
		gpa.Select("id", "name", "email"),
		gpa.Where("country", gpa.OpEqual, "USA"),
		gpa.Limit(3),
	)
	if err != nil {
		log.Printf("Failed to select customer fields: %v", err)
	} else {
		fmt.Printf("✓ Selected fields for %d US customers:\n", len(customerNames))
		for _, customer := range customerNames {
			fmt.Printf("  %d: %s (%s)\n", customer.ID, customer.Name, customer.Email)
		}
	}

	// ============================================
	// Aggregation and Counting
	// ============================================
	fmt.Println("\n=== Aggregation and Counting ===")

	// Count by status
	statuses := []string{"pending", "processing", "shipped", "delivered", "cancelled"}
	for _, status := range statuses {
		count, err := orderRepo.Count(ctx, gpa.Where("status", gpa.OpEqual, status))
		if err != nil {
			log.Printf("Failed to count %s orders: %v", status, err)
		} else {
			fmt.Printf("✓ %s orders: %d\n", status, count)
		}
	}

	// Count premium vs regular customers
	premiumCount, _ := customerRepo.Count(ctx, gpa.Where("premium", gpa.OpEqual, true))
	totalCustomers, _ := customerRepo.Count(ctx)
	regularCount := totalCustomers - premiumCount
	fmt.Printf("✓ Customer breakdown: %d premium, %d regular (total: %d)\n", 
		premiumCount, regularCount, totalCustomers)

	// ============================================
	// Complex Queries
	// ============================================
	fmt.Println("\n=== Complex Queries ===")

	// High-value orders from premium customers (requires join in real scenario)
	// For this example, we'll find high-value orders and then check customer status
	expensiveOrders, err := orderRepo.Query(ctx,
		gpa.Where("total", gpa.OpGreaterThan, 300.0),
		gpa.OrderBy("total", gpa.OrderDesc),
	)
	if err != nil {
		log.Printf("Failed to find expensive orders: %v", err)
	} else {
		fmt.Printf("✓ Analyzing %d expensive orders:\n", len(expensiveOrders))
		
		premiumOrderCount := 0
		for _, order := range expensiveOrders {
			customer, err := customerRepo.FindByID(ctx, order.CustomerID)
			if err == nil && customer.Premium {
				premiumOrderCount++
			}
		}
		fmt.Printf("  %d of these are from premium customers\n", premiumOrderCount)
	}

	// ============================================
	// Raw SQL Queries (Advanced)
	// ============================================
	fmt.Println("\n=== Raw SQL Queries ===")

	if sqlRepo, ok := orderRepo.(gpa.SQLRepository[Order]); ok {
		// Complex aggregation query
		avgOrderValues, err := sqlRepo.FindBySQL(ctx, 
			`SELECT status, COUNT(*) as count, AVG(total) as avg_total, SUM(total) as total_value 
			 FROM orders 
			 GROUP BY status 
			 ORDER BY avg_total DESC`, 
			[]interface{}{})
		if err != nil {
			log.Printf("Failed to execute aggregation query: %v", err)
		} else {
			fmt.Printf("✓ Order statistics by status:\n")
			// Note: In a real scenario, you'd create a separate struct for aggregation results
			fmt.Printf("  Found %d status groups\n", len(avgOrderValues))
		}

		// Custom join query (if relationships were properly set up)
		highValueCustomerOrders, err := sqlRepo.FindBySQL(ctx,
			`SELECT o.* FROM orders o 
			 JOIN customers c ON o.customer_id = c.id 
			 WHERE c.premium = ? AND o.total > ?
			 ORDER BY o.total DESC
			 LIMIT 5`,
			[]interface{}{true, 200.0})
		if err != nil {
			log.Printf("Failed to execute join query: %v", err)
		} else {
			fmt.Printf("✓ Found %d high-value orders from premium customers\n", len(highValueCustomerOrders))
		}
	}

	// ============================================
	// Performance Queries
	// ============================================
	fmt.Println("\n=== Performance Examples ===")

	// Existence check (more efficient than counting)
	hasHighValueOrders, err := orderRepo.Exists(ctx, 
		gpa.Where("total", gpa.OpGreaterThan, 1000.0))
	if err != nil {
		log.Printf("Failed to check for high-value orders: %v", err)
	} else {
		fmt.Printf("✓ Has orders > $1000: %t\n", hasHighValueOrders)
	}

	// Query with distinct (if supported by provider)
	distinctStatuses, err := orderRepo.Query(ctx,
		gpa.Select("status"),
		gpa.Distinct(),
	)
	if err != nil {
		log.Printf("Failed to get distinct statuses: %v", err)
	} else {
		fmt.Printf("✓ Found %d distinct order statuses\n", len(distinctStatuses))
	}

	fmt.Println("\n🎉 Advanced queries example completed!")
}

func setupTestData(ctx context.Context, customerRepo gpa.Repository[Customer], orderRepo gpa.Repository[Order]) {
	fmt.Println("\n=== Setting up test data ===")

	// Create customers
	customers := []*Customer{
		{Name: "John Smith", Email: "john.smith@email.com", City: "New York", Country: "USA", Premium: true},
		{Name: "Jane Doe", Email: "jane.doe@email.com", City: "Los Angeles", Country: "USA", Premium: false},
		{Name: "Bob Johnson", Email: "bob.johnson@email.com", City: "Chicago", Country: "USA", Premium: true},
		{Name: "Alice Brown", Email: "alice.brown@email.com", City: "Toronto", Country: "Canada", Premium: false},
		{Name: "Charlie Wilson", Email: "charlie.wilson@email.com", City: "London", Country: "UK", Premium: true},
		{Name: "Diana Davis", Email: "diana.davis@email.com", City: "Paris", Country: "France", Premium: false},
		{Name: "John Anderson", Email: "john.anderson@email.com", City: "Berlin", Country: "Germany", Premium: true},
	}

	err := customerRepo.CreateBatch(ctx, customers)
	if err != nil {
		log.Printf("Failed to create customers: %v", err)
		return
	}

	// Create orders
	orders := []*Order{
		{CustomerID: 1, Status: "delivered", Total: 299.99, Items: 2, CreatedAt: "2024-01-15 10:30:00"},
		{CustomerID: 1, Status: "shipped", Total: 599.99, Items: 1, CreatedAt: "2024-01-20 14:15:00"},
		{CustomerID: 2, Status: "pending", Total: 79.99, Items: 3, CreatedAt: "2024-01-22 09:00:00"},
		{CustomerID: 2, Status: "processing", Total: 149.99, Items: 1, CreatedAt: "2024-01-23 16:45:00"},
		{CustomerID: 3, Status: "delivered", Total: 899.99, Items: 1, CreatedAt: "2024-01-18 11:20:00"},
		{CustomerID: 3, Status: "cancelled", Total: 199.99, Items: 2, CreatedAt: "2024-01-25 13:30:00"},
		{CustomerID: 4, Status: "shipped", Total: 349.99, Items: 4, CreatedAt: "2024-01-19 15:10:00"},
		{CustomerID: 4, Status: "delivered", Total: 99.99, Items: 2, CreatedAt: "2024-01-16 08:45:00"},
		{CustomerID: 5, Status: "processing", Total: 1299.99, Items: 1, CreatedAt: "2024-01-24 12:00:00"},
		{CustomerID: 5, Status: "pending", Total: 449.99, Items: 3, CreatedAt: "2024-01-26 17:20:00"},
		{CustomerID: 6, Status: "shipped", Total: 199.99, Items: 2, CreatedAt: "2024-01-21 10:15:00"},
		{CustomerID: 7, Status: "delivered", Total: 799.99, Items: 1, CreatedAt: "2024-01-17 14:30:00"},
	}

	err = orderRepo.CreateBatch(ctx, orders)
	if err != nil {
		log.Printf("Failed to create orders: %v", err)
		return
	}

	fmt.Printf("✓ Created %d customers and %d orders\n", len(customers), len(orders))
}