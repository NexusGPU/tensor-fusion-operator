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
