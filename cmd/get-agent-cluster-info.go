package cmd

import (
	"context"
	"flag"
	"fmt"
	"log"
	"smithy/internal/meta"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type getInfoCmd struct {
	metaCommand
	serverUrl      string
	credsPath      string
	smithyClusterId string
}

func getInfoCommand() subcommands.Command {
	return &getInfoCmd{
		metaCommand: metaCommand{
			name:     "get-info",
			synopsis: "get info",
			usage:    "get-info --id <smithy-cluster-id> -server <url> -creds </path/to/file>",
		},
	}
}

func (ec *getInfoCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&ec.smithyClusterId, "id", "", "smithy cluster id")
	f.StringVar(&ec.serverUrl, "server", nats.DefaultURL, "url of the command server")
	f.StringVar(&ec.credsPath, "creds", "", "path to creds file")
}

func (ec *getInfoCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	if ec.smithyClusterId == "" {
		f.Usage()
		return subcommands.ExitFailure
	}

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
	smithyClustersDataBucket, err := js.KeyValue(ctx, meta.SmithyClustersDataBucketName)
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
