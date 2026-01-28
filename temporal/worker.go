package temporal

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"

	"go.temporal.io/sdk/activity"
	"go.temporal.io/sdk/client"
	"go.temporal.io/sdk/worker"
	"go.temporal.io/sdk/workflow"
)

//go:generate mockgen -destination mocks/worker_gen.go -source worker.go
type Worker interface {
	RegisterWorkflow(wf Workflow, runner WorkflowRunner) error

	RegisterActivity(a Activity, runner ActivityRunner) error

	ExecuteWorkflow(
		ctx context.Context,
		options client.StartWorkflowOptions,
		workflow any,
		args ...any,
	) (client.WorkflowRun, error)

	Run() error

	Stop()
}

type WorkerImpl struct {
	log        logger.Loggable
	client     client.Client
	worker     worker.Worker
	identifier string
}

func NewWorker(
	ctx context.Context,
	client client.Client,
	queue string,
	identifier string,
	config WorkerConfig,
) Worker {
	return &WorkerImpl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"temporal_worker",
			logger.Fields{
				"name": config.Name,
			},
		),
		client: client,
		worker: worker.New(client, queue, worker.Options{
			BackgroundActivityContext:          ctx,
			Identity:                           identifier,
			MaxConcurrentActivityExecutionSize: config.MaxConcurrentActivityExecutionSize,
			WorkerActivitiesPerSecond:          config.MaxActivitiesPerSecond,
			WorkerStopTimeout:                  config.WorkerStopTimeout,
		}),
		identifier: identifier,
	}
}

func (w *WorkerImpl) Run() error {
	return w.worker.Run(nil)
}

func (w *WorkerImpl) Stop() {
	w.worker.Stop()
}

func (w *WorkerImpl) RegisterWorkflow(wf Workflow, runner WorkflowRunner) error {
	w.worker.RegisterWorkflowWithOptions(runner, workflow.RegisterOptions{
		Name: string(wf.Name()),
	})

	return nil
}

func (w *WorkerImpl) RegisterActivity(a Activity, runner ActivityRunner) error {
	w.log.GetLoggerWithoutContext().Debugf("Registering activity %s (queue: %s)", a.Name(), a.Queue())

	w.worker.RegisterActivityWithOptions(runner, activity.RegisterOptions{
		Name: string(a.Name()),
	})

	return nil
}

func (w *WorkerImpl) ExecuteWorkflow(
	ctx context.Context,
	options client.StartWorkflowOptions,
	workflow any,
	args ...any,
) (client.WorkflowRun, error) {
	return w.client.ExecuteWorkflow(ctx, options, workflow, args...)
}
