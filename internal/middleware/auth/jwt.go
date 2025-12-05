package auth

import (
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gravadigital/telescopio-api/internal/domain/participant"
)

// JWT secret key - should be loaded from environment variable
var jwtSecret []byte

func init() {
	// Get JWT secret from environment, use default for development
	secret := os.Getenv("JWT_SECRET")
	if secret == "" {
		secret = "telescopio-dev-secret-change-in-production"
		fmt.Println("⚠️  WARNING: Using default JWT secret. Set JWT_SECRET environment variable in production!")
	}
	jwtSecret = []byte(secret)
}

// Claims represents the JWT claims
type Claims struct {
	UserID string            `json:"user_id"`
	Email  string            `json:"email"`
	Role   participant.Role  `json:"role"`
	jwt.RegisteredClaims
}

// GenerateToken generates a JWT token for a user
func GenerateToken(userID uuid.UUID, email string, role participant.Role) (string, error) {
	// Token expires in 24 hours
	expirationTime := time.Now().Add(24 * time.Hour)

	claims := &Claims{
		UserID: userID.String(),
		Email:  email,
		Role:   role,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(expirationTime),
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			NotBefore: jwt.NewNumericDate(time.Now()),
			Issuer:    "telescopio-api",
		},
	}

	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(jwtSecret)
	if err != nil {
		return "", fmt.Errorf("failed to sign token: %w", err)
	}

	return tokenString, nil
}

// ValidateToken validates a JWT token and returns the claims
func ValidateToken(tokenString string) (*Claims, error) {
	claims := &Claims{}

	token, err := jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		// Verify signing method
		if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", token.Header["alg"])
		}
		return jwtSecret, nil
	})

	if err != nil {
		return nil, fmt.Errorf("failed to parse token: %w", err)
	}

	if !token.Valid {
		return nil, errors.New("invalid token")
	}

	return claims, nil
}

// JWTAuthMiddleware is a Gin middleware that validates JWT tokens
func JWTAuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Get token from Authorization header
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "Missing Authorization header",
			})
			c.Abort()
			return
		}

		// Expected format: "Bearer <token>"
		var tokenString string
		if len(authHeader) > 7 && authHeader[:7] == "Bearer " {
			tokenString = authHeader[7:]
		} else {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": "Invalid Authorization header format. Expected: Bearer <token>",
			})
			c.Abort()
			return
		}

		// Validate token
		claims, err := ValidateToken(tokenString)
		if err != nil {
			c.JSON(401, gin.H{
				"error": "UNAUTHORIZED",
				"message": fmt.Sprintf("Invalid token: %v", err),
			})
			c.Abort()
			return
		}

		// Store claims in context for handlers to use
		c.Set("user_id", claims.UserID)
		c.Set("user_email", claims.Email)
		c.Set("user_role", claims.Role)

		c.Next()
	}
}

// GetUserIDFromContext extracts the user ID from the Gin context
func GetUserIDFromContext(c *gin.Context) (uuid.UUID, error) {
	userIDStr, exists := c.Get("user_id")
	if !exists {
		return uuid.Nil, errors.New("user_id not found in context")
	}

	userID, err := uuid.Parse(userIDStr.(string))
	if err != nil {
		return uuid.Nil, fmt.Errorf("invalid user_id format: %w", err)
	}

	return userID, nil
}

// GetUserRoleFromContext extracts the user role from the Gin context
func GetUserRoleFromContext(c *gin.Context) (participant.Role, error) {
	role, exists := c.Get("user_role")
	if !exists {
		return "", errors.New("user_role not found in context")
	}

	return role.(participant.Role), nil
}

// GetUserEmailFromContext extracts the user email from the Gin context
func GetUserEmailFromContext(c *gin.Context) (string, error) {
	email, exists := c.Get("user_email")
	if !exists {
		return "", errors.New("user_email not found in context")
	}

	return email.(string), nil
}
