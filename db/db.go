package db

import (
	"chat_app/config"
	"chat_app/entity"
	"fmt"
	"log"

	"go.uber.org/fx"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func NewDB(config *config.Config) *gorm.DB {
	// Construct the DSN
	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		config.DBHost, config.DBUser, config.DBPassword, config.DBName, config.DBPort)

	// Log the DSN for debugging (avoid logging sensitive info in production)
	log.Printf("Connecting to database with DSN: host=%s user=%s dbname=%s port=%s",
		config.DBHost, config.DBUser, config.DBName, config.DBPort)

	// Open the database connection
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatalf("Failed to connect to database: %v", err)
	}

	// Run migrations
	if err := db.AutoMigrate(&entity.User{}, &entity.Message{}, &entity.Group{}, &entity.GroupMember{}, &entity.BlockedUser{}); err != nil {
		log.Fatalf("Failed to run migrations: %v", err)
	}

	log.Println("Database connection established and migrations completed")
	return db
}

var Module = fx.Provide(NewDB)
