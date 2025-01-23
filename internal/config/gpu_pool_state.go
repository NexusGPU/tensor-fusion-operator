package config

import (
	"encoding/json"
	"sync"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/utils/ptr"
)

type GpuPoolState interface {
	Get(poolName string) *tfv1.GPUPoolSpec
	Set(poolName string, gps *tfv1.GPUPoolSpec)
	Delete(poolName string)
	Subscribe(poolName string)
}

type GpuPoolStateImpl struct {
	gpuPoolMap map[string]*tfv1.GPUPoolSpec
	lock       sync.RWMutex
}

func NewGpuPoolStateImpl() *GpuPoolStateImpl {
	return &GpuPoolStateImpl{
		gpuPoolMap: make(map[string]*tfv1.GPUPoolSpec),
	}
}

func (g *GpuPoolStateImpl) Get(poolName string) *tfv1.GPUPoolSpec {
	g.lock.RLock()
	defer g.lock.RUnlock()

	return g.gpuPoolMap[poolName]
}

func (g *GpuPoolStateImpl) Set(poolName string, gps *tfv1.GPUPoolSpec) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.gpuPoolMap[poolName] = gps
}

func (g *GpuPoolStateImpl) Delete(poolName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.gpuPoolMap, poolName)
}

func (g *GpuPoolStateImpl) Subscribe(poolName string) {
	// TODO: impl this
}

type MockGpuPoolState struct {
	g *GpuPoolStateImpl
}

var MockGpuPoolSpec = tfv1.GPUPoolSpec{
	ComponentConfig: tfv1.ComponentConfig{
		Worker: tfv1.WorkerConfig{
			PodTemplate: runtime.RawExtension{
				Raw: lo.Must(json.Marshal(
					corev1.PodTemplate{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								TerminationGracePeriodSeconds: ptr.To[int64](0),
								Containers: []corev1.Container{
									{
										Name:    "tensorfusion-worker",
										Image:   "busybox:stable-glibc",
										Command: []string{"sleep", "infinity"},
									},
								},
							},
						},
					},
				)),
			},
		},
		Client: tfv1.ClientConfig{
			OperatorEndpoint: "http://localhost:8080",
			PatchToPod: runtime.RawExtension{
				Raw: lo.Must(json.Marshal(map[string]any{
					"spec": map[string]any{
						"initContainers": []corev1.Container{
							{
								Name:  "inject-lib",
								Image: "busybox:stable-glibc",
							},
						},
					},
				})),
			},
			PatchToContainer: runtime.RawExtension{
				Raw: lo.Must(json.Marshal(map[string]any{
					"env": []corev1.EnvVar{
						{
							Name:  "LD_PRELOAD",
							Value: "tensorfusion.so",
						},
					},
				})),
			},
		},
	},
}

func NewMockGpuPoolState() *MockGpuPoolState {
	g := NewGpuPoolStateImpl()
	g.Set("mock", &MockGpuPoolSpec)

	return &MockGpuPoolState{g: g}
}

func (m *MockGpuPoolState) Get(poolName string) *tfv1.GPUPoolSpec {
	return m.g.Get(poolName)
}

func (m *MockGpuPoolState) Set(poolName string, gps *tfv1.GPUPoolSpec) {
	m.g.Set(poolName, gps)
}

func (m *MockGpuPoolState) Delete(poolName string) {
	m.g.Delete(poolName)
}

func (m *MockGpuPoolState) Subscribe(poolName string) {
	m.g.Subscribe(poolName)
}
