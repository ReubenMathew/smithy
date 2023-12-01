package aws

import (
	"context"
	"fmt"
	"smithy/pkg/cloud"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	imageAmiId      = "ami-0e83be366243f524a"
	InstanceTagName = "temp-nats-compute-instance"
)

func (awsClient *AwsService) CreateComputeInstances(ctx context.Context, securityGroupName string, instanceCount int32) ([]cloud.ComputeInstance, error) {
	// create instances
	res, err := awsClient.svc.RunInstances(ctx, &ec2.RunInstancesInput{
		SecurityGroups: []string{securityGroupName},
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags: []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String(InstanceTagName),
					},
				},
			},
		},
		ImageId:      aws.String(imageAmiId),
		InstanceType: types.InstanceTypeT2Micro,
		MinCount:     aws.Int32(instanceCount),
		MaxCount:     aws.Int32(instanceCount),
		KeyName:      aws.String("reuben-dev"),
		// TODO: put in cloud-init script
		UserData: aws.String(""),
	})
	if err != nil {
		return nil, fmt.Errorf("unable to run instance(s), %v", err)
	}
	// get all instance ids
	instanceIds := []string{}
	for _, instance := range res.Instances {
		instanceIds = append(instanceIds, *instance.InstanceId)
	}

	// wait for instances to be in status ok
	if err = ec2.NewInstanceRunningWaiter(awsClient.svc).
		Wait(
			context.TODO(),
			&ec2.DescribeInstancesInput{
				InstanceIds: instanceIds,
			},
			// TODO: make parameter or constant
			10*time.Minute,
		); err != nil {
		return nil, fmt.Errorf("failed to wait for instances to be in status ok, %v", err)
	}

	ec2Instances := []cloud.ComputeInstance{}

	// get public dns names of instances
	describeInstancesResp, err := awsClient.svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe instances, %v", err)
	}
	for _, reservation := range describeInstancesResp.Reservations {
		for _, instance := range reservation.Instances {
			ec2Instances = append(ec2Instances, cloud.ComputeInstance{
				DnsName:    *instance.PublicDnsName,
				InstanceId: *instance.InstanceId,
			})
		}
	}
	return ec2Instances, nil
}

func (awsClient *AwsService) GetEc2InstanceIdsFromSecurityGroupName(ctx context.Context, securityGroupName string) ([]string, error) {
	describeInstancesResp, err := awsClient.svc.DescribeInstances(ctx, &ec2.DescribeInstancesInput{
		Filters: []types.Filter{
			{
				// filter by security group name
				Name:   aws.String("instance.group-name"),
				Values: []string{securityGroupName},
			},
		},
	})
	if err != nil {
		return nil, fmt.Errorf("unable to describe instances, %v", err)
	}
	instanceIds := []string{}
	for _, reservation := range describeInstancesResp.Reservations {
		for _, instance := range reservation.Instances {
			instanceIds = append(instanceIds, *instance.InstanceId)
		}
	}
	return instanceIds, nil
}

func (awsClient *AwsService) TerminateComputeInstances(ctx context.Context, instanceIds []string) error {
	_, err := awsClient.svc.TerminateInstances(ctx, &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		return fmt.Errorf("unable to terminate instances, %v", err)
	}

	// wait for instances to be terminated
	if err = ec2.NewInstanceTerminatedWaiter(awsClient.svc).
		Wait(
			ctx, &ec2.DescribeInstancesInput{
				InstanceIds: instanceIds,
			},
			// TODO: make parameter or constant
			10*time.Minute,
		); err != nil {
		return fmt.Errorf("failed to wait for instances to be terminated, %v", err)
	}
	return nil
}
