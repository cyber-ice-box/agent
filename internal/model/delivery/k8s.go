package delivery

import (
	"gitlab.com/cyber-ice-box/agent/internal/model/service"
	coreV1 "k8s.io/api/core/v1"
)

type PodFn func(*coreV1.Pod)

type NamespaceFn func(*coreV1.Namespace)

type ApplyDeploymentConfig struct {
	Name           string
	Namespace      string
	Label          Label
	ReplicaCount   int32
	Image          string
	Ip             string
	Resources      service.ResourcesConfig
	Envs           []service.EnvConfig
	Args           []string
	Ports          []Port
	Volumes        []Volume
	Privileged     bool
	CapAdds        []string
	ReadinessProbe *Probe
}

type Label struct {
	Key   string
	Value string
}

type Volume struct {
	Name          string
	ConfigMapName string
	HostPath      string
	Mounts        []Mount
}

type Mount struct {
	MountPath string
	SubPath   string
}

type Port struct {
	Protocol string
	Port     int32
}

type Probe struct {
	Cmd           []string
	PeriodSeconds int32
}

type ApplyServiceConfig struct {
	Name        string
	Namespace   string
	ServiceType string
	Ports       []ServicePort
}

type ServicePort struct {
	Protocol   string
	Port       int32
	TargetPort int32
	NodePort   int32
}

type DeploymentStatus struct {
	Name          string
	IP            string
	AllReplicas   int32
	ReadyReplicas int32
}

type Pod struct {
	Name        string
	IP          string
	StartTime   string
	StatusReady bool
}
