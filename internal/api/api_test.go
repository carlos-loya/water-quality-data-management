package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/carlos-loya/water-quality-data-management/internal/auth"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

const testSecret = "test-secret-key"

// =========================================================================
// Mock Store
// =========================================================================

type mockStore struct {
	// Configurable return values per method
	getUserByEmailFn           func(ctx context.Context, email string) (storage.User, error)
	getUserRolesFn             func(ctx context.Context, userID uuid.UUID) ([]storage.UserRole, error)
	listFacilitiesForUserFn    func(ctx context.Context, orgID, userID uuid.UUID) ([]storage.Facility, error)
	listMonitoringLocationsFn  func(ctx context.Context, facilityID uuid.UUID) ([]storage.MonitoringLocation, error)
	listAllMonitoringLocsFn    func(ctx context.Context, orgID uuid.UUID) ([]storage.MonitoringLocation, error)
	listParametersFn           func(ctx context.Context, orgID uuid.UUID) ([]storage.Parameter, error)
	listUnitsFn                func(ctx context.Context, orgID uuid.UUID) ([]storage.UnitOfMeasure, error)
	listValidationRulesFn      func(ctx context.Context, orgID uuid.UUID) ([]storage.ValidationRule, error)
	getValidationRuleFn        func(ctx context.Context, parameterID uuid.UUID) (storage.ValidationRule, error)
	listSampleResultsFn        func(ctx context.Context, f storage.SampleResultFilter) ([]storage.SampleResult, error)
	createSampleResultFn       func(ctx context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error)
	getSampleResultFn          func(ctx context.Context, id uuid.UUID) (storage.SampleResult, error)
	reviewSampleResultFn       func(ctx context.Context, id, reviewerID uuid.UUID) (storage.SampleResult, error)
	approveSampleResultFn      func(ctx context.Context, id, approverID uuid.UUID) (storage.SampleResult, error)
	evaluateComplianceFn       func(ctx context.Context, facilityID uuid.UUID) ([]storage.ComplianceResult, error)
	getTrendingDataFn          func(ctx context.Context, facilityID uuid.UUID, days int) ([]storage.TrendingSeries, error)
	listInstrumentStatusesFn   func(ctx context.Context, facilityID uuid.UUID) ([]storage.InstrumentStatus, error)
	listCalibrationRecordsFn   func(ctx context.Context, instrumentID uuid.UUID) ([]storage.CalibrationRecord, error)
	getOrganizationIDForResFn  func(ctx context.Context, resultID uuid.UUID) (uuid.UUID, error)
	listAuditLogFn             func(ctx context.Context, recordID uuid.UUID) ([]storage.AuditEntry, error)
	listAllFacilitiesFn        func(ctx context.Context) ([]storage.Facility, error)
	listFacilityExceedancesFn  func(ctx context.Context, facilityID uuid.UUID) ([]storage.Exceedance, error)
	createAlertFn              func(ctx context.Context, p storage.CreateAlertParams) (storage.Alert, bool, error)
	listAlertsFn               func(ctx context.Context, f storage.AlertFilter) ([]storage.Alert, error)
	getAlertFn                 func(ctx context.Context, id uuid.UUID) (storage.Alert, error)
	dismissAlertFn             func(ctx context.Context, id, userID uuid.UUID) (storage.Alert, error)
}

func (m *mockStore) GetUserByEmail(ctx context.Context, email string) (storage.User, error) {
	if m.getUserByEmailFn != nil {
		return m.getUserByEmailFn(ctx, email)
	}
	return storage.User{}, pgx.ErrNoRows
}

func (m *mockStore) GetUserRoles(ctx context.Context, userID uuid.UUID) ([]storage.UserRole, error) {
	if m.getUserRolesFn != nil {
		return m.getUserRolesFn(ctx, userID)
	}
	return nil, nil
}

func (m *mockStore) GetFacilityIDForLocation(context.Context, uuid.UUID) (uuid.UUID, error) {
	return uuid.Nil, nil
}

func (m *mockStore) ListFacilitiesForUser(ctx context.Context, orgID, userID uuid.UUID) ([]storage.Facility, error) {
	if m.listFacilitiesForUserFn != nil {
		return m.listFacilitiesForUserFn(ctx, orgID, userID)
	}
	return nil, nil
}

func (m *mockStore) ListFacilities(context.Context, uuid.UUID) ([]storage.Facility, error) {
	return nil, nil
}

func (m *mockStore) ListMonitoringLocations(ctx context.Context, facilityID uuid.UUID) ([]storage.MonitoringLocation, error) {
	if m.listMonitoringLocationsFn != nil {
		return m.listMonitoringLocationsFn(ctx, facilityID)
	}
	return nil, nil
}

