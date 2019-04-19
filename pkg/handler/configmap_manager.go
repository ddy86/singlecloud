package handler

import (
	"context"
	"fmt"

	corev1 "k8s.io/api/core/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gorest/api"
	resttypes "github.com/zdnscloud/gorest/types"
	"github.com/zdnscloud/singlecloud/pkg/logger"
	"github.com/zdnscloud/singlecloud/pkg/types"
)

type ConfigMapManager struct {
	api.DefaultHandler
	clusters *ClusterManager
}

func newConfigMapManager(clusters *ClusterManager) *ConfigMapManager {
	return &ConfigMapManager{clusters: clusters}
}

func (m *ConfigMapManager) Create(ctx *resttypes.Context, yamlConf []byte) (interface{}, *resttypes.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil, resttypes.NewAPIError(resttypes.NotFound, "cluster s doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	cm := ctx.Object.(*types.ConfigMap)
	err := createConfigMap(cluster.KubeClient, namespace, cm)
	if err == nil {
		cm.SetID(cm.Name)
		return cm, nil
	}

	if apierrors.IsAlreadyExists(err) {
		return nil, resttypes.NewAPIError(resttypes.DuplicateResource, fmt.Sprintf("duplicate configmap name %s", cm.Name))
	} else {
		return nil, resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create configmap failed %s", err.Error()))
	}
}

func (m *ConfigMapManager) List(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	k8sConfigMaps, err := getConfigMaps(cluster.KubeClient, namespace)
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("list deployment info failed:%s", err.Error())
		}
		return nil
	}

	var cms []*types.ConfigMap
	for _, cm := range k8sConfigMaps.Items {
		cms = append(cms, k8sConfigMapToSCConfigMap(&cm))
	}
	return cms
}

func (m ConfigMapManager) Get(ctx *resttypes.Context) interface{} {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return nil
	}

	namespace := ctx.Object.GetParent().GetID()
	cm := ctx.Object.(*types.ConfigMap)
	k8sConfigMap, err := getConfigMap(cluster.KubeClient, namespace, cm.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) == false {
			logger.Warn("get deployment info failed:%s", err.Error())
		}
		return nil
	}

	return k8sConfigMapToSCConfigMap(k8sConfigMap)
}

func (m ConfigMapManager) Delete(ctx *resttypes.Context) *resttypes.APIError {
	cluster := m.clusters.GetClusterForSubResource(ctx.Object)
	if cluster == nil {
		return resttypes.NewAPIError(resttypes.NotFound, "cluster doesn't exist")
	}

	namespace := ctx.Object.GetParent().GetID()
	cm := ctx.Object.(*types.ConfigMap)
	err := deleteConfigMap(cluster.KubeClient, namespace, cm.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return resttypes.NewAPIError(resttypes.NotFound, fmt.Sprintf("configmap %s desn't exist", namespace))
		} else {
			return resttypes.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("delete configmap failed %s", err.Error()))
		}
	}
	return nil
}

func getConfigMap(cli client.Client, namespace, name string) (*corev1.ConfigMap, error) {
	cm := corev1.ConfigMap{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{namespace, name}, &cm)
	return &cm, err
}

func getConfigMaps(cli client.Client, namespace string) (*corev1.ConfigMapList, error) {
	cms := corev1.ConfigMapList{}
	err := cli.List(context.TODO(), &client.ListOptions{Namespace: namespace}, &cms)
	return &cms, err
}

func createConfigMap(cli client.Client, namespace string, cm *types.ConfigMap) error {
	return cli.Create(context.TODO(), scConfigMapToK8sConfigMap(cm, namespace))
}

func updateConfigMap(cli client.Client, namespace string, cm *types.ConfigMap) error {
	return cli.Update(context.TODO(), scConfigMapToK8sConfigMap(cm, namespace))
}

func scConfigMapToK8sConfigMap(cm *types.ConfigMap, namespace string) *corev1.ConfigMap {
	data := make(map[string]string)
	for _, c := range cm.Configs {
		data[c.Name] = c.Data
	}

	return &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: cm.Name, Namespace: namespace},
		Data:       data,
	}
}

func deleteConfigMap(cli client.Client, namespace, name string) error {
	deploy := &corev1.ConfigMap{
		ObjectMeta: metav1.ObjectMeta{Name: name, Namespace: namespace},
	}
	return cli.Delete(context.TODO(), deploy)
}

func k8sConfigMapToSCConfigMap(k8sConfigMap *corev1.ConfigMap) *types.ConfigMap {
	var configs []types.Config
	for n, d := range k8sConfigMap.Data {
		configs = append(configs, types.Config{
			Name: n,
			Data: d,
		})
	}
	cm := &types.ConfigMap{
		Name:    k8sConfigMap.Name,
		Configs: configs,
	}
	cm.SetID(k8sConfigMap.Name)
	cm.SetType(types.ConfigMapType)
	cm.SetCreationTimestamp(k8sConfigMap.CreationTimestamp.Time)
	return cm
}
