package auth

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

const testSecret = "test-secret-key"

func TestHashAndCheckPassword(t *testing.T) {
	hash, err := HashPassword("mypassword")
	if err != nil {
		t.Fatalf("HashPassword: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}
	if hash == "mypassword" {
		t.Fatal("hash should not equal plaintext")
	}
	if !CheckPassword(hash, "mypassword") {
		t.Error("CheckPassword should return true for correct password")
	}
}

func TestCheckPasswordWrong(t *testing.T) {
	hash, _ := HashPassword("correct")
	if CheckPassword(hash, "wrong") {
		t.Error("CheckPassword should return false for wrong password")
	}
}

func TestCheckPasswordInvalidHash(t *testing.T) {
	if CheckPassword("not-a-hash", "anything") {
		t.Error("CheckPassword should return false for invalid hash")
	}
}

func TestGenerateAndValidateToken(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	roles := []RoleClaim{
		{Role: "admin"},
		{Role: "operator", FacilityID: "fac-123"},
	}

	tokenStr, err := GenerateToken(testSecret, userID, orgID, "test@example.com", "Test User", roles)
	if err != nil {
		t.Fatalf("GenerateToken: %v", err)
	}
	if tokenStr == "" {
		t.Fatal("token should not be empty")
	}

	claims, err := ValidateToken(testSecret, tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}

	if claims.UserID != userID {
		t.Errorf("UserID = %v, want %v", claims.UserID, userID)
	}
	if claims.OrganizationID != orgID {
		t.Errorf("OrganizationID = %v, want %v", claims.OrganizationID, orgID)
	}
	if claims.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", claims.Email, "test@example.com")
	}
	if claims.Name != "Test User" {
		t.Errorf("Name = %q, want %q", claims.Name, "Test User")
	}
	if len(claims.Roles) != 2 {
		t.Fatalf("Roles len = %d, want 2", len(claims.Roles))
	}
}

func TestValidateTokenWrongSecret(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	tokenStr, _ := GenerateToken(testSecret, userID, orgID, "a@b.com", "A", nil)

	_, err := ValidateToken("wrong-secret", tokenStr)
	if err == nil {
		t.Error("ValidateToken should fail with wrong secret")
	}
}

func TestValidateTokenGarbage(t *testing.T) {
	_, err := ValidateToken(testSecret, "not.a.token")
	if err == nil {
		t.Error("ValidateToken should fail for garbage input")
	}
}

func TestValidateTokenEmpty(t *testing.T) {
	_, err := ValidateToken(testSecret, "")
	if err == nil {
		t.Error("ValidateToken should fail for empty string")
	}
}

func TestHasRole(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{
			{Role: "admin"},
			{Role: "operator", FacilityID: "fac-1"},
		},
	}

	if !claims.HasRole("admin") {
		t.Error("should have admin role")
	}
	if !claims.HasRole("operator") {
		t.Error("should have operator role")
	}
	if claims.HasRole("viewer") {
		t.Error("should not have viewer role")
	}
}

func TestHasRoleEmpty(t *testing.T) {
	claims := &Claims{Roles: nil}
	if claims.HasRole("admin") {
		t.Error("nil roles should not match any role")
	}
}

func TestHasAnyRole(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{{Role: "operator"}},
	}

	if !claims.HasAnyRole("admin", "operator") {
		t.Error("should match operator from the list")
	}
	if claims.HasAnyRole("admin", "reviewer") {
		t.Error("should not match admin or reviewer")
	}
}

func TestHasFacilityAccess(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{
			{Role: "admin"}, // org-wide (empty FacilityID)
		},
	}
	if !claims.HasFacilityAccess("any-facility") {
		t.Error("org-wide role should grant access to any facility")
	}
}

func TestHasFacilityAccessScoped(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{
			{Role: "operator", FacilityID: "fac-1"},
		},
	}
	if !claims.HasFacilityAccess("fac-1") {
		t.Error("should have access to fac-1")
	}
	if claims.HasFacilityAccess("fac-2") {
		t.Error("should not have access to fac-2")
	}
}

func TestHasFacilityAccessNoRoles(t *testing.T) {
	claims := &Claims{Roles: nil}
	if claims.HasFacilityAccess("fac-1") {
		t.Error("nil roles should not grant facility access")
	}
}

func TestWithUserAndUserFrom(t *testing.T) {
	claims := &Claims{
		Email: "test@example.com",
		Roles: []RoleClaim{{Role: "admin"}},
	}

	ctx := WithUser(context.Background(), claims)
	got := UserFrom(ctx)
	if got == nil {
		t.Fatal("UserFrom should return claims")
	}
	if got.Email != "test@example.com" {
		t.Errorf("Email = %q, want %q", got.Email, "test@example.com")
	}
}

func TestUserFromEmptyContext(t *testing.T) {
	got := UserFrom(context.Background())
	if got != nil {
		t.Error("UserFrom on empty context should return nil")
	}
}
