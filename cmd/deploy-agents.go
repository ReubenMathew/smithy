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
	smithyClustersDataBucketName = "smithy-agent-clusters"
)

type Deployer interface {
	CreateComputeInstances(ctx context.Context, securityGroupName string, instanceGroupName string, instanceCount int32) ([]cloud.ComputeInstance, error)
	CreateSecurityGroup(ctx context.Context, securityGroupName string) (securityGroupId string, err error)
}

type deployAgentsCmd struct {
	metaCommand
	numberOfAgents uint
	serverUrl      string
	credsPath      string
	smithyId       string
	timeout        time.Duration
}

func deployAgentsCommand() subcommands.Command {
	return &deployAgentsCmd{
		metaCommand: metaCommand{
			name:     "deploy-agents",
			synopsis: "provision a set agents, each within a compute instance",
			usage:    "deploy-agent -id <string> -n <int> -t <duration> -server <url> -creds </path/to/file>",
		},
	}
}

func (dac *deployAgentsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&dac.smithyId, "id", "default", "smithy cluster id")
	f.UintVar(&dac.numberOfAgents, "n", 3, "number of agents")
	f.StringVar(&dac.serverUrl, "server", nats.DefaultURL, "url to command server")
	f.StringVar(&dac.credsPath, "creds", "", "path to creds file")
	f.DurationVar(&dac.timeout, "t", 10*time.Minute, "timeout duration for all context operations")
}

func (dac *deployAgentsCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

	// timeout context
	deployCtx, cancel := context.WithTimeout(ctx, dac.timeout)
	defer cancel()

	// TODO: make fns for each value here
	var (
		securityGroupName = fmt.Sprintf("%s-%s", SecurityGroupNamePrefix, dac.smithyId)
		instanceTagName   = fmt.Sprintf("%s-%s", InstanceTagNamePrefix, dac.smithyId)
	)

	// --------------------
	// HACK: pull out later

	// default options
	opts := []nats.Option{}

	// if supplied a creds file, use it
	if dac.credsPath != "" {
		opts = append(opts, nats.UserCredentials(dac.credsPath))
	}

	// create NATS connection
	nc, err := nats.Connect(dac.serverUrl, opts...)
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
	// --------------------

	// check if smithyId already exists
	_, err = smithyClustersDataBucket.Get(deployCtx, dac.smithyId)
	switch err {
	case nil:
		log.Printf("smithy cluster %s already exists, nothing to create", dac.smithyId)
		return subcommands.ExitUsageError
	case jetstream.ErrKeyNotFound:
		// continue
	default:
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// create deployer service
	// TODO: make into a (cloud-provider) factory
	var deployer Deployer
	deployer, err = aws.New(deployCtx)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	log.Printf("creating security group: %s", securityGroupName)
	securityGroupId, err := deployer.CreateSecurityGroup(deployCtx, securityGroupName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	log.Printf("created security group %s: %s", securityGroupName, securityGroupId)

	log.Printf("creating %d compute instances", dac.numberOfAgents)
	computeInstances, err := deployer.CreateComputeInstances(deployCtx, securityGroupName, instanceTagName, int32(dac.numberOfAgents))
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	for _, ci := range computeInstances {
		log.Printf("created compute instance %s: %s", ci.InstanceId, ci.DnsName)
	}

	agentCluster := &cloud.AgentCluster{
		SecurityGroupName: securityGroupName,
		SecurityGroupId:   securityGroupId,
		ComputeInstances:  computeInstances,
	}

	// create entry in smithy cluster bucket
	if _, err = smithyClustersDataBucket.Create(deployCtx, dac.smithyId, agentCluster.Bytes()); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
