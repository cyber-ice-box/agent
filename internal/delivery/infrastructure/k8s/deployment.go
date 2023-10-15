package k8s

import (
	"context"
	"fmt"
	"gitlab.com/cyber-ice-box/agent/internal/model/delivery"
	coreV1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metaV1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	v1 "k8s.io/client-go/applyconfigurations/apps/v1"
	v14 "k8s.io/client-go/applyconfigurations/autoscaling/v1"
	v13 "k8s.io/client-go/applyconfigurations/core/v1"
	v12 "k8s.io/client-go/applyconfigurations/meta/v1"
	"strings"
)

func (k *K8s) ApplyDeployment(ctx context.Context, config delivery.ApplyDeploymentConfig) error {
	container := v13.Container()

	volumeMounts := make([]*v13.VolumeMountApplyConfiguration, 0)
	volumes := make([]*v13.VolumeApplyConfiguration, 0)
	for _, v := range config.Volumes {
		if v.HostPath != "" {
			volumes = append(volumes, v13.Volume().WithName(v.Name).
				WithHostPath(v13.HostPathVolumeSource().WithType(coreV1.HostPathDirectory).WithPath(v.HostPath)))
			continue
		}
		if v.ConfigMapName != "" {
			volumes = append(volumes, v13.Volume().WithName(v.Name).
				WithConfigMap(v13.ConfigMapVolumeSource().WithName(v.ConfigMapName)))
		}

		for _, vm := range v.Mounts {
			nVM := v13.VolumeMount().
				WithName(v.Name).
				WithMountPath(vm.MountPath)
			if vm.SubPath != "" {
				nVM = nVM.WithSubPath(vm.SubPath)
			}
			volumeMounts = append(volumeMounts, nVM)
		}
	}
	if len(volumeMounts) == 0 {
		volumes = nil
		volumeMounts = nil
	}

	envVars := make([]*v13.EnvVarApplyConfiguration, 0)
	for _, env := range config.Envs {
		envVars = append(envVars, v13.EnvVar().WithName(env.Name).WithValue(env.Value))
	}
	if len(envVars) > 0 {
		container = container.WithEnv(envVars...)
	}

	if len(config.Args) > 0 {
		container = container.WithArgs(config.Args...)
	}

	ports := make([]*v13.ContainerPortApplyConfiguration, 0)
	for _, port := range config.Ports {
		ports = append(ports, v13.ContainerPort().WithProtocol(k.parseProtocol(port.Protocol)).WithContainerPort(port.Port))
	}
	if len(ports) > 0 {
		container = container.WithPorts(ports...)
	}

	if (config.Resources.Limit.CPU != "" && config.Resources.Limit.Memory != "") || (config.Resources.Requests.CPU != "" && config.Resources.Requests.Memory != "") {
		r := v13.ResourceRequirements()
		if config.Resources.Limit.CPU != "" {
			cpu, err := resource.ParseQuantity(config.Resources.Limit.CPU)
			if err != nil {
				return err
			}
			memory, err := resource.ParseQuantity(config.Resources.Limit.Memory)
			if err != nil {
				return err
			}
			r.WithLimits(coreV1.ResourceList{
				coreV1.ResourceCPU:    cpu,
				coreV1.ResourceMemory: memory,
			})
		}
		if config.Resources.Requests.CPU != "" {
			cpu, err := resource.ParseQuantity(config.Resources.Requests.CPU)
			if err != nil {
				return err
			}
			memory, err := resource.ParseQuantity(config.Resources.Requests.Memory)
			if err != nil {
				return err
			}
			r.WithRequests(coreV1.ResourceList{
				coreV1.ResourceCPU:    cpu,
				coreV1.ResourceMemory: memory,
			})
		}
		container = container.WithResources(r)
	}

	annotations := make(map[string]string)
	if config.Ip != "" {
		annotations["cni.projectcalico.org/ipAddrs"] = fmt.Sprintf("[\"%s\"]", strings.Split(config.Ip, "/")[0])
	}

	capAdds := make([]coreV1.Capability, 0)
	for _, cd := range config.CapAdds {
		capAdds = append(capAdds, coreV1.Capability(cd))
	}
	if len(capAdds) == 0 {
		capAdds = nil
	}

	if config.ReplicaCount == 0 {
		config.ReplicaCount = 1
	}

	if config.ReadinessProbe != nil {
		container = container.WithReadinessProbe(v13.Probe().
			WithPeriodSeconds(config.ReadinessProbe.PeriodSeconds).
			WithExec(v13.ExecAction().WithCommand(config.ReadinessProbe.Cmd...)))
	}

	_, err := k.kubeClient.AppsV1().Deployments(config.Namespace).Apply(
		ctx,
		v1.Deployment(config.Name, config.Namespace).WithLabels(map[string]string{NameLabel: config.Name, platformLabel: taskLabel, config.Label.Key: config.Label.Value}).
			WithSpec(v1.DeploymentSpec().
				WithSelector(v12.LabelSelector().WithMatchLabels(map[string]string{NameLabel: config.Name})).
				WithReplicas(config.ReplicaCount).
				WithTemplate(v13.PodTemplateSpec().
					WithName(config.Name).
					WithNamespace(config.Namespace).
					WithLabels(map[string]string{NameLabel: config.Name, platformLabel: taskLabel}).
					WithAnnotations(annotations).
					WithSpec(v13.PodSpec().
						WithVolumes(volumes...).
						WithContainers(container.
							WithName(config.Name).
							WithImage(config.Image).
							WithSecurityContext(v13.SecurityContext().
								WithPrivileged(config.Privileged).
								WithAllowPrivilegeEscalation(config.Privileged).
								WithCapabilities(v13.Capabilities().
									WithAdd(capAdds...))).
							WithVolumeMounts(volumeMounts...))))),
		metaV1.ApplyOptions{FieldManager: "application/apply-patch"})
	return err
}

