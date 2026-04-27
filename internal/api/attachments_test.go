package api

import (
	"bytes"
	"context"
	"encoding/json"
	"io"
	"mime/multipart"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/carlos-loya/water-quality-data-management/internal/auth"
	"github.com/carlos-loya/water-quality-data-management/internal/storage"
	"github.com/carlos-loya/water-quality-data-management/internal/storage/blob"
)

// fakeBlobs is an in-memory blob.Store for handler tests.
type fakeBlobs struct {
	mu       sync.Mutex
	objects  map[string][]byte
	putErr   error
	openErr  error
	openMiss bool
}

func newFakeBlobs() *fakeBlobs {
	return &fakeBlobs{objects: map[string][]byte{}}
}

func (f *fakeBlobs) Put(_ context.Context, key, _ string, _ int64, r io.Reader) error {
	if f.putErr != nil {
		return f.putErr
	}
	b, err := io.ReadAll(r)
	if err != nil {
		return err
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	f.objects[key] = b
	return nil
}

func (f *fakeBlobs) Open(_ context.Context, key string) (io.ReadCloser, error) {
	if f.openErr != nil {
		return nil, f.openErr
	}
	f.mu.Lock()
	defer f.mu.Unlock()
	b, ok := f.objects[key]
	if !ok || f.openMiss {
		return nil, blob.ErrNotFound
	}
	return io.NopCloser(bytes.NewReader(b)), nil
}

func (f *fakeBlobs) Delete(_ context.Context, key string) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	delete(f.objects, key)
	return nil
}

// --- helpers ---

func multipartBody(t *testing.T, fieldName, filename, contentType string, data []byte) (*bytes.Buffer, string) {
	t.Helper()
	var buf bytes.Buffer
	mw := multipart.NewWriter(&buf)
	h := make(map[string][]string)
	h["Content-Disposition"] = []string{`form-data; name="` + fieldName + `"; filename="` + filename + `"`}
	h["Content-Type"] = []string{contentType}
	part, err := mw.CreatePart(h)
	if err != nil {
		t.Fatalf("create part: %v", err)
	}
	if _, err := part.Write(data); err != nil {
		t.Fatalf("write part: %v", err)
	}
	mw.Close()
	return &buf, mw.FormDataContentType()
}

func newPDFUploadReq(t *testing.T, path string, resultID uuid.UUID, data []byte) *http.Request {
	t.Helper()
	body, ct := multipartBody(t, "file", "evidence.pdf", "application/pdf", data)
	req := httptest.NewRequest("POST", path, body)
	req.Header.Set("Content-Type", ct)
	req.SetPathValue("id", resultID.String())
	return req
}

// =========================================================================
// POST /api/v1/sample-results/{id}/attachments
// =========================================================================

func TestUploadAttachment_HappyPath(t *testing.T) {
	resultID := uuid.New()
	locID := uuid.New()
	facID := uuid.New()

	var saved storage.CreateAttachmentParams
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, id uuid.UUID) (storage.SampleResult, error) {
			if id != resultID {
				return storage.SampleResult{}, pgx.ErrNoRows
			}
			return storage.SampleResult{ID: resultID, MonitoringLocationID: locID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
		createAttachmentFn: func(_ context.Context, p storage.CreateAttachmentParams) (storage.Attachment, error) {
			saved = p
			return storage.Attachment{
				ID: uuid.New(), OrganizationID: p.OrganizationID,
				SubjectType: p.SubjectType, SubjectID: p.SubjectID,
				Filename: p.Filename, ContentType: p.ContentType, SizeBytes: p.SizeBytes,
				StorageKey: p.StorageKey, UploadedBy: p.UploadedBy, UploadedAt: time.Now(),
			}, nil
		},
	}
	blobs := newFakeBlobs()
	h := &handler{store: store, blobs: blobs}

	req := newPDFUploadReq(t, "/api/v1/sample-results/"+resultID.String()+"/attachments", resultID, []byte("%PDF-1.4\n..."))
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.uploadAttachment(w, req)

	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if saved.SubjectType != "sample_result" || saved.SubjectID != resultID {
		t.Errorf("saved subject = %s/%s", saved.SubjectType, saved.SubjectID)
	}
	if saved.Filename != "evidence.pdf" || saved.ContentType != "application/pdf" {
		t.Errorf("saved filename/content-type = %s/%s", saved.Filename, saved.ContentType)
	}
	if len(blobs.objects) != 1 {
		t.Errorf("expected 1 blob written, got %d", len(blobs.objects))
	}
}

