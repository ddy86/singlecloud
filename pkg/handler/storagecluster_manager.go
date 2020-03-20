package handler

import (
	"context"
	"errors"
	"fmt"
	"math"
	"sort"
	"strconv"
	"strings"

	k8sstorage "k8s.io/api/storage/v1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	k8stypes "k8s.io/apimachinery/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/cement/set"
	"github.com/zdnscloud/cement/slice"
	"github.com/zdnscloud/gok8s/client"
	resterr "github.com/zdnscloud/gorest/error"
	"github.com/zdnscloud/gorest/resource"
	storagev1 "github.com/zdnscloud/immense/pkg/apis/zcloud/v1"
	"github.com/zdnscloud/immense/pkg/common"
	"github.com/zdnscloud/singlecloud/pkg/clusteragent"
	"github.com/zdnscloud/singlecloud/pkg/types"
	"github.com/zdnscloud/singlecloud/pkg/zke"
)

type StorageClusterManager struct {
	clusters *ClusterManager
}

func newStorageClusterManager(clusters *ClusterManager) *StorageClusterManager {
	return &StorageClusterManager{
		clusters: clusters,
	}
}

func (m *StorageClusterManager) List(ctx *resource.Context) (interface{}, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	k8sStorageClusters, err := getStorageClusters(cluster.GetKubeClient())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, "no found storageclusters")
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get storageClusters failed %s", err.Error()))
	}

	var storageclusters []*types.StorageCluster
	for _, item := range k8sStorageClusters.Items {
		storageclusters = append(storageclusters, k8sStorageToSCStorage(cluster, &item))
	}
	return storageclusters, nil
}

func (m StorageClusterManager) Get(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	k8sStorageCluster, err := getStorageCluster(cluster.GetKubeClient(), storagecluster.GetID())
	if err != nil {
		if apierrors.IsNotFound(err) {
			return nil, resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("no found storagecluster %s", storagecluster.GetID()))
		}
		return nil, resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("get storageCluster %s failed %s", storagecluster.GetID(), err.Error()))
	}

	return k8sStorageToSCStorageDetail(cluster, clusteragent.GetAgent(), k8sStorageCluster), nil
}

func (m StorageClusterManager) Delete(ctx *resource.Context) *resterr.APIError {
	if isAdmin(getCurrentUser(ctx)) == false {
		return resterr.NewAPIError(resterr.PermissionDenied, "only admin can delete storagecluster")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return resterr.NewAPIError(resterr.NotFound, "storagecluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	if err := deleteStorageCluster(cluster.GetKubeClient(), storagecluster.GetID()); err != nil {
		if apierrors.IsNotFound(err) {
			return resterr.NewAPIError(resterr.NotFound, fmt.Sprintf("storagecluster %s doesn't exist", storagecluster.GetID()))
		} else if strings.Contains(err.Error(), "is used by") || strings.Contains(err.Error(), "Creating") {
			return resterr.NewAPIError(resterr.PermissionDenied, fmt.Sprintf("delete storagecluster failed, %s", err.Error()))
		} else {
			return resterr.NewAPIError(resterr.ServerError, fmt.Sprintf("delete storagecluster failed, %s", err.Error()))
		}
	}
	return nil
}

func (m StorageClusterManager) Create(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can create storagecluster")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	if err := createStorageCluster(cluster, clusteragent.GetAgent(), storagecluster); err != nil {
		if apierrors.IsAlreadyExists(err) {
			return nil, resterr.NewAPIError(resterr.DuplicateResource, fmt.Sprintf("duplicate storagecluster name %s", storagecluster.Name))
		} else if strings.Contains(err.Error(), "storagecluster has already exists") || strings.Contains(err.Error(), "can not be used for") {
			return nil, resterr.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("create storagecluster failed, %s", err.Error()))
		} else {
			return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("create storagecluster failed, %s", err.Error()))
		}
	}
	storagecluster.SetID(types.StorageclusterMap[storagecluster.StorageType])
	return storagecluster, nil
}

