package auth

import (
	"context"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"golang.org/x/crypto/bcrypt"
)

type contextKey string

const userContextKey contextKey = "user"

// RoleClaim represents a single role assignment, optionally scoped to a facility.
type RoleClaim struct {
	Role       string `json:"role"`
	FacilityID string `json:"facility_id,omitempty"`
}

// Claims represents the JWT payload.
type Claims struct {
	UserID         uuid.UUID   `json:"user_id"`
	OrganizationID uuid.UUID  `json:"org_id"`
	Email          string     `json:"email"`
	Name           string     `json:"name"`
	Roles          []RoleClaim `json:"roles"`
	jwt.RegisteredClaims
}

// HashPassword hashes a plaintext password using bcrypt.
func HashPassword(password string) (string, error) {
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("hash password: %w", err)
	}
	return string(hash), nil
}

// CheckPassword compares a plaintext password against a bcrypt hash.
func CheckPassword(hash, password string) bool {
	return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)) == nil
}

// GenerateToken creates a signed JWT for the given user.
func GenerateToken(secret string, userID, orgID uuid.UUID, email, name string, roles []RoleClaim) (string, error) {
	claims := Claims{
		UserID:         userID,
		OrganizationID: orgID,
		Email:          email,
		Name:           name,
		Roles:          roles,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now()),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(24 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	return token.SignedString([]byte(secret))
}

// HasRole returns true if the user holds the named role (any scope).
func (c *Claims) HasRole(role string) bool {
	for _, r := range c.Roles {
		if r.Role == role {
			return true
		}
	}
	return false
}

// HasAnyRole returns true if the user holds any of the named roles.
func (c *Claims) HasAnyRole(roles ...string) bool {
	for _, role := range roles {
		if c.HasRole(role) {
			return true
		}
	}
	return false
}

// HasFacilityAccess returns true if the user has any role covering the given facility
// (org-wide roles with empty FacilityID cover all facilities).
func (c *Claims) HasFacilityAccess(facilityID string) bool {
	for _, r := range c.Roles {
		if r.FacilityID == "" || r.FacilityID == facilityID {
			return true
		}
	}
	return false
}

// ValidateToken parses and validates a JWT string, returning the claims.
func ValidateToken(secret, tokenStr string) (*Claims, error) {
	token, err := jwt.ParseWithClaims(tokenStr, &Claims{}, func(t *jwt.Token) (any, error) {
		if _, ok := t.Method.(*jwt.SigningMethodHMAC); !ok {
			return nil, fmt.Errorf("unexpected signing method: %v", t.Header["alg"])
		}
		return []byte(secret), nil
	})
	if err != nil {
		return nil, fmt.Errorf("parse token: %w", err)
	}
	claims, ok := token.Claims.(*Claims)
	if !ok || !token.Valid {
		return nil, fmt.Errorf("invalid token")
	}
	return claims, nil
}

// WithUser stores claims in the request context.
func WithUser(ctx context.Context, claims *Claims) context.Context {
	return context.WithValue(ctx, userContextKey, claims)
}

// UserFrom extracts claims from the request context.
func UserFrom(ctx context.Context) *Claims {
	claims, _ := ctx.Value(userContextKey).(*Claims)
	return claims
}