func TestUploadAttachment_RejectsDisallowedContentType(t *testing.T) {
	resultID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
	}
	blobs := newFakeBlobs()
	h := &handler{store: store, blobs: blobs}

	body, ct := multipartBody(t, "file", "script.sh", "application/x-sh", []byte("#!/bin/sh"))
	req := httptest.NewRequest("POST", "/x", body)
	req.Header.Set("Content-Type", ct)
	req.SetPathValue("id", resultID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.uploadAttachment(w, req)

	if w.Code != http.StatusUnsupportedMediaType {
		t.Fatalf("status = %d, want 415", w.Code)
	}
	if len(blobs.objects) != 0 {
		t.Errorf("blob store modified on rejected upload")
	}
}

func TestUploadAttachment_RejectsOversized(t *testing.T) {
	resultID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
	}
	h := &handler{store: store, blobs: newFakeBlobs()}

	// Exceed the 10 MiB cap.
	big := bytes.Repeat([]byte("x"), maxUploadBytes+2048)
	req := newPDFUploadReq(t, "/x", resultID, big)
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.uploadAttachment(w, req)

	if w.Code != http.StatusBadRequest && w.Code != http.StatusRequestEntityTooLarge {
		t.Fatalf("status = %d, want 400 or 413", w.Code)
	}
}

func TestUploadAttachment_SampleResultNotFound(t *testing.T) {
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{}, pgx.ErrNoRows
		},
	}
	h := &handler{store: store, blobs: newFakeBlobs()}

	req := newPDFUploadReq(t, "/x", uuid.New(), []byte("pdf"))
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.uploadAttachment(w, req)

	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestUploadAttachment_FacilityScopeForbidden(t *testing.T) {
	resultID := uuid.New()
	ownerFac := uuid.New()
	otherFac := uuid.New()
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return ownerFac, nil
		},
	}
	h := &handler{store: store, blobs: newFakeBlobs()}

	req := newPDFUploadReq(t, "/x", resultID, []byte("pdf"))
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator", FacilityID: otherFac.String()}})
	w := httptest.NewRecorder()
	h.uploadAttachment(w, req)

	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403", w.Code)
	}
}

