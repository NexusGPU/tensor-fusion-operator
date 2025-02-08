package alibaba

import (
	"context"
	"fmt"
	"time"

	tfv1 "github.com/NexusGPU/tensor-fusion-operator/api/v1"
	"github.com/aliyun/alibaba-cloud-sdk-go/services/ecs"

	types "github.com/NexusGPU/tensor-fusion-operator/internal/cloudprovider/types"

	"github.com/NexusGPU/tensor-fusion-operator/internal/utils"
)

type AliyunGPUNodeProvider struct {
	client *ecs.Client
}

func NewAliyunGPUNodeProvider(config tfv1.ComputingVendorConfig) (AliyunGPUNodeProvider, error) {

	var provider AliyunGPUNodeProvider

	if config.AuthType != tfv1.AuthTypeAccessKey {
		return provider, fmt.Errorf("unsupported auth type for alibaba cloud: %s", config.AuthType)
	}
	ak, err := utils.GetAccessKeyOrSecretFromEnvOrPath(
		config.Params.AccessKeyEnvVar,
		config.Params.AccessKeyPath,
	)
	if err != nil {
		return provider, err
	}
	sk, err := utils.GetAccessKeyOrSecretFromEnvOrPath(
		config.Params.SecretKeyEnvVar,
		config.Params.SecretKeyPath,
	)
	if err != nil {
		return provider, err
	}

	client, err := ecs.NewClientWithAccessKey(config.Params.DefaultRegion, ak, sk)
	if err != nil {
		return provider, fmt.Errorf("failed to create ECS client: %w", err)
	}
	provider.client = client
	return provider, nil
}

func (p AliyunGPUNodeProvider) TestConnection() error {
	request := ecs.CreateDescribeRegionsRequest()
	response, err := p.client.DescribeRegions(request)
	if err != nil {
		return fmt.Errorf("Can not connect to Aliyun ECS API: %v", err)
	}
	fmt.Printf("Successfully connected to Aliyun ECS. Available regions: %v\n", response.Regions.Region)
	return nil
}

func (p AliyunGPUNodeProvider) CreateNode(ctx context.Context, param *types.NodeCreationParam) (*types.GPUNodeStatus, error) {
	nodeClass := param.NodeClass.Spec
	request := ecs.CreateRunInstancesRequest()
	request.LaunchTemplateId = nodeClass.LaunchTemplate.ID

	if len(nodeClass.OSImageSelectorTerms) > 0 {
		// TODO: should support other query types not only ID
		request.ImageId = nodeClass.OSImageSelectorTerms[0].ID
	}
	request.InstanceType = param.InstanceType
	request.InstanceName = param.NodeName
	request.Amount = "1"

	tag := []ecs.RunInstancesTag{
		{Key: "managed-by", Value: "tensor-fusion.ai"},
		{Key: "tensor-fusion.ai/node-name", Value: param.NodeName},
		{Key: "tensor-fusion.ai/node-class", Value: param.NodeClass.Name},
	}
	for k, v := range nodeClass.Tags {
		tag = append(tag, ecs.RunInstancesTag{
			Key:   k,
			Value: v,
		})
	}
	request.Tag = &tag

	// TODO: handle other extra params which contains OS constraints, charging type, security group, volume type, user data, vpc/subnet etc.

	response, err := p.client.RunInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to create instance: %w", err)
	}
	if !response.IsSuccess() {
		return nil, fmt.Errorf("instance creation failed: %s", response.String())
	}

	// TODO, SHOULD use request id to check instance status util created
	return &types.GPUNodeStatus{
		InstanceID: response.RequestId,
		CreatedAt:  time.Now(),
	}, nil
}

func (p AliyunGPUNodeProvider) TerminateNode(ctx context.Context, param *types.NodeIdentityParam) error {
	request := ecs.CreateDeleteInstanceRequest()
	request.InstanceId = param.InstanceID
	response, err := p.client.DeleteInstance(request)
	if err != nil {
		return fmt.Errorf("failed to terminate instance: %w", err)
	}
	if !response.IsSuccess() {
		return fmt.Errorf("instance termination failed: %s", response.String())
	}
	return nil
}

func (p AliyunGPUNodeProvider) GetNodeStatus(ctx context.Context, param *types.NodeIdentityParam) (*types.GPUNodeStatus, error) {
	request := ecs.CreateDescribeInstancesRequest()
	request.InstanceIds = fmt.Sprintf("[\"%s\"]", param.InstanceID)
	response, err := p.client.DescribeInstances(request)
	if err != nil {
		return nil, fmt.Errorf("failed to describe instance: %w", err)
	}
	if len(response.Instances.Instance) == 0 {
		return nil, fmt.Errorf("instance not found")
	}
	instance := response.Instances.Instance[0]
	createTime, _ := time.Parse(time.RFC3339, instance.CreationTime)

	privateIP := ""
	publicIP := ""
	if len(instance.PublicIpAddress.IpAddress) > 0 {
		publicIP = instance.PublicIpAddress.IpAddress[0]
	}
	if len(instance.VpcAttributes.PrivateIpAddress.IpAddress) > 0 {
		privateIP = instance.VpcAttributes.PrivateIpAddress.IpAddress[0]
	}

	status := &types.GPUNodeStatus{
		InstanceID: instance.InstanceId,
		CreatedAt:  createTime,
		PrivateIP:  privateIP,
		PublicIP:   publicIP,
	}
	return status, nil
}
