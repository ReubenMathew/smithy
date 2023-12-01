package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type listAgentClustersCmd struct {
	metaCommand
}

func listAgentClustersCommand() subcommands.Command {
	return &listAgentClustersCmd{
		metaCommand: metaCommand{
			name:     "list-agent-clusters",
			synopsis: "list all smithy agent clusters",
			usage:    "list-agent-clusters",
		},
	}
}

func (ec *listAgentClustersCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

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
	smithyClustersDataBucket, err := js.KeyValue(ctx, smithyClustersDataBucketName)
	if err != nil {
		log.Println(err.Error())
		return subcommands.ExitFailure
	}
	// --------------------

	smithyClusterIds, err := smithyClustersDataBucket.Keys(ctx)
	switch err {
	case jetstream.ErrNoKeysFound:
		fmt.Println("no smithy clusters found")
		return subcommands.ExitSuccess
	case nil:
		// continue
	default:
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// print cluster ids
	for _, smithyClusterId := range smithyClusterIds {
		fmt.Println(smithyClusterId)
	}

	return subcommands.ExitSuccess
}

