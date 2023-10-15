package k8s

import (
	calico "github.com/projectcalico/api/pkg/client/clientset_generated/clientset"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

var (
	NameLabel      = "name"
	NamespaceLabel = "namespace"
	platformLabel  = "cybericebox"
	labLabel       = "lab"
	taskLabel      = "task"
	configLabel    = "config"
	serviceLabel   = "service"
)

type K8s struct {
	kubeClient   *kubernetes.Clientset
	calicoClient *calico.Clientset
}

func New() *K8s {
	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	k := &K8s{}
	k.kubeClient, err = kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	k.calicoClient, err = calico.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	return k
}
