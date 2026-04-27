package api

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/carlos-loya/water-quality-data-management/internal/auth"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
)

// ============================================================================
// GET /api/v1/alerts
// ============================================================================

func TestListAlerts_OrgWideAdmin_ReturnsAll(t *testing.T) {
	fac1, fac2 := uuid.New(), uuid.New()
	store := &mockStore{
		listAlertsFn: func(_ context.Context, _ storage.AlertFilter) ([]storage.Alert, error) {
			return []storage.Alert{
				{ID: uuid.New(), FacilityID: fac1, Type: "exceedance", Severity: "critical", Message: "a1"},
				{ID: uuid.New(), FacilityID: fac2, Type: "overdue_calibration", Severity: "warning", Message: "a2"},
			}, nil
		},
	}
	h := &handler{store: store}

	req := httptest.NewRequest("GET", "/api/v1/alerts", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}}) // org-wide
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	arr := decodeArray(t, w.Body)
	if len(arr) != 2 {
		t.Fatalf("expected 2 alerts, got %d", len(arr))
	}
}

func TestListAlerts_FacilityScopedUser_FiltersResponse(t *testing.T) {
	fac1, fac2 := uuid.New(), uuid.New()
	store := &mockStore{
		listAlertsFn: func(_ context.Context, _ storage.AlertFilter) ([]storage.Alert, error) {
			return []storage.Alert{
				{ID: uuid.New(), FacilityID: fac1, Type: "exceedance", Severity: "critical", Message: "a1"},
				{ID: uuid.New(), FacilityID: fac2, Type: "exceedance", Severity: "critical", Message: "a2"},
			}, nil
		},
	}
	h := &handler{store: store}

	req := httptest.NewRequest("GET", "/api/v1/alerts", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: fac1.String()}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	arr := decodeArray(t, w.Body)
	if len(arr) != 1 {
		t.Fatalf("expected 1 alert (filtered by facility access), got %d", len(arr))
	}
	first := arr[0].(map[string]any)
	if first["facility_id"] != fac1.String() {
		t.Errorf("expected filtered alert to be for fac1, got %v", first["facility_id"])
	}
}

