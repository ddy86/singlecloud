package types

import (
	"encoding/json"

	"github.com/zdnscloud/gorest/resource"
)

const (
	AppStatusCreate  = "create"
	AppStatusDelete  = "delete"
	AppStatusFailed  = "failed"
	AppStatusSucceed = "succeed"
)

var (
	ResourceTypeDeployment  = resource.DefaultKindName(Deployment{})
	ResourceTypeDaemonSet   = resource.DefaultKindName(DaemonSet{})
	ResourceTypeStatefulSet = resource.DefaultKindName(StatefulSet{})
	ResourceTypeJob         = resource.DefaultKindName(Job{})
	ResourceTypeCronJob     = resource.DefaultKindName(CronJob{})
	ResourceTypeConfigMap   = resource.DefaultKindName(ConfigMap{})
	ResourceTypeSecret      = resource.DefaultKindName(Secret{})
	ResourceTypeService     = resource.DefaultKindName(Service{})
	ResourceTypeIngress     = resource.DefaultKindName(Ingress{})
)

type Application struct {
	resource.ResourceBase `json:",inline"`
	Name                  string          `json:"name"`
	ChartName             string          `json:"chartName"`
	ChartVersion          string          `json:"chartVersion"`
	ChartIcon             string          `json:"chartIcon"`
	Status                string          `json:"status"`
	WorkloadCount         int             `json:"workloadCount,omitempty"`
	ReadyWorkloadCount    int             `json:"readyWorkloadCount,omitempty"`
	AppResources          AppResources    `json:"appResources,omitempty"`
	Configs               json.RawMessage `json:"configs,omitempty"`
	Manifests             []Manifest      `json:"manifests,omitempty"`
	SystemChart           bool            `json:"systemChart,omitempty"`
}

type AppResource struct {
	Name          string `json:"name"`
	Type          string `json:"type"`
	Link          string `json:"link"`
	Replicas      int    `json:"replicas,omitempty"`
	ReadyReplicas int    `json:"readyReplicas,omitempty"`
}

type Manifest struct {
	File      string `json:"file,omitempty"`
	Content   string `json:"content,omitempty"`
	Duplicate bool   `json:"duplicate,omitempty"`
}

func (a Application) GetParents() []resource.ResourceKind {
	return []resource.ResourceKind{Namespace{}}
}

type Applications []*Application

func (a Applications) Len() int {
	return len(a)
}

func (a Applications) Swap(i, j int) {
	a[i], a[j] = a[j], a[i]
}

func (a Applications) Less(i, j int) bool {
	if a[i].ChartName == a[j].ChartName {
		if a[i].ChartVersion == a[j].ChartVersion {
			return a[i].Name < a[j].Name
		} else {
			return a[i].ChartVersion < a[j].ChartVersion
		}
	} else {
		return a[i].ChartName < a[j].ChartName
	}
}

type AppResources []AppResource

func (r AppResources) Len() int {
	return len(r)
}

func (r AppResources) Swap(i, j int) {
	r[i], r[j] = r[j], r[i]
}

func (r AppResources) Less(i, j int) bool {
	if r[i].Type == r[j].Type {
		return r[i].Name < r[j].Name
	} else {
		return r[i].Type < r[j].Type
	}
}
