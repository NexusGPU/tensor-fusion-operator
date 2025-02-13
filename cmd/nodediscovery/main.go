package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	"github.com/NVIDIA/go-nvml/pkg/nvml"
	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/config"
	"github.com/NexusGPU/tensor-fusion-operator/internal/constants"
	"github.com/NexusGPU/tensor-fusion-operator/internal/reporter"
	"github.com/samber/lo"
	"k8s.io/apimachinery/pkg/api/resource"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/log/zap"
)

func main() {
	var dryRun bool
	var hostname string

	var gpuInfoConfig string
	flag.BoolVar(&dryRun, "dry-run", false, "dry run mode")
	flag.StringVar(&hostname, "hostname", "", "hostname")
	flag.StringVar(&gpuInfoConfig, "gpu-info-config", "", "specify the path to gpuInfoConfig file")

	if hostname == "" {
		hostname = os.Getenv("HOSTNAME")
	}

	gpuNodeName := os.Getenv(constants.NodeDiscoveryReportGPUNodeEnvName)

	opts := zap.Options{
		Development: true,
	}

	opts.BindFlags(flag.CommandLine)
	flag.Parse()
	ctrl.SetLogger(zap.New(zap.UseFlagOptions(&opts)))

	gpuinfos, err := config.LoadGpuInfoFromFile(gpuInfoConfig)
	if err != nil {
		ctrl.Log.Error(err, "unable to read gpuInfoConfig file")
		os.Exit(1)
	}

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
	totalTFlops := resource.MustParse("0")
	totalVRAM := resource.MustParse("0Ki")
	availableTFlops := resource.MustParse("0")
	availableVRAM := resource.MustParse("0Ki")

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
		uuid = strings.ToLower(uuid)
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
		info, ok := lo.Find(gpuinfos, func(info config.GpuInfo) bool {
			return info.FullModelName == deviceName
		})
		tflops := info.Fp16TFlops
		if !ok {
			tflops = resource.MustParse("0")
		}
		gpu := &tfv1.GPU{
			ObjectMeta: metav1.ObjectMeta{
				Name: uuid,
			},
			Status: tfv1.GPUStatus{
				Capacity: &tfv1.Resource{
					Vram:   resource.MustParse(fmt.Sprintf("%dKi", memInfo.Total)),
					Tflops: tflops,
				},
				UUID:     uuid,
				GPUModel: deviceName,
				NodeSelector: map[string]string{
					"kubernetes.io/hostname": hostname,
				},
			},
		}
		_ = controllerutil.SetControllerReference(&tfv1.GPUNode{
			ObjectMeta: metav1.ObjectMeta{
				Name: gpuNodeName,
			},
		}, gpu, reporter.Scheme)

		gpuCopy := gpu.DeepCopy()
		if err := r.Report(ctx, gpu, func() error {
			// keep Available field
			available := gpu.Status.Available
			gpu.Status = gpuCopy.Status
			if available != nil {
				gpu.Status.Available = available
			} else {
				gpu.Status.Available = gpu.Status.Capacity
			}
			return nil
		}); err != nil {
			ctrl.Log.Error(err, "failed to report GPU", "gpu", gpu)
			os.Exit(1)
		}
		totalTFlops.Add(gpu.Status.Capacity.Tflops)
		totalVRAM.Add(gpu.Status.Capacity.Vram)
		availableTFlops.Add(gpu.Status.Available.Tflops)
		availableVRAM.Add(gpu.Status.Available.Vram)
	}

	ns := nodeStatus(hostname)
	ns.TotalTFlops = totalTFlops
	ns.TotalVRAM = totalVRAM
	ns.AvailableTFlops = availableTFlops
	ns.AvailableVRAM = availableVRAM

	if err := r.Report(ctx, &tfv1.GPUNode{
		ObjectMeta: metav1.ObjectMeta{
			Name: gpuNodeName,
		},
		Status: *ns,
	}, func() error { return nil }); err != nil {
		ctrl.Log.Error(err, "failed to report GPUNode")
		os.Exit(1)
	}
}

func nodeStatus(hostname string) *tfv1.GPUNodeStatus {
	return &tfv1.GPUNodeStatus{
		KubernetesNodeName: hostname,
	}
}
