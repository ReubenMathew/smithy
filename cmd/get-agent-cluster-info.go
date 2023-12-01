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

type getAgentClusterInfoCmd struct {
	metaCommand
	smithyClusterId string
}

func getAgentClusterInfoCommand() subcommands.Command {
	return &getAgentClusterInfoCmd{
		metaCommand: metaCommand{
			name:     "get-agent-cluster-info",
			synopsis: "get agent cluster info",
			usage:    "get-agent-cluster-info --id <smithy-cluster-id>",
		},
	}
}

func (ec *getAgentClusterInfoCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&ec.smithyClusterId, "id", "", "smithy cluster id")
}

func (ec *getAgentClusterInfoCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if ec.smithyClusterId == "" {
		f.Usage()
		return subcommands.ExitFailure
	}

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

	smithyClusterEntry, err := smithyClustersDataBucket.Get(ctx, ec.smithyClusterId)
	switch err {
	case jetstream.ErrKeyNotFound:
		log.Printf("smithy cluster-id: %s not found", ec.smithyClusterId)
		return subcommands.ExitFailure
	case nil:
		// continue
	default:
		log.Println(err.Error())
		return subcommands.ExitFailure
	}

	// print cluster info
	fmt.Println(string(smithyClusterEntry.Value()))

	return subcommands.ExitSuccess
}
