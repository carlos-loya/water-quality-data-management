package auth

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
)

const testSecret = "test-secret-key"

// =========================================================================
// Password hashing
// =========================================================================

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

func TestHashPasswordEmptyString(t *testing.T) {
	hash, err := HashPassword("")
	if err != nil {
		t.Fatalf("HashPassword with empty string: %v", err)
	}
	if !CheckPassword(hash, "") {
		t.Error("CheckPassword should match empty password")
	}
	if CheckPassword(hash, "notempty") {
		t.Error("CheckPassword should reject non-empty password against empty hash")
	}
}

func TestHashPasswordLong(t *testing.T) {
	// bcrypt silently truncates at 72 bytes — both should hash the same
	long72 := strings.Repeat("a", 72)
	long100 := strings.Repeat("a", 100)

	hash, err := HashPassword(long72)
	if err != nil {
		t.Fatalf("HashPassword 72-byte: %v", err)
	}
	if !CheckPassword(hash, long72) {
		t.Error("should match 72-byte password")
	}
	// bcrypt truncates, so 100 a's and 72 a's produce the same hash
	if !CheckPassword(hash, long100) {
		t.Error("bcrypt truncates at 72 bytes, so 100 a's should also match")
	}
}

// =========================================================================
// Token generation and validation
// =========================================================================

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

func TestGenerateTokenNilRoles(t *testing.T) {
	tokenStr, err := GenerateToken(testSecret, uuid.New(), uuid.New(), "a@b.com", "A", nil)
	if err != nil {
		t.Fatalf("GenerateToken with nil roles: %v", err)
	}
	claims, err := ValidateToken(testSecret, tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if claims.Roles != nil && len(claims.Roles) != 0 {
		t.Errorf("expected nil or empty roles, got %v", claims.Roles)
	}
}

func TestGenerateTokenEmptyRoles(t *testing.T) {
	tokenStr, err := GenerateToken(testSecret, uuid.New(), uuid.New(), "a@b.com", "A", []RoleClaim{})
	if err != nil {
		t.Fatalf("GenerateToken with empty roles: %v", err)
	}
	claims, err := ValidateToken(testSecret, tokenStr)
	if err != nil {
		t.Fatalf("ValidateToken: %v", err)
	}
	if len(claims.Roles) != 0 {
		t.Errorf("expected empty roles, got %d", len(claims.Roles))
	}
}

func TestValidateTokenExpired(t *testing.T) {
	claims := Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "expired@test.com",
		Name:           "Expired",
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(time.Now().Add(-2 * time.Hour)),
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-1 * time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenStr, err := token.SignedString([]byte(testSecret))
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	_, err = ValidateToken(testSecret, tokenStr)
	if err == nil {
		t.Error("ValidateToken should reject expired token")
	}
}

func TestValidateTokenWrongAlgorithm(t *testing.T) {
	// Create a token that claims to use "none" algorithm
	claims := Claims{
		UserID: uuid.New(),
		Email:  "alg@test.com",
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodNone, claims)
	tokenStr, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
	if err != nil {
		t.Fatalf("sign token: %v", err)
	}
	_, err = ValidateToken(testSecret, tokenStr)
	if err == nil {
		t.Error("ValidateToken should reject non-HMAC algorithm")
	}
}

func TestGenerateTokenConcurrent(t *testing.T) {
	var wg sync.WaitGroup
	errs := make(chan error, 50)

	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			tok, err := GenerateToken(testSecret, uuid.New(), uuid.New(), "c@d.com", "C", []RoleClaim{{Role: "admin"}})
			if err != nil {
				errs <- err
				return
			}
			if _, err := ValidateToken(testSecret, tok); err != nil {
				errs <- err
			}
		}()
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		t.Errorf("concurrent token error: %v", err)
	}
}

// =========================================================================
// Role checks
// =========================================================================

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

	claims2 := &Claims{Roles: []RoleClaim{}}
	if claims2.HasRole("admin") {
		t.Error("empty roles should not match any role")
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

func TestHasAnyRoleEmptyArgs(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{{Role: "admin"}},
	}
	if claims.HasAnyRole() {
		t.Error("HasAnyRole with no args should return false")
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

func TestHasFacilityAccessMixedScopedAndOrgWide(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{
			{Role: "operator", FacilityID: "fac-1"},
			{Role: "admin"}, // org-wide
		},
	}
	if !claims.HasFacilityAccess("fac-1") {
		t.Error("should have access to fac-1 via scoped role")
	}
	if !claims.HasFacilityAccess("fac-2") {
		t.Error("should have access to fac-2 via org-wide admin role")
	}
}

// =========================================================================
// Context helpers
// =========================================================================

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

func TestRequireRoleWithMultipleRolesOnlyOneMatches(t *testing.T) {
	claims := &Claims{
		Roles: []RoleClaim{
			{Role: "viewer"},
			{Role: "operator"},
			{Role: "reviewer"},
		},
	}
	if !claims.HasRole("reviewer") {
		t.Error("should find reviewer among multiple roles")
	}
	if claims.HasRole("admin") {
		t.Error("should not find admin among multiple roles")
	}
}
