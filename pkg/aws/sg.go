package aws

import (
	"context"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

func (awsClient *AwsService) DeleteSecurityGroup(ctx context.Context, securityGroupId string) error {
	_, err := awsClient.svc.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupId:   aws.String(securityGroupId),
	})
	if err != nil {
		return fmt.Errorf("unable to delete security group, %v", err)
	}
	return nil
}

func (awsClient *AwsService) CreateSecurityGroup(ctx context.Context, securityGroupName string) (securityGroupId string, err error) {

	// create security group
	securityGroup, err := awsClient.svc.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		Description: aws.String("temp nats cluster security group"),
		GroupName:   aws.String(securityGroupName),
	})
	if err != nil {
		return
	}
	securityGroupId = *securityGroup.GroupId

	// TODO: make as a parameter
	waitTime := 5 * time.Minute

	// wait for security group to be created
	if err = ec2.NewSecurityGroupExistsWaiter(awsClient.svc).Wait(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{securityGroupId},
	}, waitTime); err != nil {
		err = fmt.Errorf("security group %s never became available after %f minutes: %v", securityGroupId, waitTime.Minutes(), err)
		return
	}

	// create security group traffic rules
	// egress rule for all outbound traffic is created by default
	_, err = awsClient.svc.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId: securityGroup.GroupId,
		IpPermissions: []types.IpPermission{
			{
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("NATS port for inbound traffic"),
					},
				},
				FromPort:   aws.Int32(4222),
				ToPort:     aws.Int32(4222),
				IpProtocol: aws.String("tcp"),
			},
			{
				IpRanges: []types.IpRange{
					{
						CidrIp:      aws.String("0.0.0.0/0"),
						Description: aws.String("Inbound SSH port from any machine"),
					},
				},
				IpProtocol: aws.String("tcp"),
				FromPort:   aws.Int32(22),
				ToPort:     aws.Int32(22),
			},
		},
	})
	if err != nil {
		err = fmt.Errorf("unable to authorize security group ingress, %v", err)
		return
	}
	return
}
