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
	serverUrl      string
	credsPath      string
}

func listAgentClustersCommand() subcommands.Command {
	return &listAgentClustersCmd{
		metaCommand: metaCommand{
			name:     "list-agent-clusters",
			synopsis: "list all smithy agent clusters",
			usage:    "list-agent-clusters -server <url> -creds </path/to/file>",
		},
	}
}

func (ec *listAgentClustersCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&ec.serverUrl, "server", nats.DefaultURL, "url of the command server")
	f.StringVar(&ec.credsPath, "creds", "", "path to creds file")
}

func (ec *listAgentClustersCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

	// --------------------
	// HACK: pull out later

	// default options
	opts := []nats.Option{}

	// if supplied a creds file, use it
	if ec.credsPath != "" {
		opts = append(opts, nats.UserCredentials(ec.credsPath))
	}

	// create NATS connection
	nc, err := nats.Connect(ec.serverUrl, opts...)
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

