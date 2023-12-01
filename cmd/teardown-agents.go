package cmd

import (
	"context"
	"flag"
	"log"
	"smithy/pkg/aws"
	"time"

	"github.com/google/subcommands"
)

type Teardowner interface {
	DeleteSecurityGroup(ctx context.Context, securityGroupName string) error
	TerminateComputeInstances(ctx context.Context, instanceIds []string) error
}

type teardownAgentsCmd struct {
	metaCommand
	timeout time.Duration
}

func teardownAgentsCommand() subcommands.Command {
	return &teardownAgentsCmd{
		metaCommand: metaCommand{
			name:     "teardown-agents",
			synopsis: "terminate all agents within a security group",
			usage:    "teardown-agents -t <duration>",
		},
	}
}

func (ec *teardownAgentsCmd) SetFlags(f *flag.FlagSet) {
	f.DurationVar(&ec.timeout, "t", 10*time.Minute, "timeout duration")
}

func (ec *teardownAgentsCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

	// timeout context
	teardownCtx, cancel := context.WithTimeout(ctx, ec.timeout)
	defer cancel()

	var (
		teardowner Teardowner
		err        error
	)
	teardowner, err = aws.New(teardownCtx)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// HACK: remove later
	// ------------------
	awsSvc, err := aws.New(teardownCtx)
	if err != nil {
		panic(err)
	}
	instanceIds, err := awsSvc.GetEc2InstanceIdsFromSecurityGroupName(ctx, SecurityGroupName)
	if err != nil {
		panic(err)
	}
	// ------------------

	// terminate compute instances
	log.Println("terminating down compute instances")
	if err = teardowner.TerminateComputeInstances(teardownCtx, instanceIds); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Println("terminated compute instances")

	// delete security group
	log.Println("deleting security group")
	if err = teardowner.DeleteSecurityGroup(teardownCtx, SecurityGroupName); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Println("deleted security group")

	return subcommands.ExitSuccess
}
