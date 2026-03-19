package application

import (
	"context"
	"encoding/json"
	"errors"
	"testing"

	admissionv1 "k8s.io/api/admission/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	"vault-agent/internal/syncer/domain"
)

// mockFetcher 是 domain.SecretFetcher 的測試替身。
type mockFetcher struct {
	data map[string]string
	err  error
}

func (m *mockFetcher) FetchSecret(_ context.Context, _ string, _ []string) (map[string]string, error) {
	return m.data, m.err
}

// podAdmissionRequest 將 corev1.Pod 序列化為 AdmissionRequest。
func podAdmissionRequest(t *testing.T, pod *corev1.Pod) *admissionv1.AdmissionRequest {
	t.Helper()
	b, err := json.Marshal(pod)
	if err != nil {
		t.Fatal(err)
	}
	return &admissionv1.AdmissionRequest{Object: runtime.RawExtension{Raw: b}}
}

// ---- buildEnvPatches tests ----

func TestBuildEnvPatches_EmptyData(t *testing.T) {
	t.Parallel()
	patches, err := buildEnvPatches([]corev1.Container{{Name: "c"}}, nil)
	if err != nil {
		t.Fatal(err)
	}
	if len(patches) != 0 {
		t.Errorf("expected 0 patches, got %d", len(patches))
	}
}

func TestBuildEnvPatches_NilEnvAddsInitPatch(t *testing.T) {
	t.Parallel()
	containers := []corev1.Container{{Name: "c", Env: nil}}
	data := map[string]string{"KEY": "val"}
	patches, err := buildEnvPatches(containers, data)
	if err != nil {
		t.Fatal(err)
	}
	// 期待 2 個 patch: 1 個 env init + 1 個 env add
	if len(patches) != 2 {
		t.Fatalf("len(patches) = %d, want 2", len(patches))
	}
	if patches[0].Op != "add" || patches[0].Path != "/spec/containers/0/env" {
		t.Errorf("patch[0] = %+v, want env init patch", patches[0])
	}
}

func TestBuildEnvPatches_ExistingEnvSkipsInit(t *testing.T) {
	t.Parallel()
	containers := []corev1.Container{{Name: "c", Env: []corev1.EnvVar{{Name: "EXISTING"}}}}
	data := map[string]string{"KEY": "val"}
	patches, err := buildEnvPatches(containers, data)
	if err != nil {
		t.Fatal(err)
	}
	// 僅 1 個 env add，無 init patch
	if len(patches) != 1 {
		t.Fatalf("len(patches) = %d, want 1", len(patches))
	}
	if patches[0].Path != "/spec/containers/0/env/-" {
		t.Errorf("patch[0].Path = %q, want /spec/containers/0/env/-", patches[0].Path)
	}
}

func TestBuildEnvPatches_MultiContainer(t *testing.T) {
	t.Parallel()
	containers := []corev1.Container{
		{Name: "c1", Env: nil},
		{Name: "c2", Env: nil},
	}
	data := map[string]string{"KEY": "val"}
	patches, err := buildEnvPatches(containers, data)
	if err != nil {
		t.Fatal(err)
	}
	// 2 containers × (1 init + 1 add) = 4
	if len(patches) != 4 {
		t.Fatalf("len(patches) = %d, want 4", len(patches))
	}
}

// ---- MutateUseCase.Execute tests ----

func TestExecute_NoInjectAnnotation(t *testing.T) {
	t.Parallel()
	uc := NewMutateUseCase(map[string]domain.SecretFetcher{})
	pod := &corev1.Pod{}
	patch, err := uc.Execute(context.Background(), podAdmissionRequest(t, pod))
	if err != nil {
		t.Fatal(err)
	}
	if string(patch) != "[]" {
		t.Errorf("patch = %s, want []", patch)
	}
}

func TestExecute_InjectFalse(t *testing.T) {
	t.Parallel()
	uc := NewMutateUseCase(map[string]domain.SecretFetcher{})
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{domain.AnnotationInject: "false"},
		},
	}
	patch, err := uc.Execute(context.Background(), podAdmissionRequest(t, pod))
	if err != nil {
		t.Fatal(err)
	}
	if string(patch) != "[]" {
		t.Errorf("patch = %s, want []", patch)
	}
}

func TestExecute_UnknownBackendReturnsError(t *testing.T) {
	t.Parallel()
	uc := NewMutateUseCase(map[string]domain.SecretFetcher{})
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				domain.AnnotationInject:  "true",
				domain.AnnotationPath:   "myapp/config",
				domain.AnnotationBackend: "unknown",
			},
		},
	}
	_, err := uc.Execute(context.Background(), podAdmissionRequest(t, pod))
	if err == nil {
		t.Fatal("expected error for unknown backend, got nil")
	}
}

func TestExecute_FetchErrorPropagates(t *testing.T) {
	t.Parallel()
	fetchErr := errors.New("vault unavailable")
	uc := NewMutateUseCase(map[string]domain.SecretFetcher{
		"vault": &mockFetcher{err: fetchErr},
	})
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				domain.AnnotationInject: "true",
				domain.AnnotationPath:  "myapp/config",
			},
		},
	}
	_, err := uc.Execute(context.Background(), podAdmissionRequest(t, pod))
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if !errors.Is(err, fetchErr) {
		t.Errorf("expected wrapped fetchErr, got: %v", err)
	}
}

func TestExecute_SingleContainerNilEnv(t *testing.T) {
	t.Parallel()
	uc := NewMutateUseCase(map[string]domain.SecretFetcher{
		"vault": &mockFetcher{data: map[string]string{"DB_PASS": "s3cr3t"}},
	})
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				domain.AnnotationInject: "true",
				domain.AnnotationPath:  "myapp/config",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{{Name: "app"}},
		},
	}
	patchBytes, err := uc.Execute(context.Background(), podAdmissionRequest(t, pod))
	if err != nil {
		t.Fatal(err)
	}
	var ops []jsonPatch
	if err := json.Unmarshal(patchBytes, &ops); err != nil {
		t.Fatalf("invalid patch JSON: %v, raw: %s", err, patchBytes)
	}
	// 期待 env init patch + env add patch
	if len(ops) < 2 {
		t.Errorf("expected ≥2 patches, got %d: %s", len(ops), patchBytes)
	}
}

func TestExecute_MultiContainer(t *testing.T) {
	t.Parallel()
	uc := NewMutateUseCase(map[string]domain.SecretFetcher{
		"vault": &mockFetcher{data: map[string]string{"KEY": "val"}},
	})
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Annotations: map[string]string{
				domain.AnnotationInject: "true",
				domain.AnnotationPath:  "myapp/config",
			},
		},
		Spec: corev1.PodSpec{
			Containers: []corev1.Container{
				{Name: "app"},
				{Name: "sidecar"},
			},
		},
	}
	patchBytes, err := uc.Execute(context.Background(), podAdmissionRequest(t, pod))
	if err != nil {
		t.Fatal(err)
	}
	var ops []jsonPatch
	if err := json.Unmarshal(patchBytes, &ops); err != nil {
		t.Fatalf("invalid patch JSON: %v", err)
	}
	// 2 containers × (1 init + 1 add) = 4 patches
	if len(ops) != 4 {
		t.Errorf("expected 4 patches for 2 containers, got %d: %s", len(ops), patchBytes)
	}
}
