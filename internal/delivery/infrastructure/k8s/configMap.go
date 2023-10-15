package k8s

import (
	"context"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func (k *K8s) ApplyConfigMap(ctx context.Context, name, namespace string, data map[string]string) error {
	_, err := k.kubeClient.CoreV1().ConfigMaps(namespace).Apply(ctx,
		v1.ConfigMap(name, namespace).WithLabels(map[string]string{platformLabel: configLabel}).WithData(data), metaV1.ApplyOptions{FieldManager: "application/apply-patch"})
	return err
}

func (k *K8s) GetConfigMapData(ctx context.Context, name, namespace string) (map[string]string, error) {
	get, err := k.kubeClient.CoreV1().ConfigMaps(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return nil, err
	}

	return get.Data, nil
}

func (k *K8s) DeleteConfigMap(ctx context.Context, name, namespace string) error {
	return k.kubeClient.CoreV1().ConfigMaps(namespace).Delete(ctx, name, metaV1.DeleteOptions{})
}
