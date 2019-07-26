package zke

import (
	"context"
	"time"

	"github.com/zdnscloud/singlecloud/pkg/types"

	"github.com/zdnscloud/cement/log"
	"github.com/zdnscloud/gok8s/cache"
	"github.com/zdnscloud/gok8s/client"
	"github.com/zdnscloud/gok8s/client/config"
	"github.com/zdnscloud/zke/core"
	"k8s.io/client-go/rest"
)

const (
	ClusterRetryInterval = time.Second * 10
)

func (c *Cluster) getTypesCluster() *types.Cluster {
	cluster := ZKEClusterToSCCluster(c)
	if c.KubeClient == nil {
		return cluster
	}
	version, err := c.KubeClient.ServerVersion()
	if err != nil {
		c.fsm.Event(GetInfoFailedEvent)
		return cluster
	}
	c.fsm.Event(GetInfoSuccessEvent)
	cluster.Version = version.GitVersion
	return cluster
}

func (c *Cluster) getStatus() types.ClusterStatus {
	return types.ClusterStatus(c.fsm.Current())
}

func (c *Cluster) initLoop(ctx context.Context, kubeConfig string, mgr *ZKEManager) {
	k8sConfig, err := config.BuildConfig([]byte(kubeConfig))
	if err != nil {
		log.Errorf("build cluster %s k8sconfig failed %s", c.Name, err)
		c.fsm.Event(InitFailedEvent)
		return
	}

	for {
		select {
		case <-ctx.Done():
			return
		default:
			kubeClient, err := client.New(k8sConfig, client.Options{})
			if err == nil {
				c.KubeClient = kubeClient
				if err := c.setCache(k8sConfig); err != nil {
					log.Errorf("set cluster %s cache failed %s", c.Name, err)
					c.fsm.Event(InitFailedEvent)
					return
				}
				c.fsm.Event(InitSuccessEvent, mgr)
				return
			}
			time.Sleep(ClusterRetryInterval)
		}
	}
}

func (c *Cluster) setCache(k8sConfig *rest.Config) error {
	c.K8sConfig = k8sConfig
	cache, err := cache.New(k8sConfig, cache.Options{})
	if err != nil {
		return err
	}
	go cache.Start(c.stopCh)
	cache.WaitForCacheSync(c.stopCh)
	c.Cache = cache
	return nil
}

func (c *Cluster) create(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			c.fsm.Event(CreateFailedEvent, mgr, state)
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	zkeState, k8sConfig, kubeClient, err := upCluster(ctx, c.config, &core.FullState{}, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		c.fsm.Event(CreateFailedEvent, mgr, state)
		return
	}

	c.KubeClient = kubeClient

	state.FullState = zkeState
	state.IsUnvailable = false

	if err := c.setCache(k8sConfig); err != nil {
		c.fsm.Event(CreateFailedEvent, mgr, state)
		return
	}

	c.fsm.Event(CreateSuccessEvent, mgr, state)
}

func (c *Cluster) update(ctx context.Context, state clusterState, mgr *ZKEManager) {
	defer func() {
		if r := recover(); r != nil {
			log.Errorf("zke pannic info %s", r)
			c.fsm.Event(UpdateFailedEvent, mgr, state)
		}
	}()

	logger, logCh := log.NewLog4jBufLogger(MaxZKELogLines, log.Info)
	defer logger.Close()
	c.logCh = logCh

	zkeState, k8sConfig, kubeClient, err := upCluster(ctx, c.config, state.FullState, logger)
	if err != nil {
		log.Errorf("zke err info %s", err)
		c.fsm.Event(UpdateFailedEvent, mgr, state)
		return
	}

	c.K8sConfig = k8sConfig
	c.KubeClient = kubeClient

	state.FullState = zkeState
	state.IsUnvailable = false

	c.fsm.Event(UpdateSuccessEvent, mgr, state)
}