func (m StorageClusterManager) Update(ctx *resource.Context) (resource.Resource, *resterr.APIError) {
	if isAdmin(getCurrentUser(ctx)) == false {
		return nil, resterr.NewAPIError(resterr.PermissionDenied, "only admin can update storagecluster")
	}
	cluster := m.clusters.GetClusterForSubResource(ctx.Resource)
	if cluster == nil {
		return nil, resterr.NewAPIError(resterr.NotFound, "cluster doesn't exist")
	}

	storagecluster := ctx.Resource.(*types.StorageCluster)
	if err := updateStorageCluster(cluster, clusteragent.GetAgent(), storagecluster); err != nil {
		if strings.Contains(err.Error(), "storagecluster must keep") || strings.Contains(err.Error(), "is used by") || strings.Contains(err.Error(), "can not be used for") || strings.Contains(err.Error(), "Creating") {
			return nil, resterr.NewAPIError(types.InvalidClusterConfig, fmt.Sprintf("update storagecluster failed, %s", err.Error()))
		} else {
			return nil, resterr.NewAPIError(types.ConnectClusterFailed, fmt.Sprintf("update storagecluster failed, %s", err.Error()))
		}
	}
	return storagecluster, nil
}

func getStorageCluster(cli client.Client, name string) (*storagev1.Cluster, error) {
	storagecluster := storagev1.Cluster{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", name}, &storagecluster)
	return &storagecluster, err
}

func getStorageClusters(cli client.Client) (*storagev1.ClusterList, error) {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	return &storageclusters, err
}

func deleteStorageCluster(cli client.Client, name string) error {
	k8sStorageCluster, err := getStorageCluster(cli, name)
	if err != nil {
		return err
	}
	if k8sStorageCluster.Status.Phase == storagev1.Creating || k8sStorageCluster.Status.Phase == storagev1.Updating || k8sStorageCluster.Status.Phase == storagev1.Deleting {
		return errors.New("storagecluster in Creating, Updating or Deleting, not allowed delete")
	}

	if err := checkFinalizers(cli, name); err != nil {
		return err
	}
	storagecluster := &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{Name: name},
	}
	return cli.Delete(context.TODO(), storagecluster)
}

func createStorageCluster(cluster *zke.Cluster, agent *clusteragent.AgentManager, storagecluster *types.StorageCluster) error {
	if err := checkStorageClusterExist(cluster.GetKubeClient(), storagecluster.StorageType); err != nil {
		return err
	}
	if err := isHostsValidate(cluster, agent, storagecluster.Hosts); err != nil {
		return err
	}

	k8sStorageCluster := scStorageToK8sStorage(storagecluster)
	return cluster.GetKubeClient().Create(context.TODO(), k8sStorageCluster)
}

func updateStorageCluster(cluster *zke.Cluster, agent *clusteragent.AgentManager, storagecluster *types.StorageCluster) error {
	if len(storagecluster.Hosts) == 0 {
		return errors.New("storagecluster must keep at least one node,suggest delete the storagecluster")
	}
	k8sStorageCluster, err := getStorageCluster(cluster.GetKubeClient(), storagecluster.GetID())
	if err != nil {
		return err
	}
	if k8sStorageCluster.Status.Phase == storagev1.Creating || k8sStorageCluster.Status.Phase == storagev1.Updating || k8sStorageCluster.Status.Phase == storagev1.Deleting {
		return errors.New("storagecluster in Creating, Updating or Deleting, not allowed update")
	}
	if k8sStorageCluster.GetDeletionTimestamp() != nil {
		return errors.New("storagecluster in Deleting, not allowed update")
	}
	if storagecluster.StorageType != k8sStorageCluster.Spec.StorageType {
		return errors.New("storagecluster type can not be modify")
	}

	if k8sStorageCluster.Spec.StorageType == "lvm" {
		if err := isDelHostsUsed(cluster.GetKubeClient(), k8sStorageCluster, storagecluster); err != nil {
			return err
		}
	}
	if err := isAddHostsValid(cluster, agent, k8sStorageCluster, storagecluster); err != nil {
		return err
	}

	k8sStorageCluster.Spec.Hosts = storagecluster.Hosts
	return cluster.GetKubeClient().Update(context.TODO(), k8sStorageCluster)
}

