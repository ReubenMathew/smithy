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
	clusterId  string
	agentId   string
}

func startAgentCommand() subcommands.Command {
	return &startAgentCmd{
		metaCommand: metaCommand{
			name:     "start-agent",
			synopsis: "Starts agent process",
			usage:    "start-agent -server <url> -creds <path/to/file> -cluster <string> -id <string>",
		},
	}
}

func (c *startAgentCmd) SetFlags(f *flag.FlagSet) {
	f.StringVar(&c.serverUrl, "server", nats.DefaultURL, "Server URL")
	f.StringVar(&c.credsPath, "creds", "", "Credentials file path")
	f.StringVar(&c.clusterId, "cluster", "default", "Smithy instance id")
	f.StringVar(&c.agentId, "id", "", "Agent id")
}

func (c *startAgentCmd) Execute(ctx context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {

	if c.agentId == "" {
		return subcommands.ExitFailure
	}

	// create agent
	agent, err := agent.New(c.serverUrl, c.credsPath, c.clusterId, c.agentId)
	if err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}

	// start agent
	if err = agent.Start(ctx); err != nil {
		fmt.Println(err)
		return subcommands.ExitFailure
	}
	defer agent.Stop()

	return subcommands.ExitSuccess
}
