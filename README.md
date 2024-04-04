# kafkatest

## Introduction

kafkatest package is a Golang testing library designed to facilitate integration tests involving the publishing to and consuming from Kafka. This library leverages the Redpanda Kafka-Compatible service, enabling quick and easy provisioning of a standalone Kafka server.

## Getting Started

Using kafkatest is straightforward and simple. Check the [releases tab](https://github.com/aranw/kafkatest/releases) for available releases.

```shell
go get -u github.com/aranw/kafkatest
```

For this library to function correctly, Docker must be installed on the machine where the tests are being executed.

## Usage

To use the library you first need initailise the library:

```go
i, err := kafkatest.New(t)
if err != nil {
    t.Fatalf("initialising kafkatest instance: %v", err)
}
t.Cleanup(func() { _ = i.Stop() })

if err := i.Run(context.Background()); err != nil {
    t.Fatalf("running kafka server: %v", err)
}
```

If you wish to customise some of the settings there are various functions that can be called to change settings:

```go
i := i.WithImage("docker.redpanda.com/vectorized/redpanda")
i := i.WithTag("latest")
i := i.WithTopicName("some-topic-name")
i := i.WithMaxContainerLifetime(1 * time.Minute)

if err := i.Run(context.Background()); err != nil {
    t.Fatalf("running kafka server: %v", err)
}
```

Right now the library relies on the `docker.redpanda.com/vectorized/redpanda` docker image and is pinned to `v23.3.10`. You can override it using the `WithImage` and `WithTag` functions. 

## Examples

Example usage can be found in the packages tests in `kafkatest_test.go`.
