package scheduler

import (
	"testing"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

func createGPU(name string, tflops, vram string) *tfv1.GPU {
	return &tfv1.GPU{
		ObjectMeta: metav1.ObjectMeta{
			Name: name,
		},
		Status: tfv1.GPUStatus{
			Available: &tfv1.Resource{
				Tflops: resource.MustParse(tflops),
				Vram:   resource.MustParse(vram),
			},
		},
	}
}

//nolint:unparam
func createRequest(tflops, vram string) tfv1.Resource {
	return tfv1.Resource{
		Tflops: resource.MustParse(tflops),
		Vram:   resource.MustParse(vram),
	}
}

func TestNaiveScheduler_Schedule(t *testing.T) {
	tests := []struct {
		name                string
		gpus                []*tfv1.GPU
		request             tfv1.Resource
		wantgpu             string
		wantError           bool
		wantRemainingTflops string
		wantRemainingVram   string
	}{
		{
			name: "simple match",
			gpus: []*tfv1.GPU{
				createGPU("gpu1", "100", "16Gi"),
			},
			request:             createRequest("50", "8Gi"),
			wantgpu:             "gpu1",
			wantError:           false,
			wantRemainingTflops: "50",
			wantRemainingVram:   "8Gi",
		},
		{
			name:      "no gpus",
			gpus:      []*tfv1.GPU{},
			request:   createRequest("50", "8Gi"),
			wantgpu:   "",
			wantError: true,
		},
		{
			name: "insufficient resources",
			gpus: []*tfv1.GPU{
				createGPU("gpu1", "40", "16Gi"),
			},
			request:   createRequest("50", "8Gi"),
			wantgpu:   "",
			wantError: true,
		},
		{
			name: "multiple gpus, first fit",
			gpus: []*tfv1.GPU{
				createGPU("gpu1", "40", "16Gi"),
				createGPU("gpu2", "100", "32Gi"),
			},
			request:             createRequest("50", "8Gi"),
			wantgpu:             "gpu2",
			wantError:           false,
			wantRemainingTflops: "50",
			wantRemainingVram:   "24Gi",
		},
		{
			name: "exact match",
			gpus: []*tfv1.GPU{
				createGPU("gpu1", "50", "8Gi"),
			},
			request:             createRequest("50", "8Gi"),
			wantgpu:             "gpu1",
			wantError:           false,
			wantRemainingTflops: "0",
			wantRemainingVram:   "0",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewNaiveScheduler()

			// Add gpus
			for _, gpu := range tt.gpus {
				s.OnAdd(gpu)
			}

			// Try to schedule
			got, err := s.Schedule(tt.request)

			// Check error
			if (err != nil) != tt.wantError {
				t.Errorf("Schedule() error = %v, wantError %v", err, tt.wantError)
				return
			}

			// Check result
			if !tt.wantError {
				if got == nil {
					t.Error("Schedule() returned nil gpu when error not expected")
					return
				}
				if got.Name != tt.wantgpu {
					t.Errorf("Schedule() got gpu = %v, want %v", got.Name, tt.wantgpu)
				}

				// Check remaining resources
				if tt.wantRemainingTflops != "" {
					wantTflops := resource.MustParse(tt.wantRemainingTflops)
					if got.Status.Available.Tflops.Cmp(wantTflops) != 0 {
						t.Errorf("Remaining Tflops = %v, want %v", got.Status.Available.Tflops.String(), tt.wantRemainingTflops)
					}
				}
				if tt.wantRemainingVram != "" {
					wantVram := resource.MustParse(tt.wantRemainingVram)
					if got.Status.Available.Vram.Cmp(wantVram) != 0 {
						t.Errorf("Remaining Vram = %v, want %v", got.Status.Available.Vram.String(), tt.wantRemainingVram)
					}
				}
			}
		})
	}
}

func TestNaiveScheduler_gpuOperations(t *testing.T) {
	s := NewNaiveScheduler()
	gpu1 := createGPU("gpu1", "100", "16Gi")
	request := createRequest("50", "8Gi")

	// Test OnAdd
	s.OnAdd(gpu1)
	got, err := s.Schedule(request)
	if err != nil || got.Name != "gpu1" {
		t.Errorf("After OnAdd: Schedule() got = %v, want gpu1", got)
	}

	// Test OnUpdate
	gpu1Updated := createGPU("gpu1", "40", "16Gi")
	s.OnUpdate(gpu1, gpu1Updated)
	_, err = s.Schedule(request)
	if err == nil {
		t.Error("After OnUpdate: Schedule() should fail with insufficient resources")
	}

	// Test OnDelete
	s.OnDelete(gpu1Updated)
	_, err = s.Schedule(request)
	if err == nil {
		t.Error("After OnDelete: Schedule() should fail with no gpus")
	}
}

