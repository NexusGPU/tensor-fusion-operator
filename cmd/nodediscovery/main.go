package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/reporter"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var dryRun bool
	var hostname string

	flag.BoolVar(&dryRun, "dry-run", false, "dry run mode")
	flag.StringVar(&hostname, "hostname", "", "hostname")
	if hostname == "" {
		hostname = os.Getenv("HOSTNAME")
	}

	opts := zap.Options{
		Development: true,
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	ret := nvml.Init()
	if ret != nvml.SUCCESS {
		ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to initialize NVML")
		os.Exit(1)
	}
	defer func() {
		ret := nvml.Shutdown()
		if ret != nvml.SUCCESS {
			ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to shutdown NVML")
			os.Exit(1)
		}
	}()

	var r reporter.Reporter
	if dryRun {
		r = reporter.NewDryRunReporter()
	} else {
		var err error
		r, err = reporter.NewKubeReporter("")
		if err != nil {
			ctrl.Log.Error(err, "failed to create KubeReporter.")
			os.Exit(1)
		}
	}

	count, ret := nvml.DeviceGetCount()
	if ret != nvml.SUCCESS {
		ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to get device count")
		os.Exit(1)
	}

	ctx := context.Background()
	for i := 0; i < count; i++ {
		device, ret := nvml.DeviceGetHandleByIndex(i)
		if ret != nvml.SUCCESS {
			ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to get device", "index", i)
			os.Exit(1)
		}

		uuid, ret := device.GetUUID()
		if ret != nvml.SUCCESS {
			ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to get uuid of device", "index", i)
			os.Exit(1)
		}

		deviceName, ret := device.GetName()
		if ret != nvml.SUCCESS {
			ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to get name of device", "index", i)
			os.Exit(1)
		}

		memInfo, ret := device.GetMemoryInfo_v2()
		if ret != nvml.SUCCESS {
			ctrl.Log.Error(errors.New(nvml.ErrorString(ret)), "unable to get memory info of device", "index", i)
			os.Exit(1)
		}
		gpu := &tfv1.GPU{
			ObjectMeta: metav1.ObjectMeta{
				Name: uuid,
			},
			Status: tfv1.GPUStatus{
				Capacity: tfv1.Resource{
					Vram: resource.MustParse(fmt.Sprintf("%dKi", memInfo.Total)),
					// TODO: compute Tflops based on GPU model
					Tflops: resource.MustParse("100"),
				},
				UUID:     uuid,
				GPUModel: deviceName,
				NodeSelector: map[string]string{
					"kubernetes.io/hostname": hostname,
				},
			},
		}
		gpuCopy := gpu.DeepCopy()
		if err := r.Report(ctx, gpu, func() error {
			// keep Available field
			available := gpu.Status.Available
			gpu.Status = gpuCopy.Status
			gpu.Status.Available = available
			return nil
		}); err != nil {
			ctrl.Log.Error(err, "failed to report GPU", "gpu", gpu)
			os.Exit(1)
		}
	}
}
