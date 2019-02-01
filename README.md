# dynoscaler ![License](https://img.shields.io/github/license/monsterroster/dynoscaler.svg?style=flat) ![](https://img.shields.io/github/tag/monsterroster/dynoscaler.svg?label=release&style=flat) [![Go Report Card](https://goreportcard.com/badge/github.com/monsterroster/dynoscaler)](https://goreportcard.com/report/github.com/monsterroster/dynoscaler)  [![GoDoc](https://godoc.org/github.com/monsterroster/dynoscaler?status.svg)](https://godoc.org/github.com/monsterroster/dynoscaler)

The dynoscaler package is meant to be a simple solution to scale workers on
Heroku according to the size of their corresponding RabbitMQ queues. It is written
in Go but works in a very similar way to
[hirefire](https://github.com/hirefire/hirefire) (no longer maintained) which is
also where the inspiration came from.

It should be run 24/7 either in an individual application or as a background
task as a part of some other application (that does not experience frequent
downtime).

## Project Maturity

While the project is brand-new, it is fairly simple in nature. It is currently
in use in a production environment where it does its job very well.

## Requirements

* RabbitMQ server with the 
  [Management Plugin](https://www.rabbitmq.com/management.html) to be able to use 
  the HTTP API
* Heroku Platform API access

## Usage

The setup is quite straightforward. One `DynoScaler` gets paired with one
RabbitMQ server as well as one Heroku app. Finally, you specify how a certain
worker should scale in relation to the corresponding queue size:

```go
ds := dynoscaler.NewDynoScaler(
    "baboon.rmq.cloudamqp.com",
    "username",
    "password",
    "heroku Platform API key",
    "heroku app name",
    dynoscaler.WorkerConfig{
        MsgWorkerRatios: map[int]int{1: 1},
        QueueName:       "foo",
        WorkerType:      "fooworker",
    },
    dynoscaler.WorkerConfig{
        MsgWorkerRatios: map[int]int{1: 1, 10: 2, 30: 5},
        QueueName:       "bar",
        WorkerType:      "mainworker",
    },
)

ds.Logger.SetLevel(logrus.InfoLevel)

err = ds.Monitor()
logrus.WithError(err).Error("dynoscaler monitoring failed")
```    
	
The RabbitMQ Management HTTP API is utilized for the message counts, since it
provides both the total queued message count as well as the total unacked
message count.

By default, the checking (and any necessary changes to the scaling) will be
done every 10 seconds. This is configurable using the `CheckInterval` property.

For more details about `MsgWorkerRatios` and other properties please check the
[Godoc](https://godoc.org/github.com/monsterroster/dynoscaler) documentation.

## Logging

[Logrus](https://github.com/sirupsen/logrus) is used for logging, which is
turned off by default. Please see the usage example above for an example on how
to enable it.

## Contributing

Suggestions for improvements as well as pull requests are welcome.