func (m *mockStore) ListAllMonitoringLocations(ctx context.Context, orgID uuid.UUID) ([]storage.MonitoringLocation, error) {
	if m.listAllMonitoringLocsFn != nil {
		return m.listAllMonitoringLocsFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockStore) ListParameters(ctx context.Context, orgID uuid.UUID) ([]storage.Parameter, error) {
	if m.listParametersFn != nil {
		return m.listParametersFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockStore) ListUnits(ctx context.Context, orgID uuid.UUID) ([]storage.UnitOfMeasure, error) {
	if m.listUnitsFn != nil {
		return m.listUnitsFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockStore) ListValidationRules(ctx context.Context, orgID uuid.UUID) ([]storage.ValidationRule, error) {
	if m.listValidationRulesFn != nil {
		return m.listValidationRulesFn(ctx, orgID)
	}
	return nil, nil
}

func (m *mockStore) GetValidationRule(ctx context.Context, parameterID uuid.UUID) (storage.ValidationRule, error) {
	if m.getValidationRuleFn != nil {
		return m.getValidationRuleFn(ctx, parameterID)
	}
	return storage.ValidationRule{}, pgx.ErrNoRows
}

func (m *mockStore) ListSampleResults(ctx context.Context, f storage.SampleResultFilter) ([]storage.SampleResult, error) {
	if m.listSampleResultsFn != nil {
		return m.listSampleResultsFn(ctx, f)
	}
	return nil, nil
}

func (m *mockStore) CreateSampleResult(ctx context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error) {
	if m.createSampleResultFn != nil {
		return m.createSampleResultFn(ctx, p)
	}
	return storage.SampleResult{ID: uuid.New(), Status: "draft"}, nil
}

func (m *mockStore) GetSampleResult(ctx context.Context, id uuid.UUID) (storage.SampleResult, error) {
	if m.getSampleResultFn != nil {
		return m.getSampleResultFn(ctx, id)
	}
	return storage.SampleResult{}, pgx.ErrNoRows
}

func (m *mockStore) ReviewSampleResult(ctx context.Context, id, reviewerID uuid.UUID) (storage.SampleResult, error) {
	if m.reviewSampleResultFn != nil {
		return m.reviewSampleResultFn(ctx, id, reviewerID)
	}
	return storage.SampleResult{}, pgx.ErrNoRows
}

func (m *mockStore) ApproveSampleResult(ctx context.Context, id, approverID uuid.UUID) (storage.SampleResult, error) {
	if m.approveSampleResultFn != nil {
		return m.approveSampleResultFn(ctx, id, approverID)
	}
	return storage.SampleResult{}, pgx.ErrNoRows
}

func (m *mockStore) EvaluateCompliance(ctx context.Context, facilityID uuid.UUID) ([]storage.ComplianceResult, error) {
	if m.evaluateComplianceFn != nil {
		return m.evaluateComplianceFn(ctx, facilityID)
	}
	return nil, nil
}

func (m *mockStore) GetTrendingData(ctx context.Context, facilityID uuid.UUID, days int) ([]storage.TrendingSeries, error) {
	if m.getTrendingDataFn != nil {
		return m.getTrendingDataFn(ctx, facilityID, days)
	}
	return nil, nil
}

func (m *mockStore) ListInstruments(context.Context, uuid.UUID) ([]storage.Instrument, error) {
	return nil, nil
}

func (m *mockStore) ListCalibrationRecords(ctx context.Context, instrumentID uuid.UUID) ([]storage.CalibrationRecord, error) {
	if m.listCalibrationRecordsFn != nil {
		return m.listCalibrationRecordsFn(ctx, instrumentID)
	}
	return nil, nil
}

func (m *mockStore) ListInstrumentStatuses(ctx context.Context, facilityID uuid.UUID) ([]storage.InstrumentStatus, error) {
	if m.listInstrumentStatusesFn != nil {
		return m.listInstrumentStatusesFn(ctx, facilityID)
	}
	return nil, nil
}

func (m *mockStore) GetOrganizationIDForResult(ctx context.Context, resultID uuid.UUID) (uuid.UUID, error) {
	if m.getOrganizationIDForResFn != nil {
		return m.getOrganizationIDForResFn(ctx, resultID)
	}
	return uuid.New(), nil
}

func (m *mockStore) ListAuditLog(ctx context.Context, recordID uuid.UUID) ([]storage.AuditEntry, error) {
	if m.listAuditLogFn != nil {
		return m.listAuditLogFn(ctx, recordID)
	}
	return nil, nil
}

func (m *mockStore) ListAllFacilities(ctx context.Context) ([]storage.Facility, error) {
	if m.listAllFacilitiesFn != nil {
		return m.listAllFacilitiesFn(ctx)
	}
	return nil, nil
}

func (m *mockStore) ListFacilityExceedances(ctx context.Context, facilityID uuid.UUID) ([]storage.Exceedance, error) {
	if m.listFacilityExceedancesFn != nil {
		return m.listFacilityExceedancesFn(ctx, facilityID)
	}
	return nil, nil
}

func (m *mockStore) CreateAlert(ctx context.Context, p storage.CreateAlertParams) (storage.Alert, bool, error) {
	if m.createAlertFn != nil {
		return m.createAlertFn(ctx, p)
	}
	return storage.Alert{}, false, nil
}

func (m *mockStore) ListAlerts(ctx context.Context, f storage.AlertFilter) ([]storage.Alert, error) {
	if m.listAlertsFn != nil {
		return m.listAlertsFn(ctx, f)
	}
	return nil, nil
}

func (m *mockStore) GetAlert(ctx context.Context, id uuid.UUID) (storage.Alert, error) {
	if m.getAlertFn != nil {
		return m.getAlertFn(ctx, id)
	}
	return storage.Alert{}, pgx.ErrNoRows
}

func (m *mockStore) DismissAlert(ctx context.Context, id, userID uuid.UUID) (storage.Alert, error) {
	if m.dismissAlertFn != nil {
		return m.dismissAlertFn(ctx, id, userID)
	}
	return storage.Alert{}, pgx.ErrNoRows
}

// =========================================================================
// Test helpers
// =========================================================================

func makeToken(t *testing.T, roles []auth.RoleClaim) string {
	t.Helper()
	tok, err := auth.GenerateToken(testSecret, uuid.New(), uuid.New(), "test@example.com", "Tester", roles)
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}
	return tok
}

func makeTokenWithIDs(t *testing.T, userID, orgID uuid.UUID, roles []auth.RoleClaim) string {
	t.Helper()
	tok, err := auth.GenerateToken(testSecret, userID, orgID, "test@example.com", "Tester", roles)
	if err != nil {
		t.Fatalf("generate test token: %v", err)
	}
	return tok
}

func withAuthCtx(r *http.Request, roles []auth.RoleClaim) *http.Request {
	claims := &auth.Claims{
		UserID:         uuid.New(),
		OrganizationID: uuid.New(),
		Email:          "test@example.com",
		Name:           "Tester",
		Roles:          roles,
	}
	return r.WithContext(auth.WithUser(r.Context(), claims))
}

func decodeBody(t *testing.T, body io.Reader) map[string]any {
	t.Helper()
	var m map[string]any
	if err := json.NewDecoder(body).Decode(&m); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	return m
}

func decodeArray(t *testing.T, body io.Reader) []any {
	t.Helper()
	var a []any
	if err := json.NewDecoder(body).Decode(&a); err != nil {
		t.Fatalf("decode array body: %v", err)
	}
	return a
}

// =========================================================================
// 1. Health endpoint
// =========================================================================

func TestHealth(t *testing.T) {
	h := &handler{}
	req := httptest.NewRequest("GET", "/api/v1/health", nil)
	w := httptest.NewRecorder()
	h.health(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	body := decodeBody(t, w.Body)
	if body["status"] != "ok" {
		t.Errorf("body status = %q, want %q", body["status"], "ok")
	}
}

// =========================================================================
// 2. Auth middleware
// =========================================================================

func TestWithAuthMissingHeader(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	h := withAuth(testSecret, inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWithAuthInvalidToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	h := withAuth(testSecret, inner)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer garbage-token")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
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
			t.Error("claims should be present")
		}
		w.WriteHeader(http.StatusOK)
	})
	h := withAuth(testSecret, inner)
	token := makeToken(t, []auth.RoleClaim{{Role: "admin"}})
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if !called {
		t.Error("inner handler should have been called")
	}
}

func TestWithAuthNoBearerPrefix(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	h := withAuth(testSecret, inner)
	token := makeToken(t, nil)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", token) // missing "Bearer "
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestWithAuthBearerPrefixEmptyToken(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("inner handler should not be called")
	})
	h := withAuth(testSecret, inner)
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("Authorization", "Bearer ")
	w := httptest.NewRecorder()
	h.ServeHTTP(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// =========================================================================
// 3. Role middleware
// =========================================================================

func TestRequireRoleAllowed(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := requireRole("admin", inner)
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h(w, req)
	if !called {
		t.Error("handler should be called for admin")
	}
}

func TestRequireRoleForbidden(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	h := requireRole("admin", inner)
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "viewer"}})
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireRoleNoClaims(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	h := requireRole("admin", inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireRoleMultipleRolesOneMatches(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := requireRole("reviewer", inner)
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "operator"}, {Role: "reviewer"}})
	w := httptest.NewRecorder()
	h(w, req)
	if !called {
		t.Error("handler should be called when one role matches")
	}
}