func TestUploadAttachment_InvalidID(t *testing.T) {
	h := &handler{store: &mockStore{}, blobs: newFakeBlobs()}
	body, ct := multipartBody(t, "file", "a.pdf", "application/pdf", []byte("pdf"))
	req := httptest.NewRequest("POST", "/x", body)
	req.Header.Set("Content-Type", ct)
	req.SetPathValue("id", "not-a-uuid")
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.uploadAttachment(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

// =========================================================================
// GET /api/v1/sample-results/{id}/attachments
// =========================================================================

func TestListAttachments_HappyPath(t *testing.T) {
	resultID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
		listAttachmentsFn: func(_ context.Context, st string, id uuid.UUID) ([]storage.Attachment, error) {
			if st != "sample_result" || id != resultID {
				t.Errorf("unexpected args: %s/%s", st, id)
			}
			return []storage.Attachment{{ID: uuid.New(), Filename: "a.pdf"}, {ID: uuid.New(), Filename: "b.pdf"}}, nil
		},
	}
	h := &handler{store: store}
	req := httptest.NewRequest("GET", "/x", nil)
	req.SetPathValue("id", resultID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "viewer", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.listAttachments(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	arr := decodeArray(t, w.Body)
	if len(arr) != 2 {
		t.Fatalf("expected 2, got %d", len(arr))
	}
}

// =========================================================================
// GET /api/v1/attachments/{id}
// =========================================================================

func TestDownloadAttachment_HappyPath(t *testing.T) {
	attID := uuid.New()
	resultID := uuid.New()
	facID := uuid.New()
	want := []byte("file bytes")
	blobs := newFakeBlobs()
	blobs.objects["storage-key-1"] = want

	store := &mockStore{
		getAttachmentFn: func(_ context.Context, id uuid.UUID) (storage.Attachment, error) {
			return storage.Attachment{
				ID: id, SubjectType: "sample_result", SubjectID: resultID,
				Filename: "evidence.pdf", ContentType: "application/pdf",
				SizeBytes: int64(len(want)), StorageKey: "storage-key-1",
			}, nil
		},
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
	}
	h := &handler{store: store, blobs: blobs}

	req := httptest.NewRequest("GET", "/x", nil)
	req.SetPathValue("id", attID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "viewer", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.downloadAttachment(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if got := w.Body.Bytes(); !bytes.Equal(got, want) {
		t.Errorf("bytes = %q, want %q", got, want)
	}
	if ct := w.Header().Get("Content-Type"); ct != "application/pdf" {
		t.Errorf("Content-Type = %q", ct)
	}
	if disp := w.Header().Get("Content-Disposition"); !strings.Contains(disp, "evidence.pdf") {
		t.Errorf("Content-Disposition = %q", disp)
	}
}

func TestDownloadAttachment_404WhenDeleted(t *testing.T) {
	facID := uuid.New()
	now := time.Now()
	store := &mockStore{
		getAttachmentFn: func(_ context.Context, id uuid.UUID) (storage.Attachment, error) {
			return storage.Attachment{ID: id, SubjectType: "sample_result", SubjectID: uuid.New(), DeletedAt: &now}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
	}
	h := &handler{store: store, blobs: newFakeBlobs()}
	req := httptest.NewRequest("GET", "/x", nil)
	req.SetPathValue("id", uuid.NewString())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.downloadAttachment(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

func TestDownloadAttachment_404WhenMissing(t *testing.T) {
	store := &mockStore{} // getAttachment returns pgx.ErrNoRows by default
	h := &handler{store: store, blobs: newFakeBlobs()}
	req := httptest.NewRequest("GET", "/x", nil)
	req.SetPathValue("id", uuid.NewString())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.downloadAttachment(w, req)
	if w.Code != http.StatusNotFound {
		t.Fatalf("status = %d, want 404", w.Code)
	}
}

// =========================================================================
// DELETE /api/v1/attachments/{id}
// =========================================================================

func TestDeleteAttachment_HappyPath(t *testing.T) {
	facID := uuid.New()
	called := false
	store := &mockStore{
		getAttachmentFn: func(_ context.Context, id uuid.UUID) (storage.Attachment, error) {
			return storage.Attachment{ID: id, SubjectType: "sample_result", SubjectID: uuid.New(), StorageKey: "k"}, nil
		},
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
		softDeleteAttachmentFn: func(_ context.Context, id, _ uuid.UUID) (storage.Attachment, error) {
			called = true
			now := time.Now()
			return storage.Attachment{ID: id, DeletedAt: &now}, nil
		},
	}
	h := &handler{store: store, blobs: newFakeBlobs()}
	req := httptest.NewRequest("DELETE", "/x", nil)
	req.SetPathValue("id", uuid.NewString())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "reviewer", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.deleteAttachment(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if !called {
		t.Error("SoftDeleteAttachment was not called")
	}
}

func TestDeleteAttachment_409IfAlreadyDeleted(t *testing.T) {
	now := time.Now()
	store := &mockStore{
		getAttachmentFn: func(_ context.Context, id uuid.UUID) (storage.Attachment, error) {
			return storage.Attachment{ID: id, DeletedAt: &now}, nil
		},
	}
	h := &handler{store: store, blobs: newFakeBlobs()}
	req := httptest.NewRequest("DELETE", "/x", nil)
	req.SetPathValue("id", uuid.NewString())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.deleteAttachment(w, req)
	if w.Code != http.StatusConflict {
		t.Fatalf("status = %d, want 409", w.Code)
	}
}

func TestDeleteAttachment_OperatorForbiddenViaRouter(t *testing.T) {
	// Operators can upload but can't delete; reviewer/admin only.
	store := &mockStore{}
	router := NewRouter(store, nil, newFakeBlobs(), testSecret)
	token := makeToken(t, []auth.RoleClaim{{Role: "operator"}})
	req := httptest.NewRequest("DELETE", "/api/v1/attachments/"+uuid.NewString(), nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	if w.Code != http.StatusForbidden {
		t.Fatalf("status = %d, want 403 (operator); body=%s", w.Code, w.Body.String())
	}
}

// =========================================================================
// Comments
// =========================================================================

func TestCreateComment_HappyPath(t *testing.T) {
	resultID := uuid.New()
	facID := uuid.New()
	var saved storage.CreateCommentParams
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
		createCommentFn: func(_ context.Context, p storage.CreateCommentParams) (storage.Comment, error) {
			saved = p
			return storage.Comment{ID: uuid.New(), Body: p.Body, CreatedAt: time.Now()}, nil
		},
	}
	h := &handler{store: store}
	body := bytes.NewBufferString(`{"body":"looks correct"}`)
	req := httptest.NewRequest("POST", "/x", body)
	req.SetPathValue("id", resultID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "viewer", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.createComment(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, body = %s", w.Code, w.Body.String())
	}
	if saved.Body != "looks correct" || saved.SubjectID != resultID {
		t.Errorf("saved body = %q, subject = %s", saved.Body, saved.SubjectID)
	}
}

func TestCreateComment_RejectsEmptyBody(t *testing.T) {
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{}, nil
		},
	}
	h := &handler{store: store}
	for _, raw := range []string{`{"body":""}`, `{"body":"   "}`, `{}`} {
		req := httptest.NewRequest("POST", "/x", bytes.NewBufferString(raw))
		req.SetPathValue("id", uuid.NewString())
		req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
		w := httptest.NewRecorder()
		h.createComment(w, req)
		if w.Code != http.StatusBadRequest {
			t.Errorf("body %q: status = %d, want 400", raw, w.Code)
		}
	}
}

func TestCreateComment_RejectsOversized(t *testing.T) {
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{}, nil
		},
	}
	h := &handler{store: store}
	huge := strings.Repeat("x", 4001)
	raw, _ := json.Marshal(map[string]string{"body": huge})
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(raw))
	req.SetPathValue("id", uuid.NewString())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "admin"}})
	w := httptest.NewRecorder()
	h.createComment(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400", w.Code)
	}
}

func TestListComments_HappyPath(t *testing.T) {
	resultID := uuid.New()
	facID := uuid.New()
	store := &mockStore{
		getSampleResultFn: func(_ context.Context, _ uuid.UUID) (storage.SampleResult, error) {
			return storage.SampleResult{ID: resultID}, nil
		},
		getFacilityIDForLocationFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return facID, nil
		},
		listCommentsFn: func(_ context.Context, _ string, _ uuid.UUID) ([]storage.Comment, error) {
			return []storage.Comment{{ID: uuid.New(), Body: "first"}}, nil
		},
	}
	h := &handler{store: store}
	req := httptest.NewRequest("GET", "/x", nil)
	req.SetPathValue("id", resultID.String())
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "viewer", FacilityID: facID.String()}})
	w := httptest.NewRecorder()
	h.listComments(w, req)
	if w.Code != http.StatusOK {
		t.Fatalf("status = %d", w.Code)
	}
	arr := decodeArray(t, w.Body)
	if len(arr) != 1 {
		t.Fatalf("len = %d, want 1", len(arr))
	}
}

