package common

import (
	"fmt"
	"math"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/types"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	"golang.org/x/exp/rand"
	corev1 "k8s.io/api/core/v1"
)

// Pool config contains node requirements, nodeClass indicates some base template info for creating VM nodes, this func should output the list of VM to be created to meet TFlops and VRAM gap
// Simple algorithm, try to find the instance type that best meets the gap
// TODO: implement a more advanced algorithm to combine multiple instance types
func CalculateLeastCostGPUNodes(provider types.GPUNodeProvider, cluster *tfv1.TensorFusionCluster, pool *tfv1.GPUPool, nodeClass *tfv1.GPUNodeClass, tflopsGap int64, vramGap int64) ([]types.NodeCreationParam, error) {
	if tflopsGap <= 0 && vramGap <= 0 {
		return []types.NodeCreationParam{}, nil
	}

	requirements := pool.Spec.NodeManagerConfig.NodeProvisioner.GPURequirements
	region := cluster.Spec.ComputingVendor.Params.DefaultRegion
	zones := []string{}
	os := constants.GPUNodeOSLinux

	eligibleInstances := getEligibleInstances(pool, region, provider)
	if len(eligibleInstances) == 0 {
		return []types.NodeCreationParam{}, nil
	}

	var bestInstance *types.GPUNodeInstanceInfo
	var bestCapacityType types.CapacityTypeEnum
	minCost := math.MaxFloat64
	var bestNumInstances int64

	// Default to spot, it's cheaper
	preferredCapacityType := types.CapacityTypeSpot

	for _, req := range requirements {
		if req.Key == tfv1.NodeRequirementKeyCapacityType && req.Operator == corev1.NodeSelectorOpIn {
			// user can specify other capacity types
			preferredCapacityType = types.CapacityTypeEnum(req.Values[0])
		} else if req.Key == tfv1.NodeRequirementKeyZone && req.Operator == corev1.NodeSelectorOpIn {
			zones = req.Values
		} else if req.Key == tfv1.NodeRequirementKeyOS && req.Operator == corev1.NodeSelectorOpIn {
			os = req.Values[0]
		}
	}
	if len(zones) == 0 {
		return nil, fmt.Errorf("no zones found in node requirements")
	}

	for _, instance := range eligibleInstances {

		costPerHour, err := provider.GetInstancePricing(instance.InstanceType, region, preferredCapacityType)
		if err != nil {
			continue
		}

		tflopsPerInstance := instance.FP16TFlopsPerGPU * instance.GPUCount
		vramPerInstance := instance.VRAMGigabytesPerGPU * instance.GPUCount

		if tflopsPerInstance <= 0 || vramPerInstance <= 0 {
			continue // Avoid division by zero
		}

		tflopsRequired := math.Ceil(float64(tflopsGap) / float64(tflopsPerInstance))
		vramRequired := math.Ceil(float64(vramGap) / float64(vramPerInstance))
		numInstances := int64(math.Max(tflopsRequired, vramRequired))

		totalCost := costPerHour * float64(numInstances)

		if totalCost < minCost {
			minCost = totalCost
			bestInstance = &instance
			bestNumInstances = numInstances
		}
		// if the instance type is too large to fit the gap, wait next compaction controller reconcile to make half
	}

	if bestInstance == nil {
		return nil, fmt.Errorf("no eligible instances found")
	}

	nodes := make([]types.NodeCreationParam, 0, bestNumInstances)
	for i := int64(0); i < bestNumInstances; i++ {
		// Zone and region should ideally be determined from nodeClass's subnet selectors
		nodes = append(nodes, types.NodeCreationParam{
			NodeName:     fmt.Sprintf("%s-%s-%d", pool.Name, generateRandomString(6), i+1),
			InstanceType: bestInstance.InstanceType,
			NodeClass:    nodeClass,
			Region:       region,
			Zone:         zones[rand.Intn(len(zones))],
			ExtraParams: map[string]string{
				string(tfv1.NodeRequirementKeyCapacityType): string(bestCapacityType),
				string(tfv1.NodeRequirementKeyOS):           os,
			},
		})
	}

	return nodes, nil
}

func getEligibleInstances(pool *tfv1.GPUPool, region string, provider types.GPUNodeProvider) []types.GPUNodeInstanceInfo {
	eligible := []types.GPUNodeInstanceInfo{}

	for _, instance := range provider.GetGPUNodeInstanceTypeInfo(region) {
		meetsRequirements := true

		for _, req := range pool.Spec.NodeManagerConfig.NodeProvisioner.GPURequirements {
			if req.Key == tfv1.NodeRequirementKeyCapacityType {
				continue
			}

			switch req.Key {
			case tfv1.NodeRequirementKeyInstanceType:
				if req.Operator == corev1.NodeSelectorOpIn {
					if !contains(req.Values, instance.InstanceType) {
						meetsRequirements = false
						break
					}
				}
			case tfv1.NodeRequirementKeyArchitecture:
				if req.Operator == corev1.NodeSelectorOpIn {
					if !contains(req.Values, string(instance.CPUArchitecture)) {
						meetsRequirements = false
						break
					}
				}
			case tfv1.NodeRequirementKeyGPUArchitecture:
				if req.Operator == corev1.NodeSelectorOpIn {
					if !contains(req.Values, string(instance.GPUArchitecture)) {
						meetsRequirements = false
						break
					}
				}

			}

		}

		if meetsRequirements {
			eligible = append(eligible, instance)
		}
	}

	return eligible
}

func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func generateRandomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyz"
	rand.Seed(uint64(time.Now().UnixNano()))

	result := make([]byte, length)
	for i := range result {
		result[i] = charset[rand.Intn(len(charset))]
	}
	return string(result)
}
