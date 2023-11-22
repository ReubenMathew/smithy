package main

import (
	"context"
	"log"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
)

const (
	imageAmiId = "ami-0e83be366243f524a"
)

func main() {
	cfg, err := config.LoadDefaultConfig(context.TODO(), config.WithRegion("us-east-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}

	svc := ec2.NewFromConfig(cfg)
	res, err := svc.RunInstances(context.TODO(), &ec2.RunInstancesInput{
		TagSpecifications: []types.TagSpecification{
			{
				ResourceType: types.ResourceTypeInstance,
				Tags:         []types.Tag{
					{
						Key:   aws.String("Name"),
						Value: aws.String("Reuben's tester"),
					},
				},
			},
		},
		ImageId:      aws.String(imageAmiId),
        InstanceType: types.InstanceTypeT2Micro,
        MinCount:     aws.Int32(1),
        MaxCount:     aws.Int32(1),
	})
	if err != nil {
		log.Fatalf("unable to run instance, %v", err)
	}

	log.Printf("created instance %s", *res.Instances[0].InstanceId)

	svc.TerminateInstances(context.TODO(), &ec2.TerminateInstancesInput{
		//InstanceIds: ,
	})
}
