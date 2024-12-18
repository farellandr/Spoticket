package config

import (
	"fmt"
	"os"

	"github.com/farellandr/spoticket/internal/models"
	"github.com/xendit/xendit-go/v6"
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

type XenditConfig struct {
	SecretKey string
	PublicKey string
}

func LoadXenditConfig() (*XenditConfig, error) {
	return &XenditConfig{
		SecretKey: os.Getenv("XENDIT_SECRET_KEY"),
		PublicKey: os.Getenv("XENDIT_PUBLIC_KEY"),
	}, nil
}

func InitXenditClient(config *XenditConfig) (*xendit.APIClient, error) {
	client := xendit.NewClient(config.SecretKey)

	return client, nil
}

func enableUUIDExtension(db *gorm.DB) error {
	return db.Exec("CREATE EXTENSION IF NOT EXISTS \"uuid-ossp\"").Error
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

	if err := enableUUIDExtension(db); err != nil {
		return nil, err
	}

	err = db.AutoMigrate(&models.Role{}, &models.User{}, &models.Event{}, &models.Ticket{}, &models.Purchase{}, &models.Payment{}, &models.Category{}, &models.Coupon{}, &models.UserCoupon{})
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
