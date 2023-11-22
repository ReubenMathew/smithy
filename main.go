package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	imageAmiId        = "ami-0e83be366243f524a"
	InstanceTagName   = "reuben-nats-dev-cluster"
	securityGroupName = "temp-nats-cluster"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// ec2 service
	ec2Svc := ec2.NewFromConfig(cfg)

	// delete initial security group
	// HACK: should remove this
	_, err = ec2Svc.DeleteSecurityGroup(context.TODO(), &ec2.DeleteSecurityGroupInput{
		GroupName: aws.String(securityGroupName),
	})
	if err != nil {
		log.Printf("unable to delete security group, %v", err)
	} else {
		log.Printf("deleted security group %s", securityGroupName)
	}

	// create security group
	securityGroup, err := ec2Svc.CreateSecurityGroup(context.TODO(), &ec2.CreateSecurityGroupInput{
		Description: aws.String("temp nats cluster security group"),
		GroupName:   aws.String(securityGroupName),
	})
	if err != nil {
		log.Fatalf("unable to create security group, %v", err)
	}

	// create security group traffic rules
	// egress rule for all outbound traffic is created by default
	_, err = ec2Svc.AuthorizeSecurityGroupIngress(context.TODO(), &ec2.AuthorizeSecurityGroupIngressInput{
		GroupId:    securityGroup.GroupId,
		CidrIp:     aws.String("0.0.0.0/0"),
		IpProtocol: aws.String("tcp"),
		FromPort:   aws.Int32(22),
		ToPort:     aws.Int32(22),
	})
	if err != nil {
		log.Fatalf("unable to authorize security group ingress, %v", err)
	}
	log.Printf("created security group %s", *securityGroup.GroupId)

	// HACK: make parameter later
	instanceCount := aws.Int32(3)
	// create instances
	res, err := ec2Svc.RunInstances(context.TODO(), &ec2.RunInstancesInput{
		// TODO: add security group
		SecurityGroupIds: []string{*securityGroup.GroupId},
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
		MinCount:     instanceCount,
		MaxCount:     instanceCount,
		KeyName:      aws.String("reuben-dev"),
	})
	if err != nil {
		log.Fatalf("unable to run instance, %v", err)
	}

	// get all instance ids
	instanceIds := []string{}
	for _, instance := range res.Instances {
		instanceIds = append(instanceIds, *instance.InstanceId)
	}

	// wait for instances to be in status ok
	if err = ec2.NewInstanceRunningWaiter(ec2Svc).
		Wait(
			context.TODO(),
			&ec2.DescribeInstancesInput{
				InstanceIds: instanceIds,
			},
			10*time.Minute,
		); err != nil {
		log.Fatalf("failed to wait for instances to be in status ok, %v", err)
	}
	log.Printf("created instances %v", instanceIds)

	// get public dns names of instances
	describeInstancesResp, err := ec2Svc.DescribeInstances(context.TODO(), &ec2.DescribeInstancesInput{	
		InstanceIds: instanceIds,
	})
	if err != nil {
		log.Fatalf("unable to describe instances, %v", err)
	}
	for _, reservation := range describeInstancesResp.Reservations {
		log.Printf("Reservation %s", *reservation.ReservationId)
		for _, instance := range reservation.Instances {
			log.Printf("Instance %s - %s", *instance.InstanceId, *instance.PublicDnsName)
		}
	}

	// block before terminating instances
	log.Println("Enter something here to terminate instances")
	var input string
	_, err = fmt.Scanln(&input)
	if err != nil {
		log.Fatalf("unable to read input, %v", err)
	}

	// terminate all created instances
	log.Printf("terminating instances %v", instanceIds)
	_, err = ec2Svc.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		log.Fatalf("unable to terminate instances: %v", err)
	}

	// wait for instances to be terminated
	if err = ec2.NewInstanceTerminatedWaiter(ec2Svc).
		Wait(
			context.TODO(), &ec2.DescribeInstancesInput{
				InstanceIds: instanceIds,
			},
			10*time.Minute,
		); err != nil {
		log.Fatalf("failed to wait for instances to be terminated, %v", err)
	}
	log.Printf("terminated instances %v", instanceIds)

	// delete security group
	_, err = ec2Svc.DeleteSecurityGroup(context.TODO(), &ec2.DeleteSecurityGroupInput{
		GroupId: securityGroup.GroupId,
	})
	if err != nil {
		log.Fatalf("unable to delete security group, %v", err)
	}
	log.Printf("deleted security group %s", *securityGroup.GroupId)
}
