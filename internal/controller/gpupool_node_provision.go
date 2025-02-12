package controller

import (
	"context"
	"fmt"
	"sync"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/common"
	"github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/types"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	corev1 "k8s.io/api/core/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/log"

	cloudprovider "github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider"
)

// Controller and trigger logic for abstract layer of node provisioning
// TODO: implement the logic

func (r *GPUPoolReconciler) reconcilePoolCapacityWithProvisioner(ctx context.Context, pool *tfv1.GPUPool) (bool, error) {
	log := log.FromContext(ctx)
	// check if min resource constraint is satisfied
	shouldScaleUp := false
	tflopsGap := int64(0)
	vramGap := int64(0)

	totalTFlops, _ := pool.Status.TotalTFlops.AsInt64()
	totalVRAM, _ := pool.Status.TotalVRAM.AsInt64()

	// default warmUp is zero, only scale up when available < 0
	warmUpTFlops := int64(0)
	warmUpVRAM := int64(0)
	if pool.Spec.CapacityConfig.WarmResources != nil {
		warmUpTFlops, _ = pool.Spec.CapacityConfig.WarmResources.TFlops.AsInt64()
		warmUpVRAM, _ = pool.Spec.CapacityConfig.WarmResources.VRAM.AsInt64()
	}

	if pool.Spec.CapacityConfig.MinResources != nil {
		minTFlops, _ := pool.Spec.CapacityConfig.MinResources.TFlops.AsInt64()
		minVRAM, _ := pool.Spec.CapacityConfig.MinResources.VRAM.AsInt64()

		tflopsGap = minTFlops - totalTFlops
		vramGap = minVRAM - totalVRAM

		shouldScaleUp = (tflopsGap > 0) || (vramGap > 0)
		if shouldScaleUp {
			log.Info("Should scale up GPU node due gap of currentTotal <-> min capacity", "pool", pool.Name)
		}

	}

	if !shouldScaleUp {
		availableTFlops, _ := pool.Status.AvailableTFlops.AsInt64()
		availableVRAM, _ := pool.Status.AvailableVRAM.AsInt64()

		tflopsGap = warmUpTFlops - availableTFlops
		vramGap = warmUpVRAM - availableVRAM

		shouldScaleUp = (tflopsGap > 0) || (vramGap > 0)
		if !shouldScaleUp {
			log.Info("Should scale up GPU node due gap of available <-> warmup capacity", "pool", pool.Name)
		}
	}

	if shouldScaleUp && pool.Spec.CapacityConfig.MaxResources != nil {
		maxTFlops, _ := pool.Spec.CapacityConfig.MaxResources.TFlops.AsInt64()
		maxVRAM, _ := pool.Spec.CapacityConfig.MaxResources.VRAM.AsInt64()

		if totalTFlops >= maxTFlops || totalVRAM >= maxVRAM {
			shouldScaleUp = false

			log.Info("Should not scale up GPU node due to max capacity constraint", "pool", pool.Name)

			r.Recorder.Eventf(pool, corev1.EventTypeWarning, "MaxResourceConstraintReached", "Max resource constraint can not be satisfied, can not scale up: %v", pool.Spec.CapacityConfig.MaxResources)
		}
	}

	if !shouldScaleUp {
		return false, nil
	}

	// create provisioner
	provider, cluster, err := createProvisionerByCluster(ctx, pool, r)
	if err != nil {
		return false, err
	}

	nodeClass := pool.Spec.NodeManagerConfig.NodeProvisioner.NodeClass
	if nodeClass == "" {
		return false, fmt.Errorf("failed to get node class for pool %s", pool.Name)
	}
	var nodeClassObj tfv1.GPUNodeClass
	err = r.Get(ctx, client.ObjectKey{Name: nodeClass}, &nodeClassObj)
	if err != nil {
		return false, err
	}

	// convert resource gap to least cost GPUNode creation param
	gpuNodeParams, err := common.CalculateLeastCostGPUNodes(ctx, provider, cluster, pool, &nodeClassObj, tflopsGap, vramGap)
	if err != nil {
		return false, err
	}

	var wg sync.WaitGroup
	wg.Add(len(gpuNodeParams))

	var errList []error

	for _, node := range gpuNodeParams {
		go func(node types.NodeCreationParam) {
			defer wg.Done()

			status, err := provider.CreateNode(ctx, &node)
			if err != nil {
				errList = append(errList, err)
				return
			}
			r.Recorder.Eventf(pool, corev1.EventTypeNormal, "GPUNodeCreated", "Created node: %s, IP: %s", status.InstanceID, status.PrivateIP)
		}(node)
	}

	wg.Wait()

	if len(errList) > 0 {
		return false, fmt.Errorf("failed to create nodes: %v", errList)
	}
	return len(gpuNodeParams) > 0, nil
}

func createProvisionerByCluster(ctx context.Context, pool *tfv1.GPUPool, r *GPUPoolReconciler) (types.GPUNodeProvider, *tfv1.TensorFusionCluster, error) {
	clusterName := pool.Labels[constants.LabelKeyOwner]
	if clusterName == "" {
		return nil, nil, fmt.Errorf("failed to get cluster name for pool %s", pool.Name)
	}

	cluster := tfv1.TensorFusionCluster{}
	if err := r.Get(ctx, client.ObjectKey{Name: clusterName}, &cluster); err != nil {
		return nil, nil, err
	}

	vendorCfg := cluster.Spec.ComputingVendor
	if vendorCfg == nil {
		return nil, nil, fmt.Errorf("failed to get computing vendor config for cluster %s", clusterName)
	}

	provider, err := cloudprovider.GetProvider(*vendorCfg)
	if err != nil {
		return nil, nil, err
	}

	return *provider, &cluster, nil
}