func (k *K8s) GetDeploymentsInNamespaceBySelector(ctx context.Context, namespace string, selector ...string) ([]delivery.DeploymentStatus, error) {
	labelSelector := fmt.Sprintf("%s=%s", platformLabel, labLabel)

	if len(selector) > 0 {
		labelSelector = selector[0]
	}

	dps, err := k.kubeClient.AppsV1().Deployments(namespace).List(ctx, metaV1.ListOptions{
		LabelSelector: labelSelector,
	})
	if err != nil {
		return nil, err
	}

	dpsStatus := make([]delivery.DeploymentStatus, len(dps.Items))

	for _, dp := range dps.Items {
		dpsStatus = append(dpsStatus, delivery.DeploymentStatus{
			Name:          dp.GetName(),
			IP:            dp.Spec.Template.GetAnnotations()["cni.projectcalico.org/ipAddrs"],
			AllReplicas:   dp.Status.Replicas,
			ReadyReplicas: dp.Status.ReadyReplicas,
		})
	}

	return dpsStatus, nil
}

func (k *K8s) DeploymentExists(ctx context.Context, name, namespace string) (bool, error) {
	dp, err := k.kubeClient.AppsV1().Deployments(namespace).Get(ctx, name, metaV1.GetOptions{})
	if err != nil {
		return false, err
	}
	return dp.GetName() == name && dp.GetNamespace() == namespace, nil
}

func (k *K8s) ResetDeployment(ctx context.Context, name, namespace string) error {
	pods, err := k.kubeClient.CoreV1().Pods(namespace).List(ctx, metaV1.ListOptions{LabelSelector: fmt.Sprintf("%s=%s", NameLabel, name)})
	if err != nil {
		return err
	}
	if err = k.kubeClient.CoreV1().Pods(namespace).Delete(ctx, pods.Items[0].Name, metaV1.DeleteOptions{}); err != nil {
		return err
	}
	return nil
}

func (k *K8s) ScaleDeployment(ctx context.Context, name, namespace string, scale int32) error {
	_, err := k.kubeClient.AppsV1().Deployments(namespace).ApplyScale(ctx, name, &v14.ScaleApplyConfiguration{
		Spec: &v14.ScaleSpecApplyConfiguration{Replicas: &scale},
	}, metaV1.ApplyOptions{})
	return err
}

func (k *K8s) DeleteDeployment(ctx context.Context, name, namespace string) error {
	return k.kubeClient.AppsV1().Deployments(namespace).Delete(ctx, name, metaV1.DeleteOptions{})
}
