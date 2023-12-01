package cmd

import (
	"context"
	"flag"
	"log"
	"smithy/pkg/aws"
	"smithy/pkg/cloud"
	"time"

	"github.com/google/subcommands"
)

// HACK: parameterize all of these
const (
	ImageAmiId        = "ami-0e83be366243f524a"
	InstanceTagName   = "reuben-nats-dev-cluster"
	SecurityGroupName = "temp-nats-cluster"
)

type Deployer interface {
	CreateComputeInstances(ctx context.Context, securityGroupName string, instanceCount int32) ([]cloud.ComputeInstance, error)
	CreateSecurityGroup(ctx context.Context, securityGroupName string) (securityGroupId string, err error)
}

type deployAgentsCmd struct {
	metaCommand
	numberOfAgents uint
	timeout time.Duration
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
	f.DurationVar(&ec.timeout, "t", 10 * time.Minute, "timeout duration")
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

	log.Printf("creating security group: %s", SecurityGroupName)
	securityGroupId, err := deployer.CreateSecurityGroup(deployCtx, SecurityGroupName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Printf("created security group %s", securityGroupId)

	log.Printf("creating %d compute instances", ec.numberOfAgents)
	computeInstances, err := deployer.CreateComputeInstances(deployCtx, SecurityGroupName, int32(ec.numberOfAgents))
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Printf("created compute instances: %v", computeInstances)

	return subcommands.ExitSuccess
}
