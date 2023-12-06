package agent

import (
	"context"
	"fmt"

	"github.com/nats-io/nats.go"
	"github.com/nats-io/nats.go/jetstream"
)

type Agent struct {
	clusterId string
	nc        *nats.Conn
}

func New(serverUrl string, credsPath string, clusterId string) (*Agent, error) {

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
	}, nil
}

func (a *Agent) Start(ctx context.Context) error {

	js, err := jetstream.New(a.nc)
	if err != nil {
		return err
	}

	consumer, err := js.OrderedConsumer(ctx, a.clusterId, jetstream.OrderedConsumerConfig{})
	if err != nil {
		return err
	}

	consumer.Consume(func(msg jetstream.Msg) {
		fmt.Printf("Received message: %s\n", string(msg.Data()))
	})

	// block until context is done
	<-ctx.Done()

	return nil
}

func (a *Agent) Stop() {
	a.nc.Close()
}
