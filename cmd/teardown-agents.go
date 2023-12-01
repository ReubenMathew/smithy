package cmd

import (
	"context"
	"flag"
	"log"
	"smithy/pkg/aws"
	"smithy/pkg/cloud"
	"time"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Terminator interface {
	DeleteSecurityGroup(ctx context.Context, securityGroupId string) error
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

	// --------------------
	// HACK: pull out later

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
	smithyClustersDataBucket, err := js.KeyValue(teardownCtx, smithyClustersDataBucketName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	// check if smithyId already exists
	agentClusterEntry, err := smithyClustersDataBucket.Get(teardownCtx, smithyId)
	switch err {
	case nil:
		// continue
	case jetstream.ErrKeyNotFound:
		log.Printf("smithy cluster id: %s does not exist", smithyId)
		return subcommands.ExitFailure
	default:
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	agentCluster, err := cloud.LoadAgentCluster(agentClusterEntry.Value())
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	// --------------------

	var teardowner Terminator
	teardowner, err = aws.New(teardownCtx)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// get instance ids
	instanceIds := []string{}
	for _, instance := range agentCluster.ComputeInstances {
		instanceIds = append(instanceIds, instance.InstanceId)
	}

	// terminate compute instances
	log.Printf("terminating compute instances: %v", instanceIds)
	if err = teardowner.TerminateComputeInstances(teardownCtx, instanceIds); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Println("terminated compute instances")

	// delete security group
	log.Printf("deleting security group %s: %s", agentCluster.SecurityGroupName, agentCluster.SecurityGroupId)
	if err = teardowner.DeleteSecurityGroup(teardownCtx, agentCluster.SecurityGroupId); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Println("deleted security group")

	// remove entry from bucket
	if err = smithyClustersDataBucket.Delete(ctx, smithyId); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
