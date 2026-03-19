// Package delivery 提供 WebhookHandler 實作，用於處理 K8s Mutating Webhook 的請求解析與回應。
package delivery

import (
	"context"
	"encoding/json"
	"net/http"

	"github.com/google/uuid"
	"github.com/vincent119/zlogger"
	admissionv1 "k8s.io/api/admission/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"vault-agent/internal/infra/metrics"
)

const maxRequestBodyBytes = 1 << 20 // 1 MiB，防止惡意大 payload

// Mutator 定義 Webhook 所需的 mutation 行為介面，方便測試時替換 mock。
type Mutator interface {
	Execute(ctx context.Context, req *admissionv1.AdmissionRequest) ([]byte, error)
}

// WebhookHandler 實作 /mutate HTTP endpoint 的請求解析與回應。
type WebhookHandler struct {
	mutator Mutator
}

// NewWebhookHandler 建立 WebhookHandler。
func NewWebhookHandler(mutator Mutator) *WebhookHandler {
	return &WebhookHandler{mutator: mutator}
}

// ServeHTTP 實作 http.Handler 介面。
func (h *WebhookHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		return
	}

	requestID := uuid.NewString()
	w.Header().Set("X-Request-ID", requestID)
	ctx := zlogger.WithRequestID(r.Context(), requestID)
	ctx = zlogger.WithComponent(ctx, "webhook")
	ctx = zlogger.WithOperation(ctx, "mutate")

	// 限制 request body 大小，防止記憶體耗盡
	r.Body = http.MaxBytesReader(w, r.Body, maxRequestBodyBytes)

	var review admissionv1.AdmissionReview
	if err := json.NewDecoder(r.Body).Decode(&review); err != nil {
		zlogger.WarnContext(ctx, "decode AdmissionReview failed", zlogger.Err(err))
		http.Error(w, "bad request", http.StatusBadRequest)
		return
	}

	patch, err := h.mutator.Execute(ctx, review.Request)
	if err != nil {
		zlogger.ErrorContext(ctx, "mutate usecase failed",
			zlogger.String("uid", string(review.Request.UID)),
			zlogger.Err(err),
		)
		metrics.MutateRequestsTotal.WithLabelValues("error").Inc()
		resp := &admissionv1.AdmissionResponse{
			UID:     review.Request.UID,
			Allowed: false,
			Result: &metav1.Status{
				Message: err.Error(),
			},
		}
		writeAdmissionReview(w, review.TypeMeta, resp)
		return
	}

	metrics.MutateRequestsTotal.WithLabelValues("success").Inc()
	writeAdmissionReview(w, review.TypeMeta, buildAdmissionResponse(review.Request.UID, patch))
}

// buildAdmissionResponse 建立帶有 JSON Patch 的 AdmissionResponse。
func buildAdmissionResponse(uid types.UID, patch []byte) *admissionv1.AdmissionResponse {
	pt := admissionv1.PatchTypeJSONPatch
	resp := &admissionv1.AdmissionResponse{
		UID:     uid,
		Allowed: true,
	}
	if len(patch) > 0 && string(patch) != "null" && string(patch) != "[]" {
		resp.Patch = patch
		resp.PatchType = &pt
	}
	return resp
}

func writeAdmissionReview(w http.ResponseWriter, typeMeta metav1.TypeMeta, resp *admissionv1.AdmissionResponse) {
	review := admissionv1.AdmissionReview{
		TypeMeta: typeMeta,
		Response: resp,
	}
	w.Header().Set("Content-Type", "application/json")
	_ = json.NewEncoder(w).Encode(review)
}
