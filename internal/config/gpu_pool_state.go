package config

import (
	"encoding/json"
	"fmt"
	"sync"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/samber/lo"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/runtime"
	schedulingcorev1 "k8s.io/component-helpers/scheduling/corev1"
	"k8s.io/utils/ptr"
)

type GpuPoolState interface {
	Get(poolName string) *tfv1.GPUPoolSpec
	Set(poolName string, gps *tfv1.GPUPoolSpec)
	Delete(poolName string)
	Subscribe(poolName string)
	GetMatchedPoolName(node *corev1.Node) (string, error)
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

func (g *GpuPoolStateImpl) GetMatchedPoolName(node *corev1.Node) (string, error) {
	for k, v := range g.gpuPoolMap {
		matches, err := schedulingcorev1.MatchNodeSelectorTerms(node, v.NodeManagerConfig.NodeSelector)
		if err != nil {
			return "", err
		}

		if matches {
			return k, nil
		}
	}
	return "", fmt.Errorf("no matched GPU pool")
}

type MockGpuPoolState struct {
	g *GpuPoolStateImpl
}

var MockGpuPoolSpec = tfv1.GPUPoolSpec{
	NodeManagerConfig: &tfv1.NodeManagerConfig{
		NodeSelector: &corev1.NodeSelector{
			NodeSelectorTerms: []corev1.NodeSelectorTerm{
				{
					MatchExpressions: []corev1.NodeSelectorRequirement{
						{
							Key:      "mock-label",
							Operator: "In",
							Values:   []string{"true"},
						},
					},
				},
			},
		},
	},
	ComponentConfig: &tfv1.ComponentConfig{
		Hypervisor: &tfv1.HypervisorConfig{
			PodTemplate: &runtime.RawExtension{
				Raw: lo.Must(json.Marshal(
					corev1.PodTemplate{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy: corev1.RestartPolicyOnFailure,
								Containers: []corev1.Container{
									{
										Name:    "tensorfusion-hypervisor",
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
		NodeDiscovery: &tfv1.NodeDiscoveryConfig{
			PodTemplate: &runtime.RawExtension{
				Raw: lo.Must(json.Marshal(
					corev1.PodTemplate{
						Template: corev1.PodTemplateSpec{
							Spec: corev1.PodSpec{
								RestartPolicy:                 corev1.RestartPolicyOnFailure,
								TerminationGracePeriodSeconds: ptr.To[int64](0),
								Containers: []corev1.Container{
									{
										Name:    "tensorfusion-node-discovery",
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
		Worker: &tfv1.WorkerConfig{
			PodTemplate: &runtime.RawExtension{
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
		Client: &tfv1.ClientConfig{
			OperatorEndpoint: "http://localhost:8080",
			PatchToPod: &runtime.RawExtension{
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
			PatchToContainer: &runtime.RawExtension{
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

func (m *MockGpuPoolState) GetMatchedPoolName(node *corev1.Node) (string, error) {
	return m.g.GetMatchedPoolName(node)
}
