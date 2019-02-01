package dynoscaler

import (
	"testing"

	heroku "github.com/heroku/heroku-go/v3"
	rabbithole "github.com/michaelklishin/rabbit-hole"
)

func TestCheckScalingDown(t *testing.T) {
	ds := NewDynoScaler("", "", "", "", "")

	newQuantity, scale, err := ds.checkScaling(
		WorkerConfig{
			MsgWorkerRatios: map[int]int{1: 1},
			QueueName:       "foo",
			WorkerType:      "bar",
		},
		[]rabbithole.QueueInfo{
			{
				Name:                   "foo",
				MessagesUnacknowledged: 0,
				Messages:               0,
			},
		}, []heroku.Formation{
			{
				Quantity: 1,
				Type:     "bar",
			},
		},
	)

	if err != nil {
		t.Fatalf("expected error to be nil, got %s", err.Error())
	}

	if newQuantity != 0 {
		t.Errorf("expected newQuantity to be 0, got %d", newQuantity)
	}

	if !scale {
		t.Error("expected scale to be true")
	}
}

func TestCheckScalingUp(t *testing.T) {
	ds := NewDynoScaler("", "", "", "", "")

	newQuantity, scale, err := ds.checkScaling(
		WorkerConfig{
			MsgWorkerRatios: map[int]int{1: 1},
			QueueName:       "foo",
			WorkerType:      "bar",
		},
		[]rabbithole.QueueInfo{
			{
				Name:                   "foo",
				MessagesUnacknowledged: 1,
				Messages:               1,
			},
		}, []heroku.Formation{
			{
				Quantity: 0,
				Type:     "bar",
			},
		},
	)

	if err != nil {
		t.Fatalf("expected error to be nil, got %s", err.Error())
	}

	if newQuantity != 1 {
		t.Errorf("expected newQuantity to be 1, got %d", newQuantity)
	}

	if !scale {
		t.Error("expected scale to be true")
	}
}

func TestCheckScalingUpMultiple(t *testing.T) {
	ds := NewDynoScaler("", "", "", "", "")

	newQuantity, scale, err := ds.checkScaling(
		WorkerConfig{
			MsgWorkerRatios: map[int]int{1: 1, 5: 2, 10: 4},
			QueueName:       "foo",
			WorkerType:      "bar",
		},
		[]rabbithole.QueueInfo{
			{
				Name:                   "foo",
				MessagesUnacknowledged: 0,
				Messages:               10,
			},
		}, []heroku.Formation{
			{
				Quantity: 1,
				Type:     "bar",
			},
		},
	)

	if err != nil {
		t.Fatalf("expected error to be nil, got %s", err.Error())
	}

	if newQuantity != 4 {
		t.Errorf("expected newQuantity to be 4, got %d", newQuantity)
	}

	if !scale {
		t.Error("expected scale to be true")
	}
}

func TestCheckScalingNone(t *testing.T) {
	ds := NewDynoScaler("", "", "", "", "")

	_, scale, err := ds.checkScaling(
		WorkerConfig{
			MsgWorkerRatios: map[int]int{1: 1},
			QueueName:       "foo",
			WorkerType:      "bar",
		},
		[]rabbithole.QueueInfo{
			{
				Name:                   "foo",
				MessagesUnacknowledged: 0,
				Messages:               0,
			},
		}, []heroku.Formation{
			{
				Quantity: 0,
				Type:     "bar",
			},
		},
	)

	if err != nil {
		t.Fatalf("expected error to be nil, got %s", err.Error())
	}

	if scale {
		t.Error("expected scale to be false")
	}
}

func TestCheckScalingNoQueueInfo(t *testing.T) {
	ds := NewDynoScaler("", "", "", "", "")

	_, _, err := ds.checkScaling(
		WorkerConfig{
			MsgWorkerRatios: map[int]int{1: 1},
			QueueName:       "foo",
			WorkerType:      "bar",
		},
		[]rabbithole.QueueInfo{
			{
				Name:                   "zoo",
				MessagesUnacknowledged: 0,
				Messages:               0,
			},
		}, []heroku.Formation{
			{
				Quantity: 0,
				Type:     "bar",
			},
		},
	)

	if err == nil {
		t.Fatal("expected error to not be nil")
	}

	if err.Error() != "unable to find queue info from RabbitMQ data" {
		t.Error("expected error about lack of RabbitMQ data")
	}
}

func TestCheckScalingNoFormationInfo(t *testing.T) {
	ds := NewDynoScaler("", "", "", "", "")

	_, _, err := ds.checkScaling(
		WorkerConfig{
			MsgWorkerRatios: map[int]int{1: 1},
			QueueName:       "foo",
			WorkerType:      "bar",
		},
		[]rabbithole.QueueInfo{
			{
				Name:                   "foo",
				MessagesUnacknowledged: 0,
				Messages:               0,
			},
		}, []heroku.Formation{
			{
				Quantity: 0,
				Type:     "zoo",
			},
		},
	)

	if err == nil {
		t.Fatal("expected error to not be nil")
	}

	if err.Error() != "unable to find formation info from Heroku data" {
		t.Error("expected error about lack of formation data")
	}
}
