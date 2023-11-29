package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ec2"
	"github.com/aws/aws-sdk-go-v2/service/ec2/types"
	"github.com/nats-io/nats.go"
)

type Options struct {
	Command          string
	CommandServerUrl string
	Verbose          bool
}

func DefaultOptions() Options {
	return Options{
		CommandServerUrl: nats.DefaultURL,
		Verbose:          false,
	}
}
const (
	// TODO: make as a parameter
	securityGroupName = "temp-nats-cluster-security-group"
)

func main() {

	opts := DefaultOptions()

	// read flags and positional arguments to override default options
	flag.StringVar(&opts.CommandServerUrl, "s", opts.CommandServerUrl, "The NATS server URLs (separated by comma)")
	flag.BoolVar(&opts.Verbose, "v", opts.Verbose, "Enable verbose mode")
	flag.StringVar(&opts.Command, "c", opts.Command, "Command to run")
	flag.Parse()

	// HACK: make a proper context
	ctx := context.TODO()

	// load config and establish a ec2 service
	cfg, err := config.LoadDefaultConfig(ctx, config.WithRegion("us-east-2"))
	if err != nil {
		log.Fatalf("unable to load SDK config, %v", err)
	}
	svc := ec2.NewFromConfig(cfg)

	// HACK: make proper fn
	createSecurityGroup := func() {
		log.Printf("Attempting to create security group %s...", securityGroupName)
		securityGroupId, err := CreateSecurityGroup(ctx, svc)
		if err != nil {
			log.Fatalf("unable to create security group, %v", err)
		}
		log.Printf("created security group %s", *securityGroupId)
	}

	// HACK: make proper fn
	deleteSecurityGroup := func() {
		log.Printf("Attempting to delete security group %s...", securityGroupName)
		err := DeleteSecurityGroup(ctx, svc)
		if err != nil {
			log.Fatalf("unable to delete security group, %v", err)
		}
		log.Println("deleted security group")
	}

	// HACK: make proper subcommand later
	switch opts.Command {
	case "deploy":
		createSecurityGroup()
	case "clean":
		deleteSecurityGroup()
	default:
		log.Fatalf("unknown command: %s", opts.Command)
	}

}

func DeleteSecurityGroup(ctx context.Context, svc *ec2.Client) error {
	_, err := svc.DeleteSecurityGroup(ctx, &ec2.DeleteSecurityGroupInput{
		GroupName: aws.String(securityGroupName),
	})	
	if err != nil {
		return fmt.Errorf("unable to delete security group, %v", err)
	}
	return nil
}

func CreateSecurityGroup(ctx context.Context, svc *ec2.Client) (securityGroupId *string, err error) {

	// create security group
	securityGroup, err := svc.CreateSecurityGroup(ctx, &ec2.CreateSecurityGroupInput{
		Description: aws.String("temp nats cluster security group"),
		GroupName:   aws.String(securityGroupName),
	})
	if err != nil {
		return
	}
	securityGroupId = securityGroup.GroupId

	// TODO: make as a parameter
	waitTime := 5 * time.Minute

	// wait for security group to be created
	if err = ec2.NewSecurityGroupExistsWaiter(svc).Wait(ctx, &ec2.DescribeSecurityGroupsInput{
		GroupIds: []string{*securityGroupId},
	}, waitTime); err != nil {
		err = fmt.Errorf("security group %s never became available after %f minutes: %v", *securityGroupId, waitTime.Minutes(), err)
		return
	}

	// create security group traffic rules
	// egress rule for all outbound traffic is created by default
	_, err = svc.AuthorizeSecurityGroupIngress(ctx, &ec2.AuthorizeSecurityGroupIngressInput{
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
