package k8s

import (
	"context"
	"fmt"
	"gitlab.com/cyber-ice-box/agent/internal/model/delivery"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
)

func (k *K8s) GetPodsInNamespace(ctx context.Context, namespace, name string) ([]delivery.Pod, error) {
	pods, err := k.kubeClient.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", NameLabel, name),
	})
	if err != nil {
		return nil, err
	}
	containers := make([]delivery.Pod, 0)
	for _, pod := range pods.Items {
		containers = append(containers, delivery.Pod{
			Name:        pod.Name,
			IP:          pod.Status.PodIP,
			StartTime:   pod.CreationTimestamp.String(),
			StatusReady: pod.Status.ContainerStatuses[0].Ready,
		})
	}

	return containers, nil
}

func (k *K8s) WatchPods(namespace string, addItem, deleteItem delivery.PodFn, timeout int64) {

	watchFunc := func(options metaV1.ListOptions) (watch.Interface, error) {
		return k.kubeClient.CoreV1().Pods(namespace).Watch(context.Background(), metaV1.ListOptions{TimeoutSeconds: &timeout})
	}

	watcher, _ := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})

	for event := range watcher.ResultChan() {
		item := event.Object.(*coreV1.Pod)

		switch event.Type {
		case watch.Deleted:
			deleteItem(item)
		case watch.Added:
			if len(item.Status.ContainerStatuses) > 0 && len(item.Status.PodIP) > 0 {
				addItem(item)
			}
		}
	}
}
