//go:build !noauth

// Package main implements the sequencer
package main

import (
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"golang.org/x/crypto/bcrypt"
)

// Claims defines the JWT claims
type Claims struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// User defines the user structure
type User struct {
	Username string
	Password string
	Roles    []string
}

// jwtKey is the key used to create the JWT signature
// ToDo: replace with a proper key for production
var jwtKey = []byte(getSecret(os.Getenv("SEQUENCER_JWT_KEY"), "defaultJwtKey"))

// Create a map to store users with hashed passwords
var users = map[string]User{
	"admin": {Username: "admin", Password: hashPassword(getSecret(os.Getenv("SEQUENCER_DEFAULT_ADMIN_PASSWORD_FILE"), "defaultAdminPassword")), Roles: []string{"admin"}},
	"user1": {Username: "user1", Password: hashPassword(getSecret(os.Getenv("SEQUENCER_DEFAULT_USER1_PASSWORD_FILE"), "defaultUser1Password")), Roles: []string{"user"}},
}

// hashPassword hashes a plain text password using bcrypt
func hashPassword(password string) string {
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		log.Fatalf("Failed to hash password: %v", err)
	}
	return string(hashedPassword)
}

// checkPassword compares a plain text password with a hashed password
func checkPassword(hashedPassword, password string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hashedPassword), []byte(password))
	return err == nil
}

// jwtLoginHandler is the handler for the JWT login endpoint
func jwtLoginHandler(c *gin.Context) {
	var loginVals struct {
		Username string `json:"username" binding:"required"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&loginVals); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid request"})
		return
	}

	user, exists := users[loginVals.Username]
	if !exists || !checkPassword(user.Password, loginVals.Password) {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := generateJWT(user.Username, user.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

// helper functions
// generateJWT generates a new JWT token
func generateJWT(username string, roles []string) (string, error) {
	expirationTime := time.Now().Add(7 * 24 * time.Hour)
	claims := &Claims{
		Username: username,
		Roles:    roles,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString(jwtKey)
}

// jwtAuthMiddleware is the middleware for JWT authentication
func jwtAuthMiddleware(requiredRoles []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Authorization header required"})
			c.Abort()
			return
		}

		tokenString := strings.TrimPrefix(authHeader, "Bearer ")
		claims := &Claims{}

		token, err := jwt.ParseWithClaims(tokenString, claims, func(_ *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			log.Printf("Invalid token: %v", err)
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

		log.Printf("Authenticated user: %s", claims.Username)

		hasRole := false
		for _, role := range claims.Roles {
			for _, requiredRole := range requiredRoles {
				if role == requiredRole {
					hasRole = true
					break
				}
			}
			if hasRole {
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Insufficient permissions"})
			c.Abort()
			return
		}

		c.Set("username", claims.Username)
		c.Next()
	}
}
