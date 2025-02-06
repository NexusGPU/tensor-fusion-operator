package config

import (
	"sync"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
)

type ScheduleTemplateState interface {
	Get(templateName string) *tfv1.SchedulingConfigTemplate
	Set(templateName string, tmpl *tfv1.SchedulingConfigTemplate)
	Delete(templateName string)
}

type ScheduleTemplateStateImpl struct {
	templateMap map[string]*tfv1.SchedulingConfigTemplate
	lock        sync.RWMutex
}

func NewScheduleTemplateStateImpl() *ScheduleTemplateStateImpl {
	return &ScheduleTemplateStateImpl{
		templateMap: make(map[string]*tfv1.SchedulingConfigTemplate),
	}
}

func (g *ScheduleTemplateStateImpl) Get(templateName string) *tfv1.SchedulingConfigTemplate {
	g.lock.RLock()
	defer g.lock.RUnlock()

	return g.templateMap[templateName]
}

func (g *ScheduleTemplateStateImpl) Set(templateName string, tmpl *tfv1.SchedulingConfigTemplate) {
	g.lock.Lock()
	defer g.lock.Unlock()

	g.templateMap[templateName] = tmpl
}

func (g *ScheduleTemplateStateImpl) Delete(templateName string) {
	g.lock.Lock()
	defer g.lock.Unlock()

	delete(g.templateMap, templateName)
}
