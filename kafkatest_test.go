package kafkatest_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/aranw/kafkatest"

	"github.com/twmb/franz-go/pkg/kadm"
	"github.com/twmb/franz-go/pkg/kgo"
)

func TestKafkaTest_New(t *testing.T) {
	i, err := kafkatest.New(t)
	if err != nil {
		t.Fatalf("unable to instantiate kafkatest instance: %v", err)
	}
	t.Cleanup(func() { _ = i.Stop() })
	i = i.WithTopicName("test-topic")

	if err := i.Run(context.Background()); err != nil {
		t.Fatalf("unable to run kafka server: %v", err)
	}

	cl, err := kgo.NewClient(kgo.SeedBrokers(fmt.Sprintf("localhost:%d", i.KafkaPort())))
	if err != nil {
		t.Fatalf("unable to establish kafka connection: %v", err)
	}
	if err := cl.Ping(context.Background()); err != nil {
		t.Fatalf("unable to ping kafka instance: %v", err)
	}

	kadmcl := kadm.NewClient(cl)
	if _, err := kadmcl.DescribeTopicConfigs(context.Background(), i.TopicName()); err != nil {
		t.Fatalf("unable to describe topic configs: %v", err)
	}
}

func TestKafkaTest_Run(t *testing.T) {
	kafkatest.Run(t, func(i *kafkatest.Instance) {
		cl, err := kgo.NewClient(kgo.SeedBrokers(fmt.Sprintf("localhost:%d", i.KafkaPort())))
		if err != nil {
			t.Fatalf("unable to establish kafka connection: %v", err)
		}
		if err := cl.Ping(context.Background()); err != nil {
			t.Fatalf("unable to ping kafka instance: %v", err)
		}

		kadmcl := kadm.NewClient(cl)
		if _, err := kadmcl.DescribeTopicConfigs(context.Background(), i.TopicName()); err != nil {
			t.Fatalf("unable to describe topic configs: %v", err)
		}
	})
}