// =========================================================================
// Override reason enforcement on createSampleResult
// =========================================================================

func TestCreateSampleResult_OutOfRangeWithoutOverrideRejected(t *testing.T) {
	paramID := uuid.New()
	minV, maxV := 6.5, 8.5
	store := &mockStore{
		getValidationRuleFn: func(_ context.Context, _ uuid.UUID) (storage.ValidationRule, error) {
			return storage.ValidationRule{
				ParameterID: paramID, MinValue: &minV, MaxValue: &maxV, Active: true,
			}, nil
		},
	}
	h := &handler{store: store}
	v := 9.9
	params := storage.CreateSampleResultParams{
		MonitoringLocationID: uuid.New(),
		ParameterID:          paramID,
		UnitID:               uuid.New(),
		CollectedAt:          time.Now(),
		EnteredBy:            uuid.New(),
		ResultValue:          &v,
	}
	raw, _ := json.Marshal(params)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(raw))
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator"}})
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want 400; body=%s", w.Code, w.Body.String())
	}
	if body := w.Body.String(); !strings.Contains(body, "override_reason") {
		t.Errorf("error message should mention override_reason, got %s", body)
	}
}

func TestCreateSampleResult_OutOfRangeWithOverrideAccepted(t *testing.T) {
	paramID := uuid.New()
	minV, maxV := 6.5, 8.5
	created := false
	store := &mockStore{
		getValidationRuleFn: func(_ context.Context, _ uuid.UUID) (storage.ValidationRule, error) {
			return storage.ValidationRule{MinValue: &minV, MaxValue: &maxV, Active: true}, nil
		},
		createSampleResultFn: func(_ context.Context, p storage.CreateSampleResultParams) (storage.SampleResult, error) {
			created = true
			return storage.SampleResult{ID: uuid.New(), OverrideReason: p.OverrideReason, EnteredBy: p.EnteredBy}, nil
		},
		getOrganizationIDForResFn: func(_ context.Context, _ uuid.UUID) (uuid.UUID, error) {
			return uuid.New(), nil
		},
	}
	h := &handler{store: store}
	v := 9.9
	reason := "field probe recalibration pending"
	params := storage.CreateSampleResultParams{
		MonitoringLocationID: uuid.New(),
		ParameterID:          paramID,
		UnitID:               uuid.New(),
		CollectedAt:          time.Now(),
		EnteredBy:            uuid.New(),
		ResultValue:          &v,
		OverrideReason:       &reason,
	}
	raw, _ := json.Marshal(params)
	req := httptest.NewRequest("POST", "/x", bytes.NewReader(raw))
	req = withAuthCtx(req, []auth.RoleClaim{{Role: "operator"}})
	w := httptest.NewRecorder()
	h.createSampleResult(w, req)
	if w.Code != http.StatusCreated {
		t.Fatalf("status = %d, want 201; body=%s", w.Code, w.Body.String())
	}
	if !created {
		t.Error("CreateSampleResult was not called")
	}
}
