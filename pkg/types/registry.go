package types

import (
	resttypes "github.com/zdnscloud/gorest/resource"
)

func SetRegistrySchema(schema *resttypes.Schema, handler resttypes.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"POST", "GET"}
	schema.ResourceMethods = []string{"DELETE"}
}

type Registry struct {
	resttypes.Resource `json:",inline"`
	Cluster            string `json:"cluster"`
	IngressDomain      string `json:"ingressDomain"`
	StorageClass       string `json:"storageClass"`
	StorageSize        int    `json:"storageSize"`
	AdminPassword      string `json:"adminPassword"`
	RedirectUrl        string `json:"redirectUrl"`
}

var RegistryType = resttypes.GetResourceType(Registry{})
