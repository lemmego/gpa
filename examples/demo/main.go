package main

import (
	"context"
	"github.com/lemmego/gpa"
	"github.com/lemmego/gpa/gpagorm"
	"log"
	"time"
)

// User represents a user entity with GORM-specific tags
type User struct {
	ID        uint      `gorm:"primaryKey"`
	Name      string    `gorm:"size:255;not null"`
	Email     string    `gorm:"uniqueIndex;size:255;not null"`
	Age       int       `gorm:"not null"`
	IsActive  bool      `gorm:"default:true"`
	CreatedAt time.Time `gorm:"autoCreateTime"`
	UpdatedAt time.Time `gorm:"autoUpdateTime"`
	Profile   *Profile  `gorm:"foreignKey:UserID"`
}

// Profile represents a user profile with a foreign key relationship
type Profile struct {
	ID       uint   `gorm:"primaryKey"`
	UserID   uint   `gorm:"not null;index"`
	Bio      string `gorm:"type:text"`
	Website  string `gorm:"size:255"`
	Location string `gorm:"size:100"`
}

func main() {
	config := gpa.Config{
		Driver:   "sqlite",
		Database: ":memory:",
		Options: map[string]interface{}{
			"gorm": map[string]interface{}{
				"log_level":      "info",
				"singular_table": false,
			},
		},
	}

	// Create type-safe providers
	userProvider, err := gpagorm.NewTypeSafeProvider[User](config)
	if err != nil {
		log.Fatalf("Failed to create user provider: %v", err)
	}
	defer userProvider.Close()

	repo := userProvider.Repository()

	if uRepo, ok := repo.(gpa.MigratableRepository[User]); ok {
		uRepo.MigrateTable(context.Background())
	}

	u := &User{Name: "John Doe"}
	err = repo.Create(context.Background(), u)

	println(u.ID)

	if err != nil {
		log.Fatalf("Failed to create user: %v", err)
	}
}
