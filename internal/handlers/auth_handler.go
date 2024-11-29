package handlers

import (
	"net/http"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"

	"github.com/farellandr/spoticket/internal/helpers"
	"github.com/farellandr/spoticket/internal/models"
)

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=6"`
	RoleName string `json:"role_name" binding:"required"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

func Register(c *gin.Context) {
	var req RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Please check your fields.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var role models.Role
	if err := gormDB.Where("name = ?", req.RoleName).First(&role).Error; err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid role.")
		return
	}

	var existingUser models.User
	if result := gormDB.Where("email = ?", req.Email).First(&existingUser); result.Error == nil {
		helpers.RespondWithError(c, http.StatusConflict, "User already exists.")
		return
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to hash the password.")
		return
	}

	user := models.User{
		ID:       uuid.New(),
		Email:    req.Email,
		Password: string(hashedPassword),
		RoleID:   role.ID,
	}

	if err := gormDB.Create(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to create user.")
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully."})
}

func Login(c *gin.Context) {
	var req LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		helpers.RespondWithError(c, http.StatusBadRequest, "Invalid input. Please check your fields.")
		return
	}

	db, exists := c.Get("db")
	if !exists {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Database connection not found.")
		return
	}
	gormDB := db.(*gorm.DB)

	var user models.User
	if err := gormDB.Preload("Role").Where("email = ?", req.Email).First(&user).Error; err != nil {
		helpers.RespondWithError(c, http.StatusUnauthorized, "Invalid credentials.")
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(req.Password)); err != nil {
		helpers.RespondWithError(c, http.StatusUnauthorized, "Invalid credentials.")
		return
	}

	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		helpers.RespondWithError(c, http.StatusInternalServerError, "JWT_SECRET not configured.")
		return
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    user.Role.Name,
		"exp":     time.Now().Add(24 * time.Hour).Unix(),
	})

	tokenString, err := token.SignedString([]byte(secret))
	if err != nil {
		helpers.RespondWithError(c, http.StatusInternalServerError, "Failed to generate token.")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": tokenString,
		"user": gin.H{
			"id":    user.ID,
			"email": user.Email,
			"role":  user.Role.Name,
		},
	})
}