func TestNaiveScheduler_Release(t *testing.T) {
	tests := []struct {
		name                string
		gpu                 *tfv1.GPU
		schedule            *tfv1.Resource
		release             *tfv1.Resource
		wantError           bool
		wantRemainingTflops string
		wantRemainingVram   string
	}{
		{
			name:      "release non-existent gpu",
			gpu:       createGPU("gpu1", "100", "16Gi"),
			release:   &tfv1.Resource{},
			wantError: true,
		},
		{
			name: "release after scheduling",
			gpu: &tfv1.GPU{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gpu1",
				},
				Status: tfv1.GPUStatus{
					Capacity: &tfv1.Resource{
						Tflops: resource.MustParse("100"),
						Vram:   resource.MustParse("16Gi"),
					},
					Available: &tfv1.Resource{
						Tflops: resource.MustParse("100"),
						Vram:   resource.MustParse("16Gi"),
					},
				},
			},
			schedule: &tfv1.Resource{
				Tflops: resource.MustParse("50"),
				Vram:   resource.MustParse("8Gi"),
			},
			release: &tfv1.Resource{
				Tflops: resource.MustParse("50"),
				Vram:   resource.MustParse("8Gi"),
			},
			wantError:           false,
			wantRemainingTflops: "100",
			wantRemainingVram:   "16Gi",
		},
		{
			name: "partial release",
			gpu: &tfv1.GPU{
				ObjectMeta: metav1.ObjectMeta{
					Name: "gpu1",
				},
				Status: tfv1.GPUStatus{
					Capacity: &tfv1.Resource{
						Tflops: resource.MustParse("100"),
						Vram:   resource.MustParse("16Gi"),
					},
					Available: &tfv1.Resource{
						Tflops: resource.MustParse("100"),
						Vram:   resource.MustParse("16Gi"),
					},
				},
			},
			schedule: &tfv1.Resource{
				Tflops: resource.MustParse("60"),
				Vram:   resource.MustParse("10Gi"),
			},
			release: &tfv1.Resource{
				Tflops: resource.MustParse("30"),
				Vram:   resource.MustParse("5Gi"),
			},
			wantError:           false,
			wantRemainingTflops: "70",
			wantRemainingVram:   "11Gi",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := NewNaiveScheduler()

			if !tt.wantError {
				// Add the gpu first
				s.OnAdd(tt.gpu)

				// Schedule some resources if needed
				if tt.schedule != nil {
					gpu, err := s.Schedule(*tt.schedule)
					if err != nil {
						t.Errorf("Schedule() error = %v", err)
						return
					}

					// Verify resources were allocated
					expectedTflops := tt.gpu.Status.Capacity.Tflops.DeepCopy()
					expectedVram := tt.gpu.Status.Capacity.Vram.DeepCopy()
					expectedTflops.Sub(tt.schedule.Tflops)
					expectedVram.Sub(tt.schedule.Vram)
					if gpu.Status.Available.Tflops.Cmp(expectedTflops) != 0 ||
						gpu.Status.Available.Vram.Cmp(expectedVram) != 0 {
						t.Errorf("Schedule() did not allocate resources correctly")
						return
					}
				}
			}

			err := s.Release(*tt.release, tt.gpu)
			if (err != nil) != tt.wantError {
				t.Errorf("Release() error = %v, wantError %v", err, tt.wantError)
				return
			}

			if !tt.wantError {
				// Verify resources were restored correctly
				gpu := s.gpus[tt.gpu.Name]
				if gpu.Status.Available.Tflops.String() != tt.wantRemainingTflops ||
					gpu.Status.Available.Vram.String() != tt.wantRemainingVram {
					t.Errorf("Release() resources incorrect, got tflops=%v vram=%v, want tflops=%v vram=%v",
						gpu.Status.Available.Tflops.String(),
						gpu.Status.Available.Vram.String(),
						tt.wantRemainingTflops,
						tt.wantRemainingVram)
				}
			}
		})
	}
}
