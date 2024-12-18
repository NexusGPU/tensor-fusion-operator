package config

import (
	"os"

	corev1 "k8s.io/api/core/v1"
	"k8s.io/utils/ptr"
	"sigs.k8s.io/yaml"
)

type Config struct {
	Worker      Worker      `json:"worker"`
	PodMutation PodMutation `json:"podMutation"`
}

type Worker struct {
	corev1.PodTemplate
	SendPort    int16 `json:"sendPort"`
	ReceivePort int16 `json:"receivePort"`
}

type PodMutation struct {
	OperatorEndpoint string         `json:"operatorEndpoint"`
	PatchToPod       map[string]any `json:"patchToPod"`
	PatchToContainer map[string]any `json:"patchToContainer"`
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
		Worker: Worker{
			SendPort:    1234,
			ReceivePort: 4321,
			PodTemplate: corev1.PodTemplate{
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
		},
		PodMutation: PodMutation{
			OperatorEndpoint: "http://localhost:8080",
			PatchToPod: map[string]any{
				"spec": map[string]any{
					"initContainers": []corev1.Container{
						{
							Name:  "inject-lib",
							Image: "busybox:stable-glibc",
						},
					},
				},
			},
			PatchToContainer: map[string]any{
				"env": []corev1.EnvVar{
					{
						Name:  "LD_PRELOAD",
						Value: "tensorfusion.so",
					},
				},
			},
		},
	}
}
