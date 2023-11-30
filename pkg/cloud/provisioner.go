package cloud

import "context"

type Provisioner interface {
	CreateComputeInstances(ctx context.Context, securityGroupName string, instanceCount int32) ([]ComputeInstance, error)
	TerminateComputeInstances(ctx context.Context, instanceIds []string) error
	CreateSecurityGroup(ctx context.Context, securityGroupName string) (securityGroupId string, err error)
	DeleteSecurityGroup(ctx context.Context, securityGroupName string) error
}
