package eks

import (
	"encoding/base64"
	"fmt"

	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

// K8sClient wraps a kubernetes.Clientset with token refresh support.
type K8sClient struct {
	Clientset *kubernetes.Clientset
	Token     *TokenProvider
	Config    *rest.Config
}

// NewK8sClient creates a K8s client from EKS cluster details.
func NewK8sClient(endpoint, caData string, tokenProvider *TokenProvider) (*K8sClient, error) {
	ca, err := base64.StdEncoding.DecodeString(caData)
	if err != nil {
		return nil, fmt.Errorf("decode CA: %w", err)
	}

	config := &rest.Config{
		Host: endpoint,
		TLSClientConfig: rest.TLSClientConfig{
			CAData: ca,
		},
		WrapTransport: tokenProvider.WrapTransport,
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, fmt.Errorf("create K8s client: %w", err)
	}

	return &K8sClient{
		Clientset: clientset,
		Token:     tokenProvider,
		Config:    config,
	}, nil
}
