package cmd

import (
	"context"
	"flag"
	"fmt"
	"smithy/pkg/agent"

	"github.com/google/subcommands"
	"github.com/nats-io/nats.go"
)

type startAgentCmd struct {
	metaCommand
	serverUrl string
	credsPath string
	smithyId   string
}

func startAgentCommand() subcommands.Command {
	return &startAgentCmd{
		metaCommand: metaCommand{
			name:     "start-agent",
			synopsis: "Starts agent process",
			usage:    "start-agent -server <url> -creds <path/to/file> -id <string>",
		},
	}
}

func (c *startAgentCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.serverUrl, "server", nats.DefaultURL, "Server URL")
	f.StringVar(&c.credsPath, "creds", "", "Credentials file path")
	f.StringVar(&c.smithyId, "id", "", "Smithy cluster id")
}

func (c *startAgentCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	
	if c.smithyId == "" {
		return subcommands.ExitFailure
	}

	// create agent
	agent, err := agent.New(c.serverUrl, c.credsPath, c.smithyId)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	// start agent
	fmt.Println("Starting agent...")
	if err = agent.Start(ctx); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	defer agent.Stop()

	return subcommands.ExitSuccess
}
