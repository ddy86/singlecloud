package handler

import (
	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gorest/resource"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type OuterServiceManager struct {
	clusters *ClusterManager
}

func newOuterServiceManager(clusters *ClusterManager) *OuterServiceManager {
	return &OuterServiceManager{
		clusters: clusters,
	}
}

func (m *OuterServiceManager) List(ctx *resource.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	namespace := ctx.Resource.GetParent().GetID()
	if cluster == nil {
		return nil
	}

	resp, err := getOuterServices(cluster.Name, m.clusters.Agent, namespace)
	if err != nil {
		log.Warnf("get innerservices info failed:%s", err.Error())
		return nil
	}
	return resp
}

func getOuterServices(cluster string, agent *clusteragent.AgentManager, namespace string) ([]*types.OuterService, error) {
	url := "/apis/agent.zcloud.cn/v1/namespaces/" + namespace + "/outerservices"
	outerservices := make([]*types.OuterService, 0)
	res := make([]types.OuterService, 0)
	if err := agent.ListResource(cluster, url, &res); err != nil {
		return outerservices, err
	}
	for _, outerservice := range res {
		outerservices = append(outerservices, &outerservice)
	}
	return outerservices, nil
}