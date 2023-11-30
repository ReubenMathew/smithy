package cmd

import (
	"context"
	"flag"
	"fmt"

	"github.com/google/subcommands"
)

type deployAgentsCmd struct {
	metaCommand
	numberOfAgents uint
}

func deployAgentsCommand() subcommands.Command {
	return &deployAgentsCmd{
		metaCommand: metaCommand{
			name:     "deploy-agents",
			synopsis: "Provision a set of cloud compute instances, each running an agent, within a security group",
			usage:    "deploy-agent -n <number>",
		},
	}
}

func (ec *deployAgentsCmd) SetFlags(f *flag.FlagSet) {
	f.UintVar(&ec.numberOfAgents, "n", 1, "number of agents")
}

func (ec *deployAgentsCmd) Execute(_ context.Context, f *flag.FlagSet, args ...interface{}) subcommands.ExitStatus {
	const (
		numberOfArgs = 1
	)

	rootOpts := args[0].(*rootOptions)

	if rootOpts.verbose {
		fmt.Printf("Args: %v\n", f.Args())
	}

	return subcommands.ExitSuccess
}
