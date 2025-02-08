package cloudprovider

import (
	"fmt"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/types"

	aliyun "github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/alibaba"
	aws "github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/aws"
)

func GetProvider(config tfv1.ComputingVendorConfig) (*types.GPUNodeProvider, error) {
	var err error
	var provider types.GPUNodeProvider
	switch config.Type {
	case "aws":
		provider, err = aws.NewAWSGPUNodeProvider(config)
	case "aliyun":
		provider, err = aliyun.NewAliyunGPUNodeProvider(config)
	default:
		return nil, fmt.Errorf("unsupported cloud provider: %s", config.Type)
	}
	return &provider, err
}
