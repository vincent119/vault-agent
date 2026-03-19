package infra

import (
	"context"
	"fmt"

	"github.com/vincent119/zlogger"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
)

// K8sRepository 提供對 Kubernetes Secret 資源的讀寫封裝。
type K8sRepository struct {
	client kubernetes.Interface
}

// NewK8sRepository 建立 K8sRepository。
// kubeconfig 為空時僅嘗試 In-Cluster Config（Pod 內）；非空時使用該檔案路徑，不預設讀取本機 kubeconfig。
func NewK8sRepository(kubeconfig string) (*K8sRepository, error) {
	var cfg *rest.Config
	var err error

	if kubeconfig != "" {
		zlogger.Info("k8s config source: kubeconfig", zlogger.String("k8s.kubeconfig", kubeconfig))
		cfg, err = clientcmd.BuildConfigFromFlags("", kubeconfig)
	} else {
		zlogger.Info("k8s config source: in-cluster")
		cfg, err = rest.InClusterConfig()
	}
	if err != nil {
		zlogger.Warn("k8s config load failed", zlogger.Bool("k8s.kubeconfig.provided", kubeconfig != ""), zlogger.Err(err))
		return nil, fmt.Errorf("build k8s config: %w", err)
	}

	clientset, err := kubernetes.NewForConfig(cfg)
	if err != nil {
		zlogger.Error("k8s clientset create failed", zlogger.Err(err))
		return nil, fmt.Errorf("new k8s clientset: %w", err)
	}

	zlogger.Info("k8s client initialized")
	return &K8sRepository{client: clientset}, nil
}

// GetSecret 取得指定命名空間中的 Kubernetes Secret。
func (r *K8sRepository) GetSecret(ctx context.Context, namespace, name string) (*corev1.Secret, error) {
	secret, err := r.client.CoreV1().Secrets(namespace).Get(ctx, name, metav1.GetOptions{})
	if err != nil {
		zlogger.Warn("k8s get secret failed", zlogger.String("k8s.namespace", namespace), zlogger.String("k8s.secret", name), zlogger.Err(err))
		return nil, fmt.Errorf("get secret %s/%s: %w", namespace, name, err)
	}
	return secret, nil
}

// UpdateSecret 更新 Kubernetes Secret 的資料（整個 Data 欄位）。
func (r *K8sRepository) UpdateSecret(ctx context.Context, secret *corev1.Secret) error {
	_, err := r.client.CoreV1().Secrets(secret.Namespace).Update(ctx, secret, metav1.UpdateOptions{})
	if err != nil {
		zlogger.Warn("k8s update secret failed", zlogger.String("k8s.namespace", secret.Namespace), zlogger.String("k8s.secret", secret.Name), zlogger.Err(err))
		return fmt.Errorf("update secret %s/%s: %w", secret.Namespace, secret.Name, err)
	}
	return nil
}

// ListSecretsByLabel 列出指定命名空間中，符合標籤選擇器的 Secret 列表。
func (r *K8sRepository) ListSecretsByLabel(ctx context.Context, namespace, labelSelector string) ([]corev1.Secret, error) {
	list, err := r.client.CoreV1().Secrets(namespace).List(ctx, metav1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		zlogger.Warn("k8s list secrets failed", zlogger.String("k8s.namespace", namespace), zlogger.String("k8s.label_selector", labelSelector), zlogger.Err(err))
		return nil, fmt.Errorf("list secrets namespace=%s labels=%s: %w", namespace, labelSelector, err)
	}
	return list.Items, nil
}
