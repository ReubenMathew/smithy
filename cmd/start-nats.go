package cmd

import (
	"context"
	"flag"
	"fmt"
	"smithy/pkg/agent"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
)

type startNatsCmd struct {
	metaCommand
	serverUrl string
	credsPath string
	clusterId string
}

func startNatsCommand() subcommands.Command {
	return &startAgentCmd{
		metaCommand: metaCommand{
			name:     "start-nats",
			synopsis: "Starts nats-server process for a cluster",
			usage:    "start-nats -server <url> -creds <path/to/file> -cluster <string>",
		},
	}
}

func (c *startNatsCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.serverUrl, "server", nats.DefaultURL, "Server URL")
	f.StringVar(&c.credsPath, "creds", "", "Credentials file path")
	f.StringVar(&c.clusterId, "cluster", "default", "Smithy instance id")
}

func (c *startNatsCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

	opts := []nats.Option{}

	if c.credsPath != "" {
		opts = append(opts, nats.UserCredentials(c.credsPath))
	}

	fmt.Println("Connecting to NATS server")

	nc, err := nats.Connect(c.serverUrl, opts...)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	fmt.Println("Connected to NATS server")
	if err = nc.Publish(fmt.Sprintf("%s.%s", agent.SmithyAgentsStreamName, c.clusterId), []byte("start")); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	return subcommands.ExitSuccess
}