func scStorageToK8sStorage(storagecluster *types.StorageCluster) *storagev1.Cluster {
	return &storagev1.Cluster{
		ObjectMeta: metav1.ObjectMeta{
			Name: types.StorageclusterMap[storagecluster.StorageType],
		},
		Spec: storagev1.ClusterSpec{
			StorageType: storagecluster.StorageType,
			Hosts:       storagecluster.Hosts,
		},
	}
}

func k8sStorageToSCStorage(cluster *zke.Cluster, k8sStorageCluster *storagev1.Cluster) *types.StorageCluster {
	tSize := byteToGb(sToi(k8sStorageCluster.Status.Capacity.Total.Total))
	uSize := byteToGb(sToi(k8sStorageCluster.Status.Capacity.Total.Used))
	fSize := byteToGb(sToi(k8sStorageCluster.Status.Capacity.Total.Free))
	storagecluster := &types.StorageCluster{
		Name:        k8sStorageCluster.Name,
		StorageType: k8sStorageCluster.Spec.StorageType,
		Hosts:       k8sStorageCluster.Spec.Hosts,
		Phase:       string(k8sStorageCluster.Status.Phase),
		Size:        tSize,
		UsedSize:    uSize,
		FreeSize:    fSize,
	}
	storagecluster.SetID(k8sStorageCluster.Name)
	storagecluster.SetCreationTimestamp(k8sStorageCluster.CreationTimestamp.Time)
	if k8sStorageCluster.GetDeletionTimestamp() != nil {
		storagecluster.SetDeletionTimestamp(k8sStorageCluster.DeletionTimestamp.Time)
	}
	return storagecluster
}

func k8sStorageToSCStorageDetail(cluster *zke.Cluster, agent *clusteragent.AgentManager, k8sStorageCluster *storagev1.Cluster) *types.StorageCluster {
	var info types.Storage
	if err := agent.GetResource(cluster.Name, "/storages/"+k8sStorageCluster.Spec.StorageType, &info); err != nil {
		log.Warnf("get storages from clusteragent failed:%s", err.Error())
	}

	storagecluster := k8sStorageToSCStorage(cluster, k8sStorageCluster)
	storagecluster.Nodes = countSize(k8sStorageCluster)
	storagecluster.PVs = info.PVs
	return storagecluster
}

func checkStorageClusterExist(cli client.Client, storageType string) error {
	storageclusters := storagev1.ClusterList{}
	err := cli.List(context.TODO(), nil, &storageclusters)
	if err != nil {
		return err
	}
	for _, storage := range storageclusters.Items {
		if storage.Spec.StorageType == storageType {
			return errors.New(fmt.Sprintf("The type of %s storagecluster has already exists", storageType))
		}
	}
	return nil
}

func checkFinalizers(cli client.Client, name string) error {
	var obj runtime.Object
	obj, err := getStorageCluster(cli, name)
	if err != nil {
		return err
	}
	metaObj := obj.(metav1.Object)
	finalizers := metaObj.GetFinalizers()
	if (len(finalizers) == 0) || (len(finalizers) == 1 && slice.SliceIndex(finalizers, common.ClusterPrestopHookFinalizer) == 0) {
		return nil
	} else {
		return errors.New(fmt.Sprintf("The storagecluster %s is used by some pods, you should stop those pods first", name))
	}
}

func sToi(size string) int64 {
	num, _ := strconv.ParseInt(size, 10, 64)
	return num
}

func byteToGb(num int64) string {
	f := float64(num) / (1024 * 1024 * 1024)
	return fmt.Sprintf("%.2f", math.Trunc(f*1e2)*1e-2)
}

