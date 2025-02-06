package config

import (
	"sync"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
)

type GpuNodeState interface {
	Get(nodeName string) *tfv1.GPUNode
	Set(nodeName string, gn *tfv1.GPUNode)
	Delete(nodeName string)
}

type GpuNodeStateImpl struct {
	gpuNodeMap map[string]*tfv1.GPUNode
	lock       sync.RWMutex
}

func NewGpuNodeStateImpl() *GpuNodeStateImpl {
	return &GpuNodeStateImpl{
		gpuNodeMap: make(map[string]*tfv1.GPUNode),
	}
}

func (g *GpuNodeStateImpl) Get(nodeName string) *tfv1.GPUNode {
	g.lock.RLock()
	defer g.lock.RUnlock()

	return g.gpuNodeMap[nodeName]
}

func (g *GpuNodeStateImpl) Set(nodeName string, gn *tfv1.GPUNode) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.gpuNodeMap[nodeName] = gn
}

func (g *GpuNodeStateImpl) Delete(nodeName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.gpuNodeMap, nodeName)
}
