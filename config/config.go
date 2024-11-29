package config

import (
	"fmt"
	"os"

	"github.com/farellandr/spoticket/internal/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type Config struct {
	DBHost     string
	DBPort     string
	DBUser     string
	DBPassword string
	DBName     string
}

func LoadConfig() (*Config, error) {
	return &Config{
		DBHost:     os.Getenv("DB_HOST"),
		DBPort:     os.Getenv("DB_PORT"),
		DBUser:     os.Getenv("DB_USER"),
		DBPassword: os.Getenv("DB_PASSWORD"),
		DBName:     os.Getenv("DB_NAME"),
	}, nil
}

func InitDatabase(cfg *Config) (*gorm.DB, error) {
	dsn := fmt.Sprintf(
		"host=%s user=%s password=%s dbname=%s port=%s sslmode=disable TimeZone=UTC",
		cfg.DBHost, cfg.DBUser, cfg.DBPassword, cfg.DBName, cfg.DBPort,
	)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&models.Role{}, &models.User{}, &models.Event{}, &models.Ticket{}, &models.Purchase{}, &models.Payment{})
	if err != nil {
		return nil, err
	}

	seedRoles(db)

	return db, nil
}

func seedRoles(db *gorm.DB) {
	roles := []models.Role{
		{Name: "organizer"},
		{Name: "attendee"},
		{Name: "admin"},
	}

	for _, role := range roles {
		var existingRole models.Role
		result := db.Where("name = ?", role.Name).First(&existingRole)
		if result.Error != nil {
			db.Create(&role)
		}
	}
}
