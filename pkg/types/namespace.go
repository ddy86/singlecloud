package types

import (
	"github.com/zdnscloud/gorest/resource"
)

func SetNamespaceSchema(schema *types.Schema, handler types.Handler) {
	schema.Handler = handler
	schema.CollectionMethods = []string{"GET", "POST"}
	schema.ResourceMethods = []string{"GET", "DELETE"}
	schema.Parents = []string{ClusterType}
}

type Namespace struct {
	types.Resource `json:",inline"`
	Name           string `json:"name,omitempty"`
}

var NamespaceType = types.GetResourceType(Namespace{})
