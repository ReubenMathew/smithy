package aws

import (
	"context"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
)

type AwsService struct {
	svc *ec2.Client
}

func New(ctx context.Context) (*AwsService, error) {

	region := config.WithRegion("us-east-2")

	cfg, err := config.LoadDefaultConfig(ctx, region)
	if err != nil {
		return nil, fmt.Errorf("unable to load SDK config, %v", err)
	}

	// ec2 service
	svc := ec2.NewFromConfig(cfg)
	
	return &AwsService{
		svc: svc,
	}, nil
}
