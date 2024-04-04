package kafkatest

import (
	"context"
	"fmt"
	"net"
	"testing"
	"time"

	"github.com/docker/docker/pkg/namesgenerator"
	"github.com/ory/dockertest/v3"
	"github.com/ory/dockertest/v3/docker"
	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

const (
	maxContainerLifetime = 2 * time.Minute
)

type Instance struct {
	Image                string
	Tag                  string
	kafkaPort            int
	schemaRegistryPort   int
	pandaProxyPort       int
	adminPort            int
	topicName            string
	resource             *dockertest.Resource
	maxContainerLifetime time.Duration
}

func New(t *testing.T) (*Instance, error) {
	t.Helper()

	topic := namesgenerator.GetRandomName(1)

	i := &Instance{
		Image:                "docker.redpanda.com/redpandadata/redpanda",
		Tag:                  "v23.3.10",
		topicName:            topic,
		maxContainerLifetime: maxContainerLifetime,
	}

	return i, nil
}

func (i *Instance) WithImage(image string) *Instance {
	i.Image = image
	return i
}

func (i *Instance) WithTag(tag string) *Instance {
	i.Tag = tag
	return i
}

func (i *Instance) WithTopicName(topic string) *Instance {
	i.topicName = topic
	return i
}

func (i *Instance) WithMaxContainerLifetime(d time.Duration) *Instance {
	i.maxContainerLifetime = d
	return i
}

func (i *Instance) Run(ctx context.Context) error {
	pool, err := dockertest.NewPool("")
	if err != nil {
		return fmt.Errorf("connecting to docker environment: %w", err)
	}

	ports, err := getRandomPort()
	if err != nil {
		return fmt.Errorf("unable to get random ports for kafka: %w", err)
	}

	i.kafkaPort = ports[0]
	i.schemaRegistryPort = ports[1]
	i.pandaProxyPort = ports[2]
	i.adminPort = ports[3]

	resource, err := pool.RunWithOptions(&dockertest.RunOptions{
		Repository: i.Image,
		Tag:        i.Tag,
		Env:        []string{},
		Cmd: []string{
			"redpanda",
			"start",
			"--kafka-addr",
			fmt.Sprintf("internal://0.0.0.0:9092,external://0.0.0.0:%d", i.kafkaPort),
			// # Address the broker advertises to clients that connect to the Kafka API.
			// # Use the internal addresses to connect to the Redpanda brokers'
			// # from inside the same Docker network.
			// # Use the external addresses to connect to the Redpanda brokers'
			// # from outside the Docker network.
			"--advertise-kafka-addr",
			fmt.Sprintf("internal://0.0.0.0:9092,external://0.0.0.0:%d", i.kafkaPort),
			"--pandaproxy-addr",
			fmt.Sprintf("internal://0.0.0.0:8082,external://0.0.0.0:%d", i.pandaProxyPort),
			// # Address the broker advertises to clients that connect to the HTTP Proxy.
			"--advertise-pandaproxy-addr",
			fmt.Sprintf("internal://0.0.0.0:8082,external://0.0.0.0:%d", i.pandaProxyPort),
			"--schema-registry-addr",
			fmt.Sprintf("internal://0.0.0.0:8081,external://0.0.0.0:%d", i.schemaRegistryPort),
			// # Redpanda brokers use the RPC API to communicate with eachother internally.
			"--rpc-addr",
			"127.0.0.1:33145",
			"--advertise-rpc-addr",
			"127.0.0.1:33145",
			// # Tells Seastar (the framework Redpanda uses under the hood) to use 1 core on the system.
			"--smp",
			"1",
			// # The amount of memory to make available to Redpanda.
			"--memory",
			"1G",
			"--node-id",
			"0",
			// # Mode dev-container uses well-known configuration properties for development in containers.
			"--mode",
			"dev-container",
			// # enable logs for debugging.
			// "--default-log-level=debug",
		},
		ExposedPorts: []string{
			fmt.Sprintf("%d/tcp", i.schemaRegistryPort),
			fmt.Sprintf("%d/tcp", i.pandaProxyPort),
			fmt.Sprintf("%d/tcp", i.kafkaPort),
		},
		PortBindings: map[docker.Port][]docker.PortBinding{
			docker.Port(fmt.Sprintf("%d/tcp", i.schemaRegistryPort)): {{HostIP: "localhost", HostPort: fmt.Sprintf("%d/tcp", i.schemaRegistryPort)}},
			docker.Port(fmt.Sprintf("%d/tcp", i.pandaProxyPort)):     {{HostIP: "localhost", HostPort: fmt.Sprintf("%d/tcp", i.pandaProxyPort)}},
			docker.Port(fmt.Sprintf("%d/tcp", i.kafkaPort)):          {{HostIP: "localhost", HostPort: fmt.Sprintf("%d/tcp", i.kafkaPort)}},
		},
	})
	if err != nil {
		return fmt.Errorf("creating rabbitmq container: %w", err)
	}

	_ = resource.Expire(uint(i.maxContainerLifetime / time.Second))

	var cl *kgo.Client
	if err := pool.Retry(func() error {
		var err error
		cl, err = kgo.NewClient(
			kgo.SeedBrokers(fmt.Sprintf("localhost:%d", i.kafkaPort)),
			kgo.RequestRetries(0),
			// TODO: add logger adapter
			// kgo.WithLogger(logger),
			// TODO: add kgo hooks for collecting metrics
			// kgo.WithHooks(),
		)
		if err != nil {
			return fmt.Errorf("unable to establish connection: %w", err)
		}

		return cl.Ping(context.Background())
	}); err != nil {
		return fmt.Errorf("connecting to kafka: %w", err)
	}

	i.resource = resource

	admcl := kadm.NewClient(cl)
	if _, err := admcl.CreateTopic(context.Background(), 10, 1, nil, i.topicName); err != nil {
		return fmt.Errorf("unable to create topic: %w", err)
	}

	// setup vhost

	return nil
}

func (i *Instance) SchemaRegistryPort() int {
	return i.schemaRegistryPort
}

func (i *Instance) PandaProxyPort() int {
	return i.pandaProxyPort
}

func (i *Instance) AdminPort() int {
	return i.adminPort
}

func (i *Instance) KafkaPort() int {
	return i.kafkaPort
}

func (i *Instance) Stop() error {
	return i.resource.Close()
}

func (i *Instance) TopicName() string {
	return i.topicName
}

func Run(t *testing.T, fn func(*Instance)) {
	i, err := New(t)
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = i.Stop() })

	if err := i.Run(context.Background()); err != nil {
		t.Fatal(err)
	}

	fn(i)
}

func getRandomPort() ([]int, error) {
	ports := []int{}

	for i := 0; i < 4; i++ {
		l, err := net.Listen("tcp", ":0")

		if err != nil {
			return []int{}, fmt.Errorf("net.Listen for random port: %w", err)
		}

		ports = append(ports, l.Addr().(*net.TCPAddr).Port)

		defer l.Close()
	}

	return ports, nil
}
