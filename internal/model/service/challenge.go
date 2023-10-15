package service

type ChallengeConfig struct {
	Id        string
	Instances []InstanceConfig
}

type InstanceConfig struct {
	Image     string
	Resources ResourcesConfig
	Envs      []EnvConfig
	Records   []RecordConfig
}

type ResourcesConfig struct {
	Requests ResourceConfig
	Limit    ResourceConfig
}

type ResourceConfig struct {
	Memory string
	CPU    string
}

type EnvConfig struct {
	Name  string
	Value string
}

type RecordConfig struct {
	Type string
	Name string
	Data string
}
