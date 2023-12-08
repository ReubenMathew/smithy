package cmd

import (
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"log"
	"smithy/internal/meta"
	"smithy/pkg/aws"
	"smithy/pkg/cloud"
	"text/template"
	"time"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Deployer interface {
	CreateComputeInstances(ctx context.Context, securityGroupName string, instanceGroupName string, instanceCount int32, credsPath string, clusterId string) ([]cloud.ComputeInstance, error)
	CreateSecurityGroup(ctx context.Context, securityGroupName string) (securityGroupId string, err error)
}

type deployAgentsCmd struct {
	metaCommand
	numberOfAgents uint
	serverUrl      string
	credsPath      string
	clusterId      string
	timeout        time.Duration
}

var (
	//go:embed server.conf.tmpl
	serverConfTemplate string
)

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
	f.StringVar(&dac.clusterId, "id", "default", "smithy cluster id")
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
		securityGroupName = fmt.Sprintf("%s-%s", meta.SecurityGroupNamePrefix, dac.clusterId)
		instanceTagName   = fmt.Sprintf("%s-%s", meta.InstanceTagNamePrefix, dac.clusterId)
	)

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
	smithyClustersDataBucket, err := js.KeyValue(deployCtx, meta.SmithyClustersDataBucketName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// check if clusterId already exists
	_, err = smithyClustersDataBucket.Get(deployCtx, dac.clusterId)
	switch err {
	case nil:
		log.Printf("smithy cluster %s already exists, nothing to create", dac.clusterId)
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
	computeInstances, err := deployer.CreateComputeInstances(deployCtx, securityGroupName, instanceTagName, int32(dac.numberOfAgents), dac.credsPath, dac.clusterId)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	for _, ci := range computeInstances {
		log.Printf("created compute instance %s - DnsName: %s, InstanceId: %s, PrivateIp: %s, PublicIp: %s", instanceTagName, ci.DnsName, ci.InstanceId, ci.PrivateIp, ci.PublicIp)
	}

	//  print NATS urls
	fmt.Println("nats urls:")
	for _, ci := range computeInstances {
		fmt.Printf("nats://%s:4222,", ci.DnsName)
	}
	// get rid of trailing comma and add newline
	fmt.Printf("\b\n")

	// get cluster urls
	clusterUrlsString := ""
	for _, ci := range computeInstances {
		clusterUrlsString += fmt.Sprintf("nats://%s:6222\n", ci.DnsName)
	}

	agentCluster := &cloud.AgentCluster{
		SecurityGroupName: securityGroupName,
		SecurityGroupId:   securityGroupId,
		ComputeInstances:  computeInstances,
	}

	// create entry in smithy cluster bucket
	if _, err = smithyClustersDataBucket.Create(deployCtx, dac.clusterId, agentCluster.Bytes()); err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	fmt.Printf("created smithy cluster entry %s\n", dac.clusterId)

	// fill out server.conf template
	configData := map[string]string{
		"ClusterName":         dac.clusterId,
		"ClusterRoutesString": clusterUrlsString,
	}

	tmpl := template.Must(template.New("server.conf").Parse(serverConfTemplate))

	buffer := new(bytes.Buffer)
	err = tmpl.Execute(buffer, configData)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	jsObj, err := nc.JetStream()
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	obj, err := jsObj.ObjectStore(meta.SmithyClustersObjStoreName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	configFileName := fmt.Sprintf("%s-server.conf", dac.clusterId)
	_, err = obj.PutBytes(configFileName, buffer.Bytes())
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
