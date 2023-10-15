package k8s

import (
	"context"
	"gitlab.com/cyber-ice-box/agent/internal/model/delivery"
	coreV1 "k8s.io/api/core/v1"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/util/intstr"
	v1 "k8s.io/client-go/applyconfigurations/core/v1"
)

func (k *K8s) parseProtocol(protocol string) coreV1.Protocol {
	switch protocol {
	case "UDP":
		return coreV1.ProtocolUDP
	case "SCTP":
		return coreV1.ProtocolSCTP
	default:
		return coreV1.ProtocolTCP
	}
}

func (k *K8s) ApplyService(ctx context.Context, config delivery.ApplyServiceConfig) error {
	var serviceType coreV1.ServiceType

	ports := make([]*v1.ServicePortApplyConfiguration, 0)

	switch config.ServiceType {
	case "NodePort":
		serviceType = coreV1.ServiceTypeNodePort
	case "LoadBalancer":
		serviceType = coreV1.ServiceTypeLoadBalancer
	case "ExternalName":
		serviceType = coreV1.ServiceTypeExternalName
	default:
		serviceType = coreV1.ServiceTypeClusterIP
	}

	for _, port := range config.Ports {
		protocol := k.parseProtocol(port.Protocol)
		ports = append(ports, &v1.ServicePortApplyConfiguration{
			Protocol:   &protocol,
			Port:       &port.Port,
			TargetPort: &intstr.IntOrString{IntVal: port.TargetPort},
			NodePort:   &port.NodePort,
		})
	}

	_, err := k.kubeClient.CoreV1().Services(config.Namespace).Apply(
		ctx,
		v1.Service(config.Name, config.Namespace).
			WithLabels(map[string]string{platformLabel: serviceLabel}).
			WithSpec(v1.ServiceSpec().
				WithSelector(map[string]string{NameLabel: config.Name}).
				WithType(serviceType).
				WithPorts(ports...)),
		metaV1.ApplyOptions{FieldManager: "application/apply-patch"},
	)
	return err
}

func (k *K8s) DeleteService(ctx context.Context, name, namespace string) error {
	return k.kubeClient.CoreV1().Services(namespace).Delete(ctx, name, metaV1.DeleteOptions{})
}
