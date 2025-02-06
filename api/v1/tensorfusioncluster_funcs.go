/*
Copyright The CloudNativePG Contributors

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package v1

import (
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	"k8s.io/apimachinery/pkg/api/meta"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func (tfc *TensorFusionCluster) SetAsPending() {
	tfc.Status.Phase = constants.PhasePending

	meta.SetStatusCondition(&tfc.Status.Conditions, metav1.Condition{
		Type:   constants.ConditionStatusTypeReady,
		Status: metav1.ConditionFalse,
		Reason: "NewCreation",
	})
}

func (tfc *TensorFusionCluster) SetAsFailed(err error) {
	tfc.Status.Phase = constants.PhaseFailed

	meta.SetStatusCondition(&tfc.Status.Conditions, metav1.Condition{
		Type:    constants.ConditionStatusTypeReady,
		Status:  metav1.ConditionFalse,
		Message: err.Error(),
		Reason:  "Failed",
	})
}

func (tfc *TensorFusionCluster) SetAsUnknown(err error) {
	tfc.Status.Phase = constants.PhaseUnknown

	meta.SetStatusCondition(&tfc.Status.Conditions, metav1.Condition{
		Type:    constants.ConditionStatusTypeReady,
		Status:  metav1.ConditionFalse,
		Message: err.Error(),
		Reason:  "UnknownState",
	})
}

func (tfc *TensorFusionCluster) SetAsUpdating(conditions ...metav1.Condition) {
	tfc.Status.Phase = constants.PhaseUpdating

	meta.SetStatusCondition(&tfc.Status.Conditions, metav1.Condition{
		Type:   constants.ConditionStatusTypeReady,
		Status: metav1.ConditionFalse,
		Reason: "ResourcesUpdated",
	})

	for _, condition := range conditions {
		meta.SetStatusCondition(&tfc.Status.Conditions, condition)
	}
}

func (tfc *TensorFusionCluster) SetAsReady(conditions ...metav1.Condition) {
	tfc.Status.Phase = constants.PhaseRunning

	meta.SetStatusCondition(&tfc.Status.Conditions, metav1.Condition{
		Type:   constants.ConditionStatusTypeReady,
		Status: metav1.ConditionTrue,
		Reason: "CheckPassed",
	})

	for _, condition := range conditions {
		meta.SetStatusCondition(&tfc.Status.Conditions, condition)
	}
}

func (tfc *TensorFusionCluster) RefreshStatus(ownedPools []GPUPool) {
	tfc.Status.TotalPools = int32(len(tfc.Spec.GPUPools))

	tfc.Status.ReadyGPUPools = make([]string, len(tfc.Spec.GPUPools))
	tfc.Status.NotReadyGPUPools = make([]string, 0)
	tfc.Status.TotalNodes = 0
	tfc.Status.TotalGPUs = 0
	tfc.Status.TotalTFlops = resource.Quantity{}
	tfc.Status.TotalVRAM = resource.Quantity{}
	tfc.Status.AvailableTFlops = resource.Quantity{}
	tfc.Status.AvailableVRAM = resource.Quantity{}

	for i, gpuPool := range ownedPools {
		if gpuPool.Status.Phase != constants.PhaseRunning {
			tfc.Status.NotReadyGPUPools = append(tfc.Status.NotReadyGPUPools, gpuPool.Name)
		} else {
			tfc.Status.ReadyGPUPools[i] = gpuPool.Name
		}
		tfc.Status.TotalNodes += gpuPool.Status.TotalNodes
		tfc.Status.TotalGPUs += gpuPool.Status.TotalGPUs
		tfc.Status.TotalTFlops.Add(gpuPool.Status.TotalTFlops)
		tfc.Status.TotalVRAM.Add(gpuPool.Status.TotalVRAM)
		tfc.Status.AvailableTFlops.Add(gpuPool.Status.AvailableTFlops)
		tfc.Status.AvailableVRAM.Add(gpuPool.Status.AvailableVRAM)
	}
}
