package temporal

import (
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
)

type WorkerOption = func(options *worker.Options)

type StartWorkflowOption = func(options *client.StartWorkflowOptions)
