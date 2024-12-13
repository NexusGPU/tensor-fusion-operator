package config

import (
	"os"

	"gopkg.in/yaml.v2"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
)

type Pod struct {
	Spec PodSpec `json:"spec,omitempty"`
}

type PodSpec struct {
	InitContainers   []corev1.Container `json:"initContainers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,20,rep,name=initContainers"`
	Containers       []corev1.Container `json:"containers,omitempty" patchStrategy:"merge" patchMergeKey:"name" protobuf:"bytes,2,rep,name=containers"`
	RuntimeClassName *string            `json:"runtimeClassName,omitempty" protobuf:"bytes,29,opt,name=runtimeClassName"`
}

type Config struct {
	WorkerTemplate corev1.PodTemplate `json:"workerTemplate"`
	PodMutator     PodMutator         `json:"podMutator"`
}

type PodMutator struct {
	PatchStrategicMerge Pod             `json:"patchStrategicMerge"`
	PatchEnvVars        []corev1.EnvVar `json:"envVars"`
}

func LoadConfig(filename string) (*Config, error) {
	cfg := NewDefaultConfig()
	data, err := os.ReadFile(filename)
	if err != nil {
		return cfg, err
	}
	err = yaml.Unmarshal(data, cfg)
	if err != nil {
		return nil, err
	}
	return cfg, nil
}

func NewDefaultConfig() *Config {
	return &Config{
		WorkerTemplate: corev1.PodTemplate{
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
		PodMutator: PodMutator{
			PatchStrategicMerge: Pod{
				Spec: PodSpec{
					InitContainers: []corev1.Container{
						{
							Name:  "inject-lib",
							Image: "busybox:stable-glibc",
						},
					},
				},
			},
			PatchEnvVars: []corev1.EnvVar{
				{
					Name:  "LD_PRELOAD",
					Value: "tensorfusion.so",
				},
			},
		},
	}
}