func TestRequireAnyRoleAllowed(t *testing.T) {
	var called bool
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		w.WriteHeader(http.StatusOK)
	})
	h := requireAnyRole([]string{"admin", "operator"}, inner)
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "operator"}})
	w := httptest.NewRecorder()
	h(w, req)
	if !called {
		t.Error("handler should be called for operator")
	}
}

func TestRequireAnyRoleForbidden(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	h := requireAnyRole([]string{"admin", "operator"}, inner)
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "viewer"}})
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestRequireAnyRoleNoClaims(t *testing.T) {
	inner := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("handler should not be called")
	})
	h := requireAnyRole([]string{"admin"}, inner)
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	h(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// =========================================================================
// 4. statusWriter
// =========================================================================

func TestStatusWriterCapturesCode(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, status: http.StatusOK}
	sw.WriteHeader(http.StatusNotFound)
	if sw.status != http.StatusNotFound {
		t.Errorf("status = %d, want %d", sw.status, http.StatusNotFound)
	}
}

// =========================================================================
// 5. Login handler
// =========================================================================

func TestLoginInvalidJSON(t *testing.T) {
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

func TestLoginMissingEmail(t *testing.T) {
	h := &handler{jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"password": "secret"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestLoginUserNotFound(t *testing.T) {
	m := &mockStore{} // default returns ErrNoRows
	h := &handler{store: m, jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "nobody@test.com", "password": "secret"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	resp := decodeBody(t, w.Body)
	if resp["error"] != "invalid credentials" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestLoginAccountDisabled(t *testing.T) {
	hash, _ := auth.HashPassword("password")
	m := &mockStore{
		getUserByEmailFn: func(_ context.Context, _ string) (storage.User, error) {
			return storage.User{ID: uuid.New(), Active: false, PasswordHash: hash}, nil
		},
	}
	h := &handler{store: m, jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "disabled@test.com", "password": "password"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
	resp := decodeBody(t, w.Body)
	if resp["error"] != "account is disabled" {
		t.Errorf("error = %q", resp["error"])
	}
}

func TestLoginWrongPassword(t *testing.T) {
	hash, _ := auth.HashPassword("correct")
	m := &mockStore{
		getUserByEmailFn: func(_ context.Context, _ string) (storage.User, error) {
			return storage.User{ID: uuid.New(), Active: true, PasswordHash: hash}, nil
		},
	}
	h := &handler{store: m, jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "user@test.com", "password": "wrong"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestLoginSuccess(t *testing.T) {
	userID := uuid.New()
	orgID := uuid.New()
	hash, _ := auth.HashPassword("password")
	m := &mockStore{
		getUserByEmailFn: func(_ context.Context, _ string) (storage.User, error) {
			return storage.User{ID: userID, OrganizationID: orgID, Email: "admin@test.com", Name: "Admin", Active: true, PasswordHash: hash}, nil
		},
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]storage.UserRole, error) {
			return []storage.UserRole{{RoleName: "admin"}}, nil
		},
	}
	h := &handler{store: m, jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "admin@test.com", "password": "password"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeBody(t, w.Body)
	if resp["token"] == nil || resp["token"] == "" {
		t.Error("expected non-empty token")
	}
	user := resp["user"].(map[string]any)
	if user["email"] != "admin@test.com" {
		t.Errorf("user email = %q", user["email"])
	}
}

func TestLoginGetUserRolesError(t *testing.T) {
	hash, _ := auth.HashPassword("password")
	m := &mockStore{
		getUserByEmailFn: func(_ context.Context, _ string) (storage.User, error) {
			return storage.User{ID: uuid.New(), Active: true, PasswordHash: hash}, nil
		},
		getUserRolesFn: func(_ context.Context, _ uuid.UUID) ([]storage.UserRole, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	h := &handler{store: m, jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]string{"email": "a@b.com", "password": "password"})
	req := httptest.NewRequest("POST", "/api/v1/auth/login", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.login(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// =========================================================================
// 6. Me handler
// =========================================================================

func TestMeWithAuth(t *testing.T) {
	h := &handler{}
	req := withAuthCtx(httptest.NewRequest("GET", "/api/v1/auth/me", nil), []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.me(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeBody(t, w.Body)
	if resp["email"] != "test@example.com" {
		t.Errorf("email = %q", resp["email"])
	}
}

func TestMeWithoutAuth(t *testing.T) {
	h := &handler{}
	req := httptest.NewRequest("GET", "/api/v1/auth/me", nil)
	w := httptest.NewRecorder()
	h.me(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

// =========================================================================
// 7. ListFacilities handler
// =========================================================================

func TestListFacilitiesValid(t *testing.T) {
	facID := uuid.New()
	m := &mockStore{
		listFacilitiesForUserFn: func(_ context.Context, _, _ uuid.UUID) ([]storage.Facility, error) {
			return []storage.Facility{{ID: facID, Name: "Plant A"}}, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/api/v1/organizations/"+uuid.New().String()+"/facilities", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listFacilities(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListFacilitiesInvalidOrgID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("GET", "/api/v1/organizations/bad/facilities", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("org_id", "bad")
	w := httptest.NewRecorder()
	h.listFacilities(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListFacilitiesEmpty(t *testing.T) {
	m := &mockStore{
		listFacilitiesForUserFn: func(_ context.Context, _, _ uuid.UUID) ([]storage.Facility, error) {
			return []storage.Facility{}, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listFacilities(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	arr := decodeArray(t, w.Body)
	if len(arr) != 0 {
		t.Errorf("expected empty array, got %d", len(arr))
	}
}

func TestListFacilitiesDBError(t *testing.T) {
	m := &mockStore{
		listFacilitiesForUserFn: func(_ context.Context, _, _ uuid.UUID) ([]storage.Facility, error) {
			return nil, fmt.Errorf("db error")
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listFacilities(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

// =========================================================================
// 8. ListParameters / ListUnits
// =========================================================================

func TestListParametersValid(t *testing.T) {
	m := &mockStore{
		listParametersFn: func(_ context.Context, _ uuid.UUID) ([]storage.Parameter, error) {
			return []storage.Parameter{{Code: "PH"}}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listParameters(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListParametersInvalidOrgID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("org_id", "bad")
	w := httptest.NewRecorder()
	h.listParameters(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListUnitsValid(t *testing.T) {
	m := &mockStore{
		listUnitsFn: func(_ context.Context, _ uuid.UUID) ([]storage.UnitOfMeasure, error) {
			return []storage.UnitOfMeasure{{Code: "mg/L"}}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listUnits(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListUnitsInvalidOrgID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("org_id", "bad")
	w := httptest.NewRecorder()
	h.listUnits(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// 9. CreateSampleResult handler
// =========================================================================

func TestCreateSampleResultBadJSON(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader([]byte("{")))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestCreateSampleResultMissingFields(t *testing.T) {
	tests := []struct {
		name    string
		body    map[string]any
		errText string
	}{
		{"missing location", map[string]any{"parameter_id": uuid.New(), "unit_id": uuid.New(), "collected_at": "2026-03-15T10:00:00Z", "entered_by": uuid.New(), "result_value": 7.2}, "monitoring_location_id is required"},
		{"missing parameter", map[string]any{"monitoring_location_id": uuid.New(), "unit_id": uuid.New(), "collected_at": "2026-03-15T10:00:00Z", "entered_by": uuid.New(), "result_value": 7.2}, "parameter_id is required"},
		{"missing unit", map[string]any{"monitoring_location_id": uuid.New(), "parameter_id": uuid.New(), "collected_at": "2026-03-15T10:00:00Z", "entered_by": uuid.New(), "result_value": 7.2}, "unit_id is required"},
		{"missing collected_at", map[string]any{"monitoring_location_id": uuid.New(), "parameter_id": uuid.New(), "unit_id": uuid.New(), "entered_by": uuid.New(), "result_value": 7.2}, "collected_at is required"},
		{"missing entered_by", map[string]any{"monitoring_location_id": uuid.New(), "parameter_id": uuid.New(), "unit_id": uuid.New(), "collected_at": "2026-03-15T10:00:00Z", "result_value": 7.2}, "entered_by is required"},
		{"no value or qualifier", map[string]any{"monitoring_location_id": uuid.New(), "parameter_id": uuid.New(), "unit_id": uuid.New(), "collected_at": "2026-03-15T10:00:00Z", "entered_by": uuid.New()}, "either result_value or result_qualifier is required"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			h := &handler{store: &mockStore{}}
			body, _ := json.Marshal(tc.body)
			req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
			w := httptest.NewRecorder()
			h.createSampleResult(w, req)
			if w.Code != http.StatusBadRequest {
				t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
			}
			resp := decodeBody(t, w.Body)
			if resp["error"] != tc.errText {
				t.Errorf("error = %q, want %q", resp["error"], tc.errText)
			}
		})
	}
}

func TestCreateSampleResultSuccess(t *testing.T) {
	resultID := uuid.New()
	m := &mockStore{
		createSampleResultFn: func(_ context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error) {
			return storage.SampleResult{
				ID:      resultID,
				Status:  "draft",
				Source:  p.Source,
				UnitID:  p.UnitID,
			}, nil
		},
		getOrganizationIDForResFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	h := &handler{store: m, jwtSecret: testSecret}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New(),
		"parameter_id":           uuid.New(),
		"unit_id":                uuid.New(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New(),
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/api/v1/sample-results", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	resp := decodeBody(t, w.Body)
	if resp["status"] != "draft" {
		t.Errorf("status = %q, want draft", resp["status"])
	}
}

func TestCreateSampleResultDefaultSource(t *testing.T) {
	var capturedSource string
	m := &mockStore{
		createSampleResultFn: func(_ context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error) {
			capturedSource = p.Source
			return storage.SampleResult{ID: uuid.New(), Status: "draft", Source: p.Source}, nil
		},
		getOrganizationIDForResFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	h := &handler{store: m}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New(),
		"parameter_id":           uuid.New(),
		"unit_id":                uuid.New(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New(),
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if capturedSource != "manual" {
		t.Errorf("source = %q, want 'manual'", capturedSource)
	}
}

func TestCreateSampleResultDBError(t *testing.T) {
	m := &mockStore{
		createSampleResultFn: func(_ context.Context, _ storage.CreateSampleResultParams) (storage.SampleResult, error) {
			return storage.SampleResult{}, fmt.Errorf("db error")
		},
	}
	h := &handler{store: m}
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New(),
		"parameter_id":           uuid.New(),
		"unit_id":                uuid.New(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New(),
		"result_value":           7.2,
	})
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusInternalServerError {
		t.Errorf("status = %d, want %d", w.Code, http.StatusInternalServerError)
	}
}

func TestCreateSampleResultValidationRuleMinMax(t *testing.T) {
	minVal := 0.0
	maxVal := 14.0
	m := &mockStore{
		getValidationRuleFn: func(_ context.Context, _ uuid.UUID) (storage.ValidationRule, error) {
			return storage.ValidationRule{MinValue: &minVal, MaxValue: &maxVal, IsRequired: true}, nil
		},
	}
	h := &handler{store: m}

	// Below minimum
	body, _ := json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New(),
		"parameter_id":           uuid.New(),
		"unit_id":                uuid.New(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New(),
		"result_value":           -1.0,
	})
	req := httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("below min: status = %d, want %d", w.Code, http.StatusBadRequest)
	}

	// Above maximum
	body, _ = json.Marshal(map[string]any{
		"monitoring_location_id": uuid.New(),
		"parameter_id":           uuid.New(),
		"unit_id":                uuid.New(),
		"collected_at":           "2026-03-15T10:00:00Z",
		"entered_by":             uuid.New(),
		"result_value":           15.0,
	})
	req = httptest.NewRequest("POST", "/test", bytes.NewReader(body))
	w = httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("above max: status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// 10. ReviewSampleResult handler
// =========================================================================

func TestReviewSampleResultInvalidID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("PATCH", "/test", nil), []auth.RoleClaim{{Role: "reviewer"}})
	req.SetPathValue("id", "bad-uuid")
	w := httptest.NewRecorder()
	h.reviewSampleResult(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestReviewSampleResultNoAuth(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("PATCH", "/test", nil)
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.reviewSampleResult(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want %d", w.Code, http.StatusUnauthorized)
	}
}

func TestReviewSampleResultNotFound(t *testing.T) {
	h := &handler{store: &mockStore{}} // getSampleResult returns ErrNoRows by default
	req := withAuthCtx(httptest.NewRequest("PATCH", "/test", nil), []auth.RoleClaim{{Role: "reviewer"}})
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.reviewSampleResult(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestReviewSampleResultAlreadyReviewed(t *testing.T) {
	m := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{Status: "reviewed"}, nil
		},
		// ReviewSampleResult returns ErrNoRows when status != draft
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("PATCH", "/test", nil), []auth.RoleClaim{{Role: "reviewer"}})
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.reviewSampleResult(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestReviewSampleResultSuccess(t *testing.T) {
	reviewerID := uuid.New()
	now := time.Now()
	m := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{Status: "draft", EnteredBy: uuid.New()}, nil
		},
		reviewSampleResultFn: func(_ context.Context, _, rID uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{Status: "reviewed", ReviewedBy: &rID, ReviewedAt: &now}, nil
		},
		getOrganizationIDForResFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	h := &handler{store: m}

	claims := &auth.Claims{UserID: reviewerID, Roles: []auth.RoleClaim{{Role: "reviewer"}}}
	req := httptest.NewRequest("PATCH", "/test", nil)
	req = req.WithContext(auth.WithUser(req.Context(), claims))
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.reviewSampleResult(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeBody(t, w.Body)
	if resp["status"] != "reviewed" {
		t.Errorf("status = %q, want reviewed", resp["status"])
	}
}

// =========================================================================
// 11. ApproveSampleResult handler
// =========================================================================

func TestApproveSampleResultInvalidID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("PATCH", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("id", "bad-uuid")
	w := httptest.NewRecorder()
	h.approveSampleResult(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestApproveSampleResultNotFound(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("PATCH", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.approveSampleResult(w, req)
	if w.Code != http.StatusNotFound {
		t.Errorf("status = %d, want %d", w.Code, http.StatusNotFound)
	}
}

func TestApproveSampleResultNotInReviewedStatus(t *testing.T) {
	m := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{Status: "draft"}, nil
		},
		// ApproveSampleResult returns ErrNoRows when status != reviewed
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("PATCH", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.approveSampleResult(w, req)
	if w.Code != http.StatusConflict {
		t.Errorf("status = %d, want %d", w.Code, http.StatusConflict)
	}
}

func TestApproveSampleResultSuccess(t *testing.T) {
	approverID := uuid.New()
	now := time.Now()
	m := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{Status: "reviewed", EnteredBy: uuid.New()}, nil
		},
		approveSampleResultFn: func(_ context.Context, _, aID uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{Status: "approved", ApprovedBy: &aID, ApprovedAt: &now}, nil
		},
		getOrganizationIDForResFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	h := &handler{store: m}
	claims := &auth.Claims{UserID: approverID, Roles: []auth.RoleClaim{{Role: "admin"}}}
	req := httptest.NewRequest("PATCH", "/test", nil)
	req = req.WithContext(auth.WithUser(req.Context(), claims))
	req.SetPathValue("id", uuid.New().String())
	w := httptest.NewRecorder()
	h.approveSampleResult(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeBody(t, w.Body)
	if resp["status"] != "approved" {
		t.Errorf("status = %q, want approved", resp["status"])
	}
}

// =========================================================================
// 12. ListSampleResults handler
// =========================================================================

func TestListSampleResultsNoFilters(t *testing.T) {
	m := &mockStore{
		listSampleResultsFn: func(_ context.Context, _ storage.SampleResultFilter) ([]storage.SampleResult, error) {
			return []storage.SampleResult{{ID: uuid.New()}}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/api/v1/sample-results", nil)
	w := httptest.NewRecorder()
	h.listSampleResults(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListSampleResultsInvalidLocationID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/sample-results?monitoring_location_id=bad", nil)
	w := httptest.NewRecorder()
	h.listSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListSampleResultsInvalidParameterID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/sample-results?parameter_id=bad", nil)
	w := httptest.NewRecorder()
	h.listSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListSampleResultsInvalidLimit(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/sample-results?limit=-1", nil)
	w := httptest.NewRecorder()
	h.listSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListSampleResultsInvalidStartDate(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/sample-results?start_date=bad", nil)
	w := httptest.NewRecorder()
	h.listSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListSampleResultsEmpty(t *testing.T) {
	m := &mockStore{
		listSampleResultsFn: func(_ context.Context, _ storage.SampleResultFilter) ([]storage.SampleResult, error) {
			return []storage.SampleResult{}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/api/v1/sample-results", nil)
	w := httptest.NewRecorder()
	h.listSampleResults(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

// =========================================================================
// 13. EvaluateCompliance handler
// =========================================================================

func TestEvaluateComplianceValid(t *testing.T) {
	m := &mockStore{
		evaluateComplianceFn: func(_ context.Context, _ uuid.UUID) ([]storage.ComplianceResult, error) {
			return []storage.ComplianceResult{{Compliance: "OK"}}, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.evaluateCompliance(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestEvaluateComplianceInvalidFacilityID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", "bad")
	w := httptest.NewRecorder()
	h.evaluateCompliance(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestEvaluateComplianceNoAccess(t *testing.T) {
	h := &handler{store: &mockStore{}}
	facID := uuid.New().String()
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "operator", FacilityID: uuid.New().String()}})
	req.SetPathValue("facility_id", facID)
	w := httptest.NewRecorder()
	h.evaluateCompliance(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// =========================================================================
// 14. GetTrending handler
// =========================================================================

func TestGetTrendingValid(t *testing.T) {
	m := &mockStore{
		getTrendingDataFn: func(_ context.Context, _ uuid.UUID, days int) ([]storage.TrendingSeries, error) {
			if days != 90 {
				t.Errorf("days = %d, want 90", days)
			}
			return []storage.TrendingSeries{}, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test?days=90", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.getTrending(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestGetTrendingDefaultDays(t *testing.T) {
	var capturedDays int
	m := &mockStore{
		getTrendingDataFn: func(_ context.Context, _ uuid.UUID, days int) ([]storage.TrendingSeries, error) {
			capturedDays = days
			return nil, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.getTrending(w, req)
	if capturedDays != 30 {
		t.Errorf("days = %d, want 30", capturedDays)
	}
}

func TestGetTrendingInvalidDaysUsesDefault(t *testing.T) {
	var capturedDays int
	m := &mockStore{
		getTrendingDataFn: func(_ context.Context, _ uuid.UUID, days int) ([]storage.TrendingSeries, error) {
			capturedDays = days
			return nil, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test?days=abc", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.getTrending(w, req)
	if capturedDays != 30 {
		t.Errorf("invalid days should default to 30, got %d", capturedDays)
	}
}

func TestGetTrendingDaysOverLimit(t *testing.T) {
	var capturedDays int
	m := &mockStore{
		getTrendingDataFn: func(_ context.Context, _ uuid.UUID, days int) ([]storage.TrendingSeries, error) {
			capturedDays = days
			return nil, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test?days=500", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.getTrending(w, req)
	if capturedDays != 30 {
		t.Errorf("days > 365 should default to 30, got %d", capturedDays)
	}
}

func TestGetTrendingDaysZero(t *testing.T) {
	var capturedDays int
	m := &mockStore{
		getTrendingDataFn: func(_ context.Context, _ uuid.UUID, days int) ([]storage.TrendingSeries, error) {
			capturedDays = days
			return nil, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test?days=0", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.getTrending(w, req)
	if capturedDays != 30 {
		t.Errorf("days=0 should default to 30, got %d", capturedDays)
	}
}

// =========================================================================
// 15. ListInstrumentStatuses handler
// =========================================================================

func TestListInstrumentStatusesValid(t *testing.T) {
	m := &mockStore{
		listInstrumentStatusesFn: func(_ context.Context, _ uuid.UUID) ([]storage.InstrumentStatus, error) {
			return []storage.InstrumentStatus{{Name: "pH Meter"}}, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listInstrumentStatuses(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListInstrumentStatusesInvalidFacilityID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", "bad")
	w := httptest.NewRecorder()
	h.listInstrumentStatuses(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// 16. ListCalibrationRecords handler
// =========================================================================

func TestListCalibrationRecordsValid(t *testing.T) {
	m := &mockStore{
		listCalibrationRecordsFn: func(_ context.Context, _ uuid.UUID) ([]storage.CalibrationRecord, error) {
			return []storage.CalibrationRecord{}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("instrument_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listCalibrationRecords(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListCalibrationRecordsInvalidInstrumentID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("instrument_id", "bad")
	w := httptest.NewRecorder()
	h.listCalibrationRecords(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// 17. ListAuditLog handler
// =========================================================================

func TestListAuditLogValid(t *testing.T) {
	m := &mockStore{
		listAuditLogFn: func(_ context.Context, _ uuid.UUID) ([]storage.AuditEntry, error) {
			return []storage.AuditEntry{}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("record_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listAuditLog(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListAuditLogInvalidRecordID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("record_id", "bad")
	w := httptest.NewRecorder()
	h.listAuditLog(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// 18. ListMonitoringLocations handler
// =========================================================================

func TestListMonitoringLocationsValid(t *testing.T) {
	m := &mockStore{
		listMonitoringLocationsFn: func(_ context.Context, _ uuid.UUID) ([]storage.MonitoringLocation, error) {
			return []storage.MonitoringLocation{{Name: "Effluent"}}, nil
		},
	}
	h := &handler{store: m}
	facID := uuid.New().String()
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", facID)
	w := httptest.NewRecorder()
	h.listMonitoringLocations(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListMonitoringLocationsInvalidFacilityID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", "bad")
	w := httptest.NewRecorder()
	h.listMonitoringLocations(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestListMonitoringLocationsNoAccess(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "operator", FacilityID: uuid.New().String()}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listMonitoringLocations(w, req)
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

// =========================================================================
// 19. checkFacilityAccess
// =========================================================================

func TestCheckFacilityAccessNilClaims(t *testing.T) {
	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	result := checkFacilityAccess(w, req, uuid.New())
	if result {
		t.Error("should return false for nil claims")
	}
	if w.Code != http.StatusForbidden {
		t.Errorf("status = %d, want %d", w.Code, http.StatusForbidden)
	}
}

func TestCheckFacilityAccessOrgWideRole(t *testing.T) {
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	result := checkFacilityAccess(w, req, uuid.New())
	if !result {
		t.Error("org-wide role should grant access")
	}
}

func TestCheckFacilityAccessScopedRoleMatching(t *testing.T) {
	facID := uuid.New()
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	result := checkFacilityAccess(w, req, facID)
	if !result {
		t.Error("matching scoped role should grant access")
	}
}

func TestCheckFacilityAccessScopedRoleNoMatch(t *testing.T) {
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "operator", FacilityID: uuid.New().String()}})
	w := httptest.NewRecorder()
	result := checkFacilityAccess(w, req, uuid.New())
	if result {
		t.Error("non-matching scoped role should deny access")
	}
}

// =========================================================================
// 20. ImportSampleResults handler
// =========================================================================

func TestImportSampleResultsInvalidOrgID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("POST", "/test", nil)
	req.SetPathValue("org_id", "bad")
	w := httptest.NewRecorder()
	h.importSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportSampleResultsMissingFile(t *testing.T) {
	h := &handler{store: &mockStore{}}
	// Create multipart form without file
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	writer.WriteField("entered_by", uuid.New().String())
	writer.Close()

	req := httptest.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.importSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportSampleResultsMissingEnteredBy(t *testing.T) {
	h := &handler{store: &mockStore{
		listAllMonitoringLocsFn: func(_ context.Context, _ uuid.UUID) ([]storage.MonitoringLocation, error) {
			return nil, nil
		},
		listParametersFn: func(_ context.Context, _ uuid.UUID) ([]storage.Parameter, error) {
			return nil, nil
		},
		listUnitsFn: func(_ context.Context, _ uuid.UUID) ([]storage.UnitOfMeasure, error) {
			return nil, nil
		},
	}}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "data.csv")
	part.Write([]byte("monitoring_location,parameter_code,collected_at,result_value,unit_code\n"))
	writer.Close()

	req := httptest.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.importSampleResults(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestImportSampleResultsSuccess(t *testing.T) {
	locID := uuid.New()
	paramID := uuid.New()
	unitID := uuid.New()
	m := &mockStore{
		listAllMonitoringLocsFn: func(_ context.Context, _ uuid.UUID) ([]storage.MonitoringLocation, error) {
			return []storage.MonitoringLocation{{ID: locID, Name: "Effluent"}}, nil
		},
		listParametersFn: func(_ context.Context, _ uuid.UUID) ([]storage.Parameter, error) {
			return []storage.Parameter{{ID: paramID, Code: "PH"}}, nil
		},
		listUnitsFn: func(_ context.Context, _ uuid.UUID) ([]storage.UnitOfMeasure, error) {
			return []storage.UnitOfMeasure{{ID: unitID, Code: "SU"}}, nil
		},
		createSampleResultFn: func(_ context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error) {
			return storage.SampleResult{ID: uuid.New(), Status: "draft"}, nil
		},
		getOrganizationIDForResFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	h := &handler{store: m}

	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)
	part, _ := writer.CreateFormFile("file", "data.csv")
	part.Write([]byte("monitoring_location,parameter_code,collected_at,result_value,unit_code\nEffluent,PH,2025-06-01,7.2,SU\n"))
	writer.WriteField("entered_by", uuid.New().String())
	writer.Close()

	req := httptest.NewRequest("POST", "/test", &buf)
	req.Header.Set("Content-Type", writer.FormDataContentType())
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.importSampleResults(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	resp := decodeBody(t, w.Body)
	if resp["imported"] != float64(1) {
		t.Errorf("imported = %v, want 1", resp["imported"])
	}
}

// =========================================================================
// 21. ListValidationRules handler
// =========================================================================

func TestListValidationRulesValid(t *testing.T) {
	m := &mockStore{
		listValidationRulesFn: func(_ context.Context, _ uuid.UUID) ([]storage.ValidationRule, error) {
			return []storage.ValidationRule{}, nil
		},
	}
	h := &handler{store: m}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("org_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.listValidationRules(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
}

func TestListValidationRulesInvalidOrgID(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/test", nil)
	req.SetPathValue("org_id", "bad")
	w := httptest.NewRecorder()
	h.listValidationRules(w, req)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

// =========================================================================
// 22. Compliance report handlers
// =========================================================================

func TestComplianceExcelValid(t *testing.T) {
	v := 7.0
	m := &mockStore{
		evaluateComplianceFn: func(_ context.Context, _ uuid.UUID) ([]storage.ComplianceResult, error) {
			return []storage.ComplianceResult{{FacilityName: "F", ParameterName: "P", ResultValue: &v, UnitCode: "SU", CollectedAt: time.Now(), LimitType: "daily_max", LimitValue: 9, Compliance: "OK"}}, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.complianceExcel(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/vnd.openxmlformats-officedocument.spreadsheetml.sheet" {
		t.Errorf("Content-Type = %q", ct)
	}
	if cd := w.Header().Get("Content-Disposition"); cd == "" {
		t.Error("expected Content-Disposition header")
	}
}

func TestCompliancePDFValid(t *testing.T) {
	m := &mockStore{
		evaluateComplianceFn: func(_ context.Context, _ uuid.UUID) ([]storage.ComplianceResult, error) {
			return nil, nil
		},
	}
	h := &handler{store: m}
	req := withAuthCtx(httptest.NewRequest("GET", "/test", nil), []auth.RoleClaim{{Role: "admin"}})
	req.SetPathValue("facility_id", uuid.New().String())
	w := httptest.NewRecorder()
	h.compliancePDF(w, req)
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q", ct)
	}
}

// =========================================================================
// 23. writeJSON / writeError helpers
// =========================================================================

func TestWriteJSON(t *testing.T) {
	w := httptest.NewRecorder()
	writeJSON(w, http.StatusCreated, map[string]string{"key": "val"})
	if w.Code != http.StatusCreated {
		t.Errorf("status = %d, want %d", w.Code, http.StatusCreated)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/json" {
		t.Errorf("Content-Type = %q", ct)
	}
}

func TestWriteError(t *testing.T) {
	w := httptest.NewRecorder()
	writeError(w, http.StatusBadRequest, "something went wrong")
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
	body := decodeBody(t, w.Body)
	if body["error"] != "something went wrong" {
		t.Errorf("error = %q", body["error"])
	}
}
