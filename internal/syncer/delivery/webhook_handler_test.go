package delivery_test

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"vault-agent/internal/syncer/delivery"
)

// mockMutator 是 delivery.Mutator 的測試替身。
type mockMutator struct {
	patch []byte
	err   error
}

func (m *mockMutator) Execute(_ context.Context, _ *admissionv1.AdmissionRequest) ([]byte, error) {
	return m.patch, m.err
}

// makeReviewBody 建立含有指定 UID 的 AdmissionReview JSON。
func makeReviewBody(uid string) []byte {
	r := admissionv1.AdmissionReview{
		TypeMeta: metav1.TypeMeta{
			APIVersion: "admission.k8s.io/v1",
			Kind:       "AdmissionReview",
		},
		Request: &admissionv1.AdmissionRequest{
			UID: types.UID(uid),
		},
	}
	b, _ := json.Marshal(r)
	return b
}

func newHandler(m delivery.Mutator) *delivery.WebhookHandler {
	return delivery.NewWebhookHandler(m)
}

func TestWebhookHandler_MethodNotAllowed(t *testing.T) {
	t.Parallel()
	h := newHandler(&mockMutator{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodGet, "/mutate", nil)
	h.ServeHTTP(w, r)
	if w.Code != http.StatusMethodNotAllowed {
		t.Errorf("status = %d, want %d", w.Code, http.StatusMethodNotAllowed)
	}
}

func TestWebhookHandler_BadRequestBody(t *testing.T) {
	t.Parallel()
	h := newHandler(&mockMutator{})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBufferString("not-json"))
	h.ServeHTTP(w, r)
	if w.Code != http.StatusBadRequest {
		t.Errorf("status = %d, want %d", w.Code, http.StatusBadRequest)
	}
}

func TestWebhookHandler_MutatorError_ReturnsAllowedFalse(t *testing.T) {
	t.Parallel()
	h := newHandler(&mockMutator{err: errors.New("vault down")})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBuffer(makeReviewBody("uid-1")))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Response.Allowed {
		t.Error("expected Allowed=false on mutator error")
	}
	if resp.Response.UID != "uid-1" {
		t.Errorf("UID = %q, want uid-1", resp.Response.UID)
	}
	if resp.Response.Result == nil || resp.Response.Result.Message == "" {
		t.Error("expected non-empty error message in Result")
	}
}

func TestWebhookHandler_EmptyPatch_AllowedWithoutPatch(t *testing.T) {
	t.Parallel()
	h := newHandler(&mockMutator{patch: []byte("[]")})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBuffer(makeReviewBody("uid-2")))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Response.Allowed {
		t.Error("expected Allowed=true for empty patch")
	}
	if resp.Response.Patch != nil {
		t.Errorf("expected nil Patch, got %s", resp.Response.Patch)
	}
	if resp.Response.PatchType != nil {
		t.Errorf("expected nil PatchType, got %v", resp.Response.PatchType)
	}
}

func TestWebhookHandler_WithPatch_PatchTypeIsJSONPatch(t *testing.T) {
	t.Parallel()
	patch := []byte(`[{"op":"add","path":"/spec/containers/0/env/-","value":{"name":"K","value":"V"}}]`)
	h := newHandler(&mockMutator{patch: patch})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBuffer(makeReviewBody("uid-3")))

	h.ServeHTTP(w, r)

	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", w.Code)
	}
	var resp admissionv1.AdmissionReview
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if !resp.Response.Allowed {
		t.Error("expected Allowed=true")
	}
	if resp.Response.Patch == nil {
		t.Error("expected non-nil Patch")
	}
	if resp.Response.PatchType == nil || *resp.Response.PatchType != admissionv1.PatchTypeJSONPatch {
		t.Errorf("PatchType = %v, want JSONPatch", resp.Response.PatchType)
	}
}

func TestWebhookHandler_ContentType(t *testing.T) {
	t.Parallel()
	h := newHandler(&mockMutator{patch: []byte("[]")})
	w := httptest.NewRecorder()
	r := httptest.NewRequest(http.MethodPost, "/mutate", bytes.NewBuffer(makeReviewBody("uid-4")))

	h.ServeHTTP(w, r)

	ct := w.Header().Get("Content-Type")
	if ct != "application/json" {
		t.Errorf("Content-Type = %q, want application/json", ct)
	}
}
