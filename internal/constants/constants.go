package constants

import "time"

const (
	// TensorFusionDomain is the domain prefix used for all tensor-fusion.ai related annotations and finalizers
	TensorFusionDomain = "tensor-fusion.ai"

	// Finalizer constants
	TensorFusionFinalizerSuffix = "finalizer"
	TensorFusionFinalizer       = TensorFusionDomain + "/" + TensorFusionFinalizerSuffix

	// Annotation key constants
	EnableContainerAnnotationFormat = TensorFusionDomain + "/enable-%s"
	TFLOPSContainerAnnotationFormat = TensorFusionDomain + "/tflops-%s"
	VRAMContainerAnnotationFormat   = TensorFusionDomain + "/vram-%s"

	PendingRequeueDuration = time.Second * 3

	GetConnectionURLEnv    = "TENSOR_FUSION_OPERATOR_GET_CONNECTION_URL"
	ConnectionNameEnv      = "TENSOR_FUSION_CONNECTION_NAME"
	ConnectionNamespaceEnv = "TENSOR_FUSION_CONNECTION_NAMESPACE"
)
