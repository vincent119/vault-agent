package application

import (
	"context"
	"testing"
	"time"

	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"

	"vault-agent/internal/syncer/domain"
)

// syncMockFetcher 是 domain.SecretFetcher 的測試替身（避免與 mutate test 的 mockFetcher 衝突）。
type syncMockFetcher struct {
	data map[string]string
	err  error
}

func (s *syncMockFetcher) FetchSecret(_ context.Context, _ string, _ []string) (map[string]string, error) {
	return s.data, s.err
}

// mockK8sRepo 是 K8sSecretRepo 的測試替身，記錄所有 UpdateSecret 呼叫。
type mockK8sRepo struct {
	secrets   []corev1.Secret
	listErr   error
	updated   []*corev1.Secret
	updateErr error
}

func (m *mockK8sRepo) ListSecretsByLabel(_ context.Context, _, _ string) ([]corev1.Secret, error) {
	return m.secrets, m.listErr
}

func (m *mockK8sRepo) UpdateSecret(_ context.Context, s *corev1.Secret) error {
	m.updated = append(m.updated, s.DeepCopy())
	return m.updateErr
}

// newSyncWorker 建立一個用於測試的 SyncWorkerUseCase（interval=1s, nop logger）。
func newSyncWorker(repo K8sSecretRepo, fetchers map[string]domain.SecretFetcher) *SyncWorkerUseCase {
	return NewSyncWorkerUseCase(fetchers, repo, "", time.Second)
}

// ---- syncOnce tests ----

func TestSyncOnce_UpdatesWhenValueDiffers(t *testing.T) {
	t.Parallel()
	repo := &mockK8sRepo{
		secrets: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Name:      "myapp",
					Namespace: "default",
					Annotations: map[string]string{
						domain.AnnotationInject:  "true",
						domain.AnnotationPath:    "myapp/config",
						domain.AnnotationBackend: "vault",
					},
				},
				Data: map[string][]byte{"DB_PASS": []byte("old")},
			},
		},
	}
	w := newSyncWorker(repo, map[string]domain.SecretFetcher{
		"vault": &syncMockFetcher{data: map[string]string{"DB_PASS": "new"}},
	})

	w.syncOnce(context.Background())

	if len(repo.updated) != 1 {
		t.Fatalf("expected 1 UpdateSecret call, got %d", len(repo.updated))
	}
	if got := string(repo.updated[0].Data["DB_PASS"]); got != "new" {
		t.Errorf("DB_PASS = %q, want %q", got, "new")
	}
}

func TestSyncOnce_SkipsWhenValuesIdentical(t *testing.T) {
	t.Parallel()
	repo := &mockK8sRepo{
		secrets: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						domain.AnnotationInject: "true",
						domain.AnnotationPath:  "myapp/config",
					},
				},
				Data: map[string][]byte{"DB_PASS": []byte("same")},
			},
		},
	}
	w := newSyncWorker(repo, map[string]domain.SecretFetcher{
		"vault": &syncMockFetcher{data: map[string]string{"DB_PASS": "same"}},
	})

	w.syncOnce(context.Background())

	if len(repo.updated) != 0 {
		t.Errorf("expected 0 UpdateSecret calls, got %d", len(repo.updated))
	}
}

func TestSyncOnce_SkipsSecretWithoutPath(t *testing.T) {
	t.Parallel()
	repo := &mockK8sRepo{
		secrets: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						domain.AnnotationInject: "true",
						// path 未設定 → ParseSecretRef 回傳 nil
					},
				},
			},
		},
	}
	w := newSyncWorker(repo, map[string]domain.SecretFetcher{
		"vault": &syncMockFetcher{data: map[string]string{"KEY": "val"}},
	})

	w.syncOnce(context.Background()) // 不應 panic 或 update

	if len(repo.updated) != 0 {
		t.Errorf("expected 0 updates, got %d", len(repo.updated))
	}
}

func TestSyncOnce_UnknownBackendIsSkipped(t *testing.T) {
	t.Parallel()
	repo := &mockK8sRepo{
		secrets: []corev1.Secret{
			{
				ObjectMeta: metav1.ObjectMeta{
					Annotations: map[string]string{
						domain.AnnotationInject:  "true",
						domain.AnnotationPath:    "myapp/config",
						domain.AnnotationBackend: "nonexistent",
					},
				},
			},
		},
	}
	w := newSyncWorker(repo, map[string]domain.SecretFetcher{
		"vault": &syncMockFetcher{data: map[string]string{"KEY": "val"}},
	})

	w.syncOnce(context.Background())

	if len(repo.updated) != 0 {
		t.Errorf("expected 0 updates for unknown backend, got %d", len(repo.updated))
	}
}

// ---- Run 優雅關機測試 ----

func TestRun_StopsOnContextCancellation(t *testing.T) {
	t.Parallel()
	repo := &mockK8sRepo{}
	w := NewSyncWorkerUseCase(
		map[string]domain.SecretFetcher{"vault": &syncMockFetcher{}},
		repo,
		"",
		100*time.Millisecond, // 短間隔讓測試快速執行
	)

	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		w.Run(ctx)
		close(done)
	}()

	cancel()

	select {
	case <-done:
		// 正常退出
	case <-time.After(2 * time.Second):
		t.Error("Run() did not stop within 2s after context cancellation")
	}
}
