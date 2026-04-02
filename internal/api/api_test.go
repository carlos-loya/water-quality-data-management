package api

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"

	"github.com/carlos-loya/water-quality-data-management/internal/auth"
)

const testSecret = "test-secret-key"

// makeToken generates a valid JWT for testing.
func makeToken(t *testing.T, roles []auth.RoleClaim) string {
	t.Helper()
	tok, err := auth.GenerateToken(testSecret, uuid.New(), uuid.New(), "test@example.com", "Tester", roles)
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}
	return tok
}

// --- Health endpoint ---

func TestHealth(t *testing.T) {
	h := &handler{}
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	h.health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["status"] != "ok" {
		t.Errorf("body status = %q, want %q", body["status"], "ok")
	}
}

// --- Auth middleware ---

func TestWithAuthMissingHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	handler := withAuth(testSecret, inner)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWithAuthInvalidToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	handler := withAuth(testSecret, inner)

	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer garbage-token")
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWithAuthValidToken(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		claims := auth.UserFrom(r.Context())
		if claims == nil {
			t.Error("claims should be present in context")
			return
		}
		if claims.Email != "test@example.com" {
			t.Errorf("email = %q, want %q", claims.Email, "test@example.com")
		}
		w.WriteHeader(http.StatusOK)
	})
	handler := withAuth(testSecret, inner)

	token := makeToken(t, []auth.RoleClaim{{Role: "admin"}})
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if !called {
		t.Error("inner handler should have been called")
	}
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestWithAuthNoBearerPrefix(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	handler := withAuth(testSecret, inner)

	token := makeToken(t, nil)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", token) // missing "Bearer " prefix
	w := httptest.NewRecorder()
	handler.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// --- Role middleware ---

func TestRequireRoleAllowed(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := requireRole("admin", inner)

	claims := &auth.Claims{Roles: []auth.RoleClaim{{Role: "admin"}}}
	ctx := auth.WithUser(httptest.NewRequest("GET", "/test", nil).Context(), claims)
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("handler should be called for admin")
	}
}

func TestRequireRoleForbidden(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	handler := requireRole("admin", inner)

	claims := &auth.Claims{Roles: []auth.RoleClaim{{Role: "viewer"}}}
	ctx := auth.WithUser(httptest.NewRequest("GET", "/test", nil).Context(), claims)
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireRoleNoClaims(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	handler := requireRole("admin", inner)

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAnyRoleAllowed(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	handler := requireAnyRole([]string{"admin", "operator"}, inner)

	claims := &auth.Claims{Roles: []auth.RoleClaim{{Role: "operator"}}}
	ctx := auth.WithUser(httptest.NewRequest("GET", "/test", nil).Context(), claims)
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler(w, req)

	if !called {
		t.Error("handler should be called for operator")
	}
}

func TestRequireAnyRoleForbidden(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	handler := requireAnyRole([]string{"admin", "operator"}, inner)

	claims := &auth.Claims{Roles: []auth.RoleClaim{{Role: "viewer"}}}
	ctx := auth.WithUser(httptest.NewRequest("GET", "/test", nil).Context(), claims)
	req := httptest.NewRequest("GET", "/test", nil).WithContext(ctx)
	w := httptest.NewRecorder()
	handler(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// --- Login handler validation ---

func TestLoginMissingBody(t *testing.T) {
	h := &handler{jwtSecret: testSecret}
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader([]byte("not json")))
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginEmptyFields(t *testing.T) {
	h := &handler{jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "", "password": ""})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "email and password are required" {
		t.Errorf("error = %q, want %q", resp["error"], "email and password are required")
	}
}

func TestLoginMissingPassword(t *testing.T) {
	h := &handler{jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "test@example.com"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// --- CreateSampleResult handler validation ---

func TestCreateSampleResultBadJSON(t *testing.T) {
	h := &handler{}
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader([]byte("{")))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateSampleResultMissingLocationID(t *testing.T) {
	h := &handler{}
	body, _ := json.Marshal(map[string]any{
		"parameter_id": uuid.New().String(),
		"unit_id":      uuid.New().String(),
		"collected_at": "2026-03-15T10:00:00Z",
		"entered_by":   uuid.New().String(),
		"result_value": 7.2,
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "monitoring_location_id is required" {
		t.Errorf("error = %q, want %q", resp["error"], "monitoring_location_id is required")
	}
}

func TestCreateSampleResultMissingParameterID(t *testing.T) {
	h := &handler{}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New().String(),
		"unit_id":                uuid.New().String(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New().String(),
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "parameter_id is required" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestCreateSampleResultMissingUnitID(t *testing.T) {
	h := &handler{}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New().String(),
		"parameter_id":           uuid.New().String(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New().String(),
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "unit_id is required" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestCreateSampleResultMissingCollectedAt(t *testing.T) {
	h := &handler{}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New().String(),
		"parameter_id":           uuid.New().String(),
		"unit_id":                uuid.New().String(),
		"entered_by":             uuid.New().String(),
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "collected_at is required" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestCreateSampleResultMissingEnteredBy(t *testing.T) {
	h := &handler{}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New().String(),
		"parameter_id":           uuid.New().String(),
		"unit_id":                uuid.New().String(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "entered_by is required" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestCreateSampleResultNoValueOrQualifier(t *testing.T) {
	h := &handler{}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New().String(),
		"parameter_id":           uuid.New().String(),
		"unit_id":                uuid.New().String(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New().String(),
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var resp map[string]string
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["error"] != "either result_value or result_qualifier is required" {
		t.Errorf("error = %q", resp["error"])
	}
}

// --- writeJSON / writeError helpers ---

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "val"})

	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q, want %q", ct, "application/json")
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["key"] != "val" {
		t.Errorf("body key = %q, want %q", body["key"], "val")
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "something went wrong")

	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["error"] != "something went wrong" {
		t.Errorf("error = %q", body["error"])
	}
}
