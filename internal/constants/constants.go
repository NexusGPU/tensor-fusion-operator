package constants

import "time"

const (
	// Domain is the domain prefix used for all tensor-fusion.ai related annotations and finalizers
	Domain = "tensor-fusion.ai"

	// Finalizer constants
	FinalizerSuffix = "finalizer"
	Finalizer       = Domain + "/" + FinalizerSuffix

	LabelKeyOwner = Domain + "/managed-by"

	PoolIdentifierAnnotationKey = Domain + "/pool"
	TensorFusionEnabledLabelKey = Domain + "/enabled"
	InitialGPUNodeSelector      = "nvidia.com/gpu.present=true"

	// Annotation key constants
	GpuPoolAnnotationKey = Domain + "/gpupool"
	// %s -> container_name
	TFLOPSRequestAnnotationFormat = Domain + "/tflops-request-%s"
	VRAMRequestAnnotationFormat   = Domain + "/vram-request-%s"
	TFLOPSLimitAnnotationFormat   = Domain + "/tflops-limit-%s"
	VRAMLimitAnnotationFormat     = Domain + "/vram-limit-%s"

	PendingRequeueDuration = time.Second * 3

	GetConnectionURLEnv    = "TENSOR_FUSION_OPERATOR_GET_CONNECTION_URL"
	ConnectionNameEnv      = "TENSOR_FUSION_CONNECTION_NAME"
	ConnectionNamespaceEnv = "TENSOR_FUSION_CONNECTION_NAMESPACE"

	WorkerPortEnv       = "TENSOR_FUSION_WORKER_PORT"
	NamespaceEnv        = "OPERATOR_NAMESPACE"
	NamespaceDefaultVal = "tensor-fusion"
)

const (
	ConditionStatusTypeReady           = "Ready"
	ConditionStatusTypeGPUScheduled    = "GPUScheduled"
	ConditionStatusTypeConnectionReady = "ConnectionReady"
	ConditionStatusTypeNodeProvisioned = "NodeProvisioned"
	ConditionStatusTypePoolReady       = "PoolReady"

	ConditionStatusTypeGPUPool               = "GPUPoolReady"
	ConditionStatusTypeTimeSeriesDatabase    = "TimeSeriesDatabaseReady"
	ConditionStatusTypeCloudVendorConnection = "CloudVendorConnectionReady"
)

const (
	PhaseUnknown    = "Unknown"
	PhasePending    = "Pending"
	PhaseUpdating   = "Updating"
	PhaseScheduling = "Scheduling"
	PhaseMigrating  = "Migrating"
	PhaseDestroying = "Destroying"

	PhaseRunning   = "Running"
	PhaseSucceeded = "Succeeded"
	PhaseFailed    = "Failed"
)
