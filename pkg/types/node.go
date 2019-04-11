package types

import (
	resttypes "github.com/zdnscloud/gorest/types"
)

func SetNodeSchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET"}
	schema.ResourceMethods = []string{"GET"}
	schema.Parent = ClusterType
}

type Node struct {
	resttypes.Resource   `json:",inline"`
	Name                 string            `json:"name,omitempty"`
	Address              string            `json:"address,omitempty"`
	Role                 string            `json:"role,omitempty"`
	Labels               map[string]string `json:"labels,omitempty"`
	Annotations          map[string]string `json:"annotations,omitempty"`
	OperatingSystem      string            `json:"operatingSystem,omitempty"`
	OperatingSystemImage string            `json:"operatingSystemImage,omitempty"`
	DockerVersion        string            `json:"dockerVersion,omitempty"`
	Cpu                  string            `json:"cpu,omitempty"`
	CpuUsedRatio         string            `json:"cpuUsedRatio,omitempty"`
	Memory               string            `json:"memory,omitempty"`
	MemoryUsedRatio      string            `json:"memoryUsedRatio,omitempty"`
	Pod                  string            `json:"pod"`
	PodUsedRatio         string            `json:"podUsedRatio,omitempty"`
}

var NodeType = resttypes.GetResourceType(Node{})
