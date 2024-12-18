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

func (wg *WorkerGenerator) GenerateConnectionURL(_gpu *tfv1.GPU, connection *tfv1.TensorFusionConnection, pod *corev1.Pod) string {
	return fmt.Sprintf("native+%s+%d+%d", pod.Status.PodIP, wg.WorkerConfig.ReceivePort, wg.WorkerConfig.SendPort)
}

func (wg *WorkerGenerator) GenerateWorkerPod(
	connection *tfv1.TensorFusionConnection,
	namespacedName types.NamespacedName,
) *corev1.Pod {
	return &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      namespacedName.Name,
			Namespace: namespacedName.Namespace,
		},
		Spec: wg.WorkerConfig.Template.Spec,
	}
}