func TestListAlerts_WithFacilityIDFilter_PassesToStore(t *testing.T) {
	facID := uuid.New()
	var received storage.AlertFilter
	store := &mockStore{
		listAlertsFn: func(_ context.Context, f storage.AlertFilter) ([]storage.Alert, error) {
			received = f
			return []storage.Alert{}, nil
		},
	}
	h := &handler{store: store}

	req := httptest.NewRequest("GET", "/api/v1/alerts?facility_id="+facID.String(), nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	if received.FacilityID == nil || *received.FacilityID != facID {
		t.Errorf("expected filter.FacilityID=%s, got %v", facID, received.FacilityID)
	}
}

func TestListAlerts_WithFacilityIDFilter_403WhenNoAccess(t *testing.T) {
	facID := uuid.New()
	store := &mockStore{}
	h := &handler{store: store}

	req := httptest.NewRequest("GET", "/api/v1/alerts?facility_id="+facID.String(), nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: uuid.NewString()}}) // different facility
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestListAlerts_InvalidFacilityID_400(t *testing.T) {
	h := &handler{store: &mockStore{}}

	req := httptest.NewRequest("GET", "/api/v1/alerts?facility_id=not-a-uuid", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListAlerts_InvalidType_400(t *testing.T) {
	h := &handler{store: &mockStore{}}

	req := httptest.NewRequest("GET", "/api/v1/alerts?type=weather", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListAlerts_ValidType_PassesThrough(t *testing.T) {
	var received storage.AlertFilter
	store := &mockStore{
		listAlertsFn: func(_ context.Context, f storage.AlertFilter) ([]storage.Alert, error) {
			received = f
			return []storage.Alert{}, nil
		},
	}
	h := &handler{store: store}

	req := httptest.NewRequest("GET", "/api/v1/alerts?type=exceedance", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200", w.Code)
	}
	if received.Type == nil || *received.Type != "exceedance" {
		t.Errorf("expected type=exceedance, got %v", received.Type)
	}
}

func TestListAlerts_DismissedFilter(t *testing.T) {
	cases := []struct {
		in   string
		want bool
	}{{"true", true}, {"false", false}}
	for _, tc := range cases {
		t.Run(tc.in, func(t *testing.T) {
			var received storage.AlertFilter
			store := &mockStore{
				listAlertsFn: func(_ context.Context, f storage.AlertFilter) ([]storage.Alert, error) {
					received = f
					return []storage.Alert{}, nil
				},
			}
			h := &handler{store: store}
			req := httptest.NewRequest("GET", "/api/v1/alerts?dismissed="+tc.in, nil)
			req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
			w := httptest.NewRecorder()
			h.listAlerts(w, req)
			if w.Code != http.StatusOK {
				t.Fatalf("status = %d, want 200", w.Code)
			}
			if received.Dismissed == nil || *received.Dismissed != tc.want {
				t.Errorf("expected dismissed=%v, got %v", tc.want, received.Dismissed)
			}
		})
	}
}

func TestListAlerts_InvalidDismissed_400(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/alerts?dismissed=maybe", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListAlerts_InvalidLimit_400(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/alerts?limit=-5", nil)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.listAlerts(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListAlerts_Unauthenticated_401(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("GET", "/api/v1/alerts", nil) // no auth context
	w := httptest.NewRecorder()
	h.listAlerts(w, req)
	if w.Code != http.StatusUnauthorized {
		t.Fatalf("status = %d, want 401", w.Code)
	}
}

// ============================================================================
// POST /api/v1/alerts/{id}/dismiss
// ============================================================================

func TestDismissAlert_HappyPath(t *testing.T) {
	alertID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getAlertFn: func(_ context.Context, id uuid.UUID) (storage.Alert, error) {
			if id != alertID {
				return storage.Alert{}, pgx.ErrNoRows
			}
			return storage.Alert{ID: alertID, FacilityID: facID, Type: "exceedance", Severity: "critical", Message: "msg"}, nil
		},
		dismissAlertFn: func(_ context.Context, id, userID uuid.UUID) (storage.Alert, error) {
			return storage.Alert{ID: id, FacilityID: facID, Type: "exceedance", Severity: "critical", Message: "msg", DismissedBy: &userID}, nil
		},
	}
	h := &handler{store: store}

	req := httptest.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/dismiss", nil)
	req.SetPathValue("id", alertID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.dismissAlert(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200, body=%s", w.Code, w.Body.String())
	}
	var body map[string]any
	if err := json.Unmarshal(w.Body.Bytes(), &body); err != nil {
		t.Fatalf("decode body: %v", err)
	}
	if body["dismissed_by"] == nil {
		t.Error("expected dismissed_by to be set in response")
	}
}

func TestDismissAlert_InvalidUUID_400(t *testing.T) {
	h := &handler{store: &mockStore{}}
	req := httptest.NewRequest("POST", "/api/v1/alerts/not-a-uuid/dismiss", nil)
	req.SetPathValue("id", "not-a-uuid")
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator"}})
	w := httptest.NewRecorder()
	h.dismissAlert(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestDismissAlert_NotFound_404(t *testing.T) {
	store := &mockStore{
		getAlertFn: func(_ context.Context, _ uuid.UUID) (storage.Alert, error) {
			return storage.Alert{}, pgx.ErrNoRows
		},
	}
	h := &handler{store: store}
	id := uuid.New()
	req := httptest.NewRequest("POST", "/api/v1/alerts/"+id.String()+"/dismiss", nil)
	req.SetPathValue("id", id.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.dismissAlert(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDismissAlert_NoFacilityAccess_403(t *testing.T) {
	alertID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getAlertFn: func(_ context.Context, _ uuid.UUID) (storage.Alert, error) {
			return storage.Alert{ID: alertID, FacilityID: facID}, nil
		},
	}
	h := &handler{store: store}
	req := httptest.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/dismiss", nil)
	req.SetPathValue("id", alertID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: uuid.NewString()}}) // wrong facility
	w := httptest.NewRecorder()
	h.dismissAlert(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestDismissAlert_AlreadyDismissed_409(t *testing.T) {
	alertID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getAlertFn: func(_ context.Context, _ uuid.UUID) (storage.Alert, error) {
			return storage.Alert{ID: alertID, FacilityID: facID}, nil
		},
		dismissAlertFn: func(_ context.Context, _ uuid.UUID, _ uuid.UUID) (storage.Alert, error) {
			return storage.Alert{}, pgx.ErrNoRows
		},
	}
	h := &handler{store: store}
	req := httptest.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/dismiss", nil)
	req.SetPathValue("id", alertID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.dismissAlert(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

// ============================================================================
// RBAC on the dismiss route — viewer must be blocked before reaching handler.
// This exercises the router-level requireAnyRole wrapper.
// ============================================================================

func TestDismissAlert_ViewerForbidden_ViaRouter(t *testing.T) {
	alertID := uuid.New()
	store := &mockStore{} // not expected to be called
	router := NewRouter(store, nil, nil, testSecret)

	token := makeToken(t, []auth.RoleClaim{{Role: "viewer"}})
	req := httptest.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/dismiss", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (viewers cannot dismiss); body=%s", w.Code, w.Body.String())
	}
}

func TestDismissAlert_OperatorAllowed_ViaRouter(t *testing.T) {
	alertID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getAlertFn: func(_ context.Context, _ uuid.UUID) (storage.Alert, error) {
			return storage.Alert{ID: alertID, FacilityID: facID}, nil
		},
		dismissAlertFn: func(_ context.Context, id, userID uuid.UUID) (storage.Alert, error) {
			return storage.Alert{ID: id, FacilityID: facID, DismissedBy: &userID}, nil
		},
	}
	router := NewRouter(store, nil, nil, testSecret)

	token := makeToken(t, []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	req := httptest.NewRequest("POST", "/api/v1/alerts/"+alertID.String()+"/dismiss", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, want 200; body=%s", w.Code, w.Body.String())
	}
}
