/*
Package dynoscaler scales Heroku workers proportionally to RabbitMQ queues.

It utilizes information about queued/unacked messages in a RabbitMQ queue
combined with pre-defined message-worker ratios.

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

For more details, visit https://github.com/monsterroster/dynoscaler.
*/
package dynoscaler

import (
	"context"
	"sort"
	"time"

	heroku "github.com/heroku/heroku-go/v3"
	rabbithole "github.com/michaelklishin/rabbit-hole"
	"github.com/pkg/errors"
	"github.com/sirupsen/logrus"
)

// DynoScaler has the ability to scale dynos on Heroku
// according to some configuration combined with details
// about the message counts in a RabbitMQ queue.
type DynoScaler struct {
	rabbitMQHost     string
	rabbitMQUsername string
	rabbitMQPassword string
	herokuAPIKey     string
	herokuAppID      string
	workerConfigs    []WorkerConfig
	log              *logrus.Entry

	// How long to sleep between the checks.
	CheckInterval time.Duration

	// Where to log errors.
	Logger *logrus.Logger
}

// NewDynoScaler initializes a new DynoScaler with specified and default values.
// It will connect to the RabbitMQ Management API with TLS using the provided
// information to get current details about the queues. It also utilizes the
// Heroku Platform API to get the current formation for the specified app, and
// to update the formation (scale) to the desired quantity based on the total
// number of unacked and queued messages.
func NewDynoScaler(
	rabbitMQHost,
	rabbitMQUsername,
	rabbitMQPassword,
	herokuAPIKey,
	herokuAppID string,
	workerConfigs ...WorkerConfig,
) DynoScaler {
	logger := logrus.New()
	logger.SetLevel(logrus.PanicLevel)

	return DynoScaler{
		rabbitMQHost:     rabbitMQHost,
		rabbitMQUsername: rabbitMQUsername,
		rabbitMQPassword: rabbitMQPassword,
		herokuAPIKey:     herokuAPIKey,
		herokuAppID:      herokuAppID,
		workerConfigs:    workerConfigs,
		log:              logger.WithField("pkg", "dynoscaler"),
		CheckInterval:    10 * time.Second,
		Logger:           logger,
	}
}

// Monitor watches the queue message count and scales the dynos accordingly.
func (ds *DynoScaler) Monitor() error {
	heroku.DefaultTransport.BearerToken = ds.herokuAPIKey
	hs := heroku.NewService(heroku.DefaultClient)

	rmqc, err := rabbithole.NewClient("https://"+ds.rabbitMQHost, ds.rabbitMQUsername, ds.rabbitMQPassword)
	if err != nil {
		return errors.Wrap(err, "failed to initialize rabbithole client")
	}

	// make sure auth works and app exists
	_, err = hs.DynoList(context.TODO(), ds.herokuAppID, nil)
	if err != nil {
		return errors.Wrap(err, "failed to verify Heroku app exists")
	}

	ds.log.Info("starting monitoring")

	for {
		queues, err := rmqc.ListQueues()
		if err != nil {
			ds.log.WithError(err).Error("failed to list queues")
			time.Sleep(ds.CheckInterval)
			continue
		}

		formationList, err := hs.FormationList(context.TODO(), ds.herokuAppID, nil)
		if err != nil {
			ds.log.WithError(err).Error("failed to list formations")
			time.Sleep(ds.CheckInterval)
			continue
		}

		for _, wc := range ds.workerConfigs {
			newQuantity, scale, err := ds.checkScaling(wc, queues, formationList)
			if err != nil {
				ds.log.WithError(err).WithFields(logrus.Fields{
					"heroku_app":  ds.herokuAppID,
					"worker_type": wc.WorkerType,
				}).Error("failed to check whether to scale or not")
				time.Sleep(ds.CheckInterval)
				continue
			}

			if scale {
				ds.log.WithFields(logrus.Fields{
					"heroku_app":   ds.herokuAppID,
					"worker_type":  wc.WorkerType,
					"new_quantity": newQuantity,
				}).Info("scaling dynos")

				err := scaleDynos(hs, ds.herokuAppID, wc.WorkerType, newQuantity)
				if err != nil {
					ds.log.WithError(err).Error("failed to update Heroku formation")
					time.Sleep(ds.CheckInterval)
					continue
				}
			}
		}

		time.Sleep(ds.CheckInterval)
	}
}

// scaleDynos scales herokuAppName's process with the name workerType (name that is
// used in the Procfile) to the number of dynos specified by quantity.
func scaleDynos(hs *heroku.Service, herokuAppName, workerType string, quantity int) error {
	_, err := hs.FormationUpdate(
		context.TODO(),
		herokuAppName,
		workerType,
		heroku.FormationUpdateOpts{Quantity: &quantity},
	)

	return err
}

// maxWorkerCount returns the number of workers that should
// be used according to the ratio map and the current message count.
func maxWorkerCount(ratioMap map[int]int, curMsgCount int) int {
	// don't rely on golang random map order
	keys := make([]int, len(ratioMap))
	i := 0
	for k := range ratioMap {
		keys[i] = k
		i++
	}
	sort.Ints(keys)

	max := 0

	for _, msgCount := range keys {
		if curMsgCount >= msgCount {
			max = ratioMap[msgCount]
		} else {
			break
		}
	}

	return max
}

// checkScaling checks whether the worker should be scaled and what it should be scaled to.
func (ds *DynoScaler) checkScaling(
	qc WorkerConfig,
	queues []rabbithole.QueueInfo,
	formations []heroku.Formation,
) (newQuantity int, scale bool, err error) {

	var qInfo *rabbithole.QueueInfo
	for _, qi := range queues {
		if qi.Name == qc.QueueName {
			qInfo = &qi
			break
		}
	}

	if qInfo == nil {
		return 0, false, errors.New("unable to find queue info from RabbitMQ data")
	}

	var formation *heroku.Formation
	for _, f := range formations {
		if f.Type == qc.WorkerType {
			formation = &f
			break
		}
	}

	if formation == nil {
		return 0, false, errors.New("unable to find formation info from Heroku data")
	}

	totalMsgs := qInfo.MessagesUnacknowledged + qInfo.Messages

	if totalMsgs > 0 {
		desiredQuantity := maxWorkerCount(qc.MsgWorkerRatios, totalMsgs)
		if formation.Quantity < desiredQuantity {
			scale = true
			newQuantity = desiredQuantity
		}
	} else if formation.Quantity > 0 {
		scale = true
		newQuantity = 0
	}

	return newQuantity, scale, nil
}
