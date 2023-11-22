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
	imageAmiId   = "ami-0e83be366243f524a"
	InstanceName = "reuben-nats-dev-cluster"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	// ec2 service
	ec2Svc := ec2.NewFromConfig(cfg)

	// create security group
	securityGroup, err := ec2Svc.CreateSecurityGroup(context.TODO(), &ec2.CreateSecurityGroupInput{
		Description: aws.String("temp nats cluster security group"),
		GroupName:   aws.String("temp-nats-cluster"),
	})
	if err != nil {
		log.Fatalf("unable to create security group, %v", err)
	}

	// create security group inbound traffic rules
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
						Value: aws.String(InstanceName),
					},
				},
			},
		},
		ImageId:      aws.String(imageAmiId),
		InstanceType: types.InstanceTypeT2Micro,
		MinCount:     instanceCount,
		MaxCount:     instanceCount,
	})
	if err != nil {
		log.Fatalf("unable to run instance, %v", err)
	}

	// get all instance ids
	instanceIds := []string{}
	for _, instance := range res.Instances {
		instanceIds = append(instanceIds, *instance.InstanceId)
	}
	createInstancesTimer := time.NewTimer(10 * time.Minute)
	createInstancesStatusTicker := time.NewTicker(5 * time.Second)
	expectedCompletedStates := len(res.Instances)
	// wait for instances to be ready
createInstancesWaitLoop:
	for {
		select {
		case <-createInstancesTimer.C:
			log.Fatalln("Creating instances took too long")
		case <-createInstancesStatusTicker.C:
			completedStates := 0
			// get instance statuses
			var describeInstanceStatusesOutput *ec2.DescribeInstanceStatusOutput
			describeInstanceStatusesOutput, err = ec2Svc.DescribeInstanceStatus(context.TODO(), &ec2.DescribeInstanceStatusInput{
				InstanceIds: instanceIds,
			})
			if err != nil {
				log.Fatalf("unable to describe instance statuses, %v", err)
			}
			for _, instance := range describeInstanceStatusesOutput.InstanceStatuses {
				currentState := instance.InstanceState.Name
				log.Printf("instance %s is in state %s", *instance.InstanceId, currentState)
				if currentState == types.InstanceStateNameRunning {
					completedStates++
				}
			}
			if completedStates == expectedCompletedStates {
				log.Println("All instances successfully created and running")
				break createInstancesWaitLoop
			}
		}
	}
	log.Printf("created instances %v", instanceIds)

	// wait
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
	// wait for instances to terminate
	terminateInstancesTimer := time.NewTimer(10 * time.Minute)
	terminateInstancesStatusTicker := time.NewTicker(5 * time.Second)
	expectedTerminatedStates := len(res.Instances)
terminateInstancesWaitLoop:
	for {
		select {
		case <-terminateInstancesTimer.C:
			log.Fatalln("Terminating instances took too long")
		case <-terminateInstancesStatusTicker.C:
			terminatedStates := 0
			// get instance statuses
			var describeInstanceStatusesOutput *ec2.DescribeInstanceStatusOutput
			describeInstanceStatusesOutput, err = ec2Svc.DescribeInstanceStatus(context.TODO(), &ec2.DescribeInstanceStatusInput{
				InstanceIds: instanceIds,
				// need to include this so that terminated instances are included in the response
				IncludeAllInstances: aws.Bool(true),
			})
			if err != nil {
				log.Fatalf("unable to describe instance statuses, %v", err)
			}
			for _, instance := range describeInstanceStatusesOutput.InstanceStatuses {
				currentState := instance.InstanceState.Name
				log.Printf("instance %s is in state %s", *instance.InstanceId, currentState)
				if currentState == types.InstanceStateNameTerminated {
					terminatedStates++
				}
			}
			if terminatedStates == expectedTerminatedStates {
				break terminateInstancesWaitLoop
			}
		}
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
