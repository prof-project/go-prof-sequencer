//go:build !noauth

package main

import (
	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v4"
	"net/http"
	"strings"
	"time"
)

// Claims defines the JWT claims
type Claims struct {
	Username string   `json:"username"`
	Roles    []string `json:"roles"`
	jwt.RegisteredClaims
}

// Define a User struct
type User struct {
	Username string
	Password string
	Roles    []string
}

// jwtKey is the key used to create the JWT signature
// ToDo: replace with a proper key for production
var jwtKey = []byte("my_secret_key")

// tokens is a slice to store the generated tokens
var tokens []string

// Create a map to store users
var users = map[string]User{
	"admin": {Username: "admin", Password: "secret", Roles: []string{"admin", "user"}},
	"user1": {Username: "user1", Password: "password1", Roles: []string{"user"}},
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
	if !exists || user.Password != loginVals.Password {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	token, err := generateJWT(user.Username, user.Roles)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to generate token"})
		return
	}
	tokens = append(tokens, token)

	c.JSON(http.StatusOK, gin.H{
		"token": token,
	})
}

// helper functions
// generateJWT generates a new JWT token
func generateJWT(username string, roles []string) (string, error) {
	expirationTime := time.Now().Add(5 * time.Minute)
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

		token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
			return jwtKey, nil
		})

		if err != nil || !token.Valid {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid token"})
			c.Abort()
			return
		}

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
