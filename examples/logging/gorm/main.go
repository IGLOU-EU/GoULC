// build: go run -tags=gorm main.go
package main

import (
	"gitlab.com/iglou.eu/goulc/logging"
	"gitlab.com/iglou.eu/goulc/logging/model"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// User is a simple model for demonstration
type User struct {
	gorm.Model
	Name  string
	Email string
}

func main() {
	// Create a new default logger
	logger, err := logging.New("", &model.Config{
		Level:   "DEBUG",
		Colored: true,
	})
	if err != nil {
		panic(err)
	}

	// Create GORM logger from our custom logger
	gormLogger := logging.NewGormLogger(logger.WithGroup("gorm"))

	// Initialize GORM with SQLite and our custom logger
	db, err := gorm.Open(sqlite.Open("test.db"), &gorm.Config{
		Logger: gormLogger,
	})
	if err != nil {
		logger.Error("Failed to connect to database", "error", err)
	}

	// Auto migrate the schema
	err = db.AutoMigrate(&User{})
	if err != nil {
		logger.Error("Failed to migrate database", "error", err)
	}

	// Create a user - this will generate log entries
	user := User{
		Name:  "John Doe",
		Email: "john@example.com",
	}
	result := db.Create(&user)
	if result.Error != nil {
		logger.Error("Failed to create user", "error", result.Error)
	}

	// Find the user - this will also generate log entries
	var foundUser User

	result = db.First(&foundUser, 1000)
	if result.Error != nil {
		logger.Error("Failed to find user", "error", result.Error)
	}

	result = db.First(&foundUser, user.ID)
	if result.Error != nil {
		logger.Error("Failed to find user", "error", result.Error)
	}

	// Update the user
	result = db.Model(&foundUser).Update("Name", "John Smith")
	if result.Error != nil {
		logger.Error("Failed to update user", "error", result.Error)
	}

	// Delete the user
	result = db.Delete(&foundUser)
	if result.Error != nil {
		logger.Error("Failed to delete user", "error", result.Error)
	}

	logger.Info("Example completed successfully")
}
