package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"smithy/pkg/aws"
	"smithy/pkg/cloud"
	"time"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

const (
	// TODO: put this in a better place
	InstanceTagNamePrefix        = "smithy-compute-node"
	SecurityGroupNamePrefix      = "smithy-sg"
	smithyClustersDataBucketName = "smithy-clusters"

	// TODO: make parameters
	smithyId = "default123"
)

type Deployer interface {
	CreateComputeInstances(ctx context.Context, securityGroupName string, instanceGroupName string, instanceCount int32) ([]cloud.ComputeInstance, error)
	CreateSecurityGroup(ctx context.Context, securityGroupName string) (securityGroupId string, err error)
}

type deployAgentsCmd struct {
	metaCommand
	numberOfAgents uint
	timeout        time.Duration
}

func deployAgentsCommand() subcommands.Command {
	return &deployAgentsCmd{
		metaCommand: metaCommand{
			name:     "deploy-agents",
			synopsis: "provision a set of cloud compute instances, each running an agent, within a security group",
			usage:    "deploy-agent -n <number> -t <duration>",
		},
	}
}

func (ec *deployAgentsCmd) SetFlags(f *flag.FlagSet) {
	f.UintVar(&ec.numberOfAgents, "n", 3, "number of agents")
	f.DurationVar(&ec.timeout, "t", 10*time.Minute, "timeout duration")
}

func (ec *deployAgentsCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

	// timeout context
	deployCtx, cancel := context.WithTimeout(ctx, ec.timeout)
	defer cancel()

	// TODO: make into a factory
	var (
		deployer Deployer
		err      error
	)
	deployer, err = aws.New(deployCtx)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// TODO: make fns for each value here
	var (
		securityGroupName = fmt.Sprintf("%s-%s", SecurityGroupNamePrefix, smithyId)
		instanceTagName   = fmt.Sprintf("%s-%s", InstanceTagNamePrefix, smithyId)
	)

	// STEPS
	// 1. does smithyId already exist ? exit : continue
	// 2. create security group and compute instances
	// 3. create smithyId->cloud.AgentCluster

	// HACK: pull out later
	// --------------------

	// create NATS connection
	// TODO: pass url and creds as parameters
	nc, err := nats.Connect(nats.DefaultURL)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	defer nc.Close()
	// create jetstream context
	js, err := jetstream.New(nc)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	// bind to smithy cluster bucket
	smithyClustersDataBucket, err := js.KeyValue(deployCtx, smithyClustersDataBucketName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	// check if smithyId already exists
	_, err = smithyClustersDataBucket.Get(deployCtx, smithyId)
	switch err {
	case nil:
		log.Printf("smithy cluster %s already exists, nothing to create", smithyId)
		return subcommands.ExitUsageError
	case jetstream.ErrKeyNotFound:
		// continue
	default:
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	// --------------------

	log.Printf("creating security group: %s", securityGroupName)
	securityGroupId, err := deployer.CreateSecurityGroup(deployCtx, securityGroupName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Printf("created security group %s", securityGroupId)

	log.Printf("creating %d compute instances", ec.numberOfAgents)
	computeInstances, err := deployer.CreateComputeInstances(deployCtx, securityGroupName, instanceTagName, int32(ec.numberOfAgents))
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Printf("created compute instances: %v", computeInstances)

	agentCluster := &cloud.AgentCluster{
		SecurityGroupName: securityGroupName,
		SecurityGroupId:   securityGroupId,
		ComputeInstances:  computeInstances,
	}

	// HACK: pull out later
	// --------------------

	// create entry in smithy cluster bucket
	if _, err = smithyClustersDataBucket.Create(deployCtx, smithyId, agentCluster.Bytes()); err != nil {
		log.Println(err.Error())	
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
