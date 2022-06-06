package core

import "k8s.io/apimachinery/pkg/api/resource"

type RunnerConfig struct {
	Name string
	Namespace string
	ResourceClass string
	Image string
	CPU resource.Quantity
	Memory resource.Quantity
}

type AgentConfig struct {
	Runners []string
}