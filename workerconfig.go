package dynoscaler

// WorkerConfig holds the scaling settings for a specific dyno and queue.
type WorkerConfig struct {
	// Number of workers to use once the queue reaches a certain
	// number of messages. For example, if {1: 1, 10: 2, 30: 5}
	// was used, one worker would be used when the first message
	// comes in, then if the queue grows to 10 messages, a second
	// worker would be started up. Finally, if the queue grows to
	// 30 messages, another 3 workers would be started up.
	MsgWorkerRatios map[int]int

	// Name of the AMQP queue to track.
	QueueName string

	// Name of the process on Heroku.
	// This is the same name you use in the Procfile.
	WorkerType string
}
