package agent

import (
	"context"
	"fmt"
	"log"
	"os"
	"os/exec"
	"smithy/internal/meta"

	"github.com/nats-io/nats.go"
)

type Agent struct {
	clusterId string
	agentId   string
	nc        *nats.Conn
}

const (
	SmithyAgentsStreamName = "smithy-agents"
)

func New(serverUrl string, credsPath string, clusterId string, agentId string) (*Agent, error) {

	opts := []nats.Option{}

	if credsPath != "" {
		opts = append(opts, nats.UserCredentials(credsPath))
	}

	nc, err := nats.Connect(serverUrl, opts...)
	if err != nil {
		return nil, err
	}

	return &Agent{
		nc:        nc,
		clusterId: clusterId,
		agentId:   agentId,
	}, nil
}

func (a *Agent) Start(ctx context.Context) error {

	log.Printf("Started Agent %s for cluster %s", a.agentId, a.clusterId)

	// get server config from object store
	js, err := a.nc.JetStream()
	if err != nil {
		return err
	}
	obj, err := js.ObjectStore(meta.SmithyClustersObjStoreName)
	if err != nil {
		return err
	}

	configFileName := fmt.Sprintf("%s-server.conf", a.clusterId)
	serverConfigFilePath := fmt.Sprintf("%s/%s", os.Getenv("HOME"), configFileName)

	a.nc.Subscribe(fmt.Sprintf("%s.%s", SmithyAgentsStreamName, a.clusterId), func(msg *nats.Msg) {
		fmt.Printf("Received message: %s\n", string(msg.Data))

		if string(msg.Data) == "start" {

			// create empty server.conf file in home directory
			if err = os.WriteFile(serverConfigFilePath, []byte(""), 0644); err != nil {
				panic(err)
			}
			// get server.conf file from object store
			if err = obj.GetFile(configFileName, serverConfigFilePath); err != nil {
				fmt.Printf("Error getting file: %s\n", err.Error())
				return
			}

			// run the command `nats-server` in a subprocess
			args := []string{"-c", serverConfigFilePath, "-name", a.agentId}
			cmd := exec.Command("nats-server", args...)
			if err = cmd.Start(); err != nil {
				fmt.Printf("Error running command: %s\n", err.Error())
			} else {
				fmt.Printf("nats-server running in process %d\n", cmd.Process.Pid)
			}
		}

	})

	// block until context is done
	<-ctx.Done()

	return nil
}

func (a *Agent) Stop() {
	a.nc.Close()
}