func countSize(k8sStorageCluster *storagev1.Cluster) []types.StorageNode {
	var nodes types.StorageNodes
	ns := make(map[string]map[string]int64)
	nodestat := make(map[string]bool)
	stat := true
	for _, i := range k8sStorageCluster.Status.Capacity.Instances {
		if !i.Stat {
			stat = false
		}
		nodestat[i.Host] = stat
		v, ok := ns[i.Host]
		if ok {
			v["Total"] += sToi(i.Info.Total)
			v["Used"] += sToi(i.Info.Used)
			v["Free"] += sToi(i.Info.Free)
		} else {
			info := make(map[string]int64)
			info["Total"] = sToi(i.Info.Total)
			info["Used"] = sToi(i.Info.Used)
			info["Free"] = sToi(i.Info.Free)
			ns[i.Host] = info
		}
	}
	for k, v := range ns {
		node := types.StorageNode{
			Name:     k,
			Size:     byteToGb(v["Total"]),
			UsedSize: byteToGb(v["Used"]),
			FreeSize: byteToGb(v["Free"]),
			Stat:     nodestat[k],
		}
		nodes = append(nodes, node)
	}
	sort.Sort(nodes)
	return nodes
}

func getStorageDriver(cli client.Client, storageType string) (string, error) {
	storageClassse := k8sstorage.StorageClass{}
	err := cli.Get(context.TODO(), k8stypes.NamespacedName{"", storageType}, &storageClassse)
	if err != nil {
		return "", err
	}
	return storageClassse.Provisioner, nil
}

func getAttachedHosts(cli client.Client, driverName string, nodes []string) ([]string, error) {
	var hosts []string
	volumeAttachments := k8sstorage.VolumeAttachmentList{}
	err := cli.List(context.TODO(), nil, &volumeAttachments)
	if err != nil {
		return hosts, err
	}
	for _, volumeAttachment := range volumeAttachments.Items {
		if driverName != volumeAttachment.Spec.Attacher {
			continue
		}
		if slice.SliceIndex(nodes, volumeAttachment.Spec.NodeName) >= 0 {
			if slice.SliceIndex(hosts, volumeAttachment.Spec.NodeName) >= 0 {
				continue
			}
			hosts = append(hosts, volumeAttachment.Spec.NodeName)
		}
	}
	return hosts, nil
}

func isAddHostsValid(cluster *zke.Cluster, agent *clusteragent.AgentManager, k8sStorageCluster *storagev1.Cluster, storagecluster *types.StorageCluster) error {
	s1 := set.StringSetFromSlice(k8sStorageCluster.Spec.Hosts)
	s2 := set.StringSetFromSlice(storagecluster.Hosts)
	addHosts := s2.Difference(s1).ToSlice()
	return isHostsValidate(cluster, agent, addHosts)
}

func isDelHostsUsed(cli client.Client, k8sStorageCluster *storagev1.Cluster, storagecluster *types.StorageCluster) error {
	driverName, err := getStorageDriver(cli, k8sStorageCluster.Spec.StorageType)
	if err != nil {
		return err
	}

	s1 := set.StringSetFromSlice(k8sStorageCluster.Spec.Hosts)
	s2 := set.StringSetFromSlice(storagecluster.Hosts)
	delHosts := s1.Difference(s2).ToSlice()

	usedHosts, err := getAttachedHosts(cli, driverName, delHosts)
	if err != nil {
		return err
	}
	if len(usedHosts) > 0 {
		return errors.New(fmt.Sprintf("The storagehosts %s is used by some pods, you should stop those pods first", usedHosts))
	}
	return nil
}

func isHostsValidate(cluster *zke.Cluster, agent *clusteragent.AgentManager, hosts []string) error {
	resp, err := getBlockDevices(cluster.Name, cluster.GetKubeClient(), agent)
	if err != nil {
		return err
	}
	var invalidHosts []string
	for _, host := range hosts {
		if !checkUsed(resp, host) {
			continue
		}
		invalidHosts = append(invalidHosts, host)
	}
	if len(invalidHosts) > 0 {
		return errors.New(fmt.Sprintf("hosts %s can not be used for storage", invalidHosts))
	}
	return nil
}

func checkUsed(blockinfo []*types.BlockDevice, node string) bool {
	for _, info := range blockinfo {
		if info.NodeName == node && info.UsedBy == "" {
			return false
		}
	}
	return true
}
