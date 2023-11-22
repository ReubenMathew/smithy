package main

import (
	"context"
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
	svc := ec2.NewFromConfig(cfg)

	// num of instances to create
	instanceCount := aws.Int32(3)

	// create instances
	res, err := svc.RunInstances(context.TODO(), &ec2.RunInstancesInput{
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

	instanceIds := []string{}
	for _, instance := range res.Instances {
		instanceIds = append(instanceIds, *instance.InstanceId)
	}
	log.Printf("created instances %v", instanceIds)

	createInstancesTimer := time.NewTimer(10 * time.Minute)
	statusCheckTicker := time.NewTicker(5 * time.Second)
	expectedCompletedStates := len(res.Instances)
	// wait for instances to be ready
	createInstancesWaitLoop: for {
		select {
		case <-createInstancesTimer.C:
			log.Fatalln("Creating instances took too long")
		case <-statusCheckTicker.C:
			completedStates := 0
			for _, instance := range res.Instances {
				currentState := instance.State.Name
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

	// wait
	time.Sleep(30 * time.Second)
	log.Println("Waiting for 30s")

	// terminate all created instances
	_, err = svc.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
		InstanceIds: instanceIds,
	})
	if err != nil {
		log.Fatalf("unable to terminate instances: %v", err)
	}
	// wait for instances to terminate
	terminateInstancesTimer := time.NewTimer(10 * time.Minute)
	terminateInstancesWaitLoop: for {
		select {
		case <-terminateInstancesTimer.C:
			log.Fatalln("Terminating instances took too long")
		case <-statusCheckTicker.C:
			completedStates := 0
			for _, instance := range res.Instances {
				currentState := instance.State.Name
				if currentState == types.InstanceStateNameTerminated {
					completedStates++
				}
			}
			if completedStates == expectedCompletedStates {
				log.Println("All instances successfully created and running")
				break terminateInstancesWaitLoop
			}
		}
	}
	log.Printf("terminated instances %v", instanceIds)

}
