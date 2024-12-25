package worker

import (
	"fmt"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
)

type WorkerGenerator struct {
	WorkerConfig *config.Worker
}

func (wg *WorkerGenerator) GenerateConnectionURL(connection *tfv1.TensorFusionConnection, pod *corev1.Pod) string {
	return fmt.Sprintf("native+%s+%d", pod.Status.PodIP, wg.WorkerConfig.Port)
}

func (wg *WorkerGenerator) GenerateWorkerPod(
	gpu *tfv1.GPU,
	connection *tfv1.TensorFusionConnection,
	namespacedName types.NamespacedName,
) *corev1.Pod {

	spec := wg.WorkerConfig.Template.Spec
	if spec.NodeSelector == nil {
		spec.NodeSelector = make(map[string]string)
	}
	spec.NodeSelector = gpu.Status.NodeSelector

	spec.Containers[0].Env = append(spec.Containers[0].Env, corev1.EnvVar{
		Name:  "NVIDIA_VISIBLE_DEVICES",
		Value: gpu.Status.UUID,
	})

	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: spec,
	}
}
