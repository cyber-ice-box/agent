package k8s

import (
	"context"
	"fmt"
	"gitlab.com/cyber-ice-box/agent/internal/model/delivery"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/watch"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
	"k8s.io/client-go/tools/cache"
	toolsWatch "k8s.io/client-go/tools/watch"
	"strings"
)

func (k *K8s) ApplyNamespace(ctx context.Context, name string, ipPoolName *string) error {
	annotations := make(map[string]string)
	if ipPoolName != nil {
		annotations["cni.projectcalico.org/ipv4pools"] = fmt.Sprintf("[\"%s\"]", *ipPoolName)
	}

	_, err := k.kubeClient.CoreV1().Namespaces().Apply(
		ctx,
		v1.Namespace(name).WithAnnotations(annotations).WithLabels(map[string]string{platformLabel: labLabel}),
		metaV1.ApplyOptions{FieldManager: "application/apply-patch"})
	return err
}

func (k *K8s) GetNamespaces(ctx context.Context) ([]string, error) {
	nss, err := k.kubeClient.CoreV1().Namespaces().List(ctx, metaV1.ListOptions{
		LabelSelector: fmt.Sprintf("%s=%s", platformLabel, labLabel),
	})
	if err != nil {
		return nil, err
	}

	nsNames := make([]string, len(nss.Items))

	for _, ns := range nss.Items {
		nsNames = append(nsNames, ns.GetName())
	}

	return nsNames, nil
}

func (k *K8s) NamespaceExists(ctx context.Context, name string) (bool, error) {
	ns, err := k.kubeClient.CoreV1().Namespaces().Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		if strings.Contains(err.Error(), "not found") {
			return false, nil
		} else {
			return false, err
		}
	}

	return ns.GetName() == name, nil
}

func (k *K8s) DeleteNamespace(ctx context.Context, name string) error {
	return k.kubeClient.CoreV1().Namespaces().Delete(ctx, name, metaV1.DeleteOptions{})
}

func (k *K8s) WatchNamespace(addItem, deleteItem delivery.NamespaceFn, timeout int64) {
	watchFunc := func(options metaV1.ListOptions) (watch.Interface, error) {
		return k.kubeClient.CoreV1().Namespaces().Watch(context.Background(), metaV1.ListOptions{TimeoutSeconds: &timeout})
	}

	watcher, _ := toolsWatch.NewRetryWatcher("1", &cache.ListWatch{WatchFunc: watchFunc})

	for event := range watcher.ResultChan() {
		item := event.Object.(*coreV1.Namespace)

		switch event.Type {
		case watch.Modified:
		case watch.Bookmark:
		case watch.Error:
		case watch.Deleted:
			deleteItem(item)
		case watch.Added:
			addItem(item)
		}
	}
}
