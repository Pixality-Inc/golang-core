package temporal

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"

	"go.temporal.io/sdk/client"
)

type Workflow interface {
	Name() WorkflowName
	Apply(ctx context.Context, workflowId string, queue string, input any, options ...StartWorkflowOption) (client.WorkflowRun, error)
}

type WorkflowImpl struct {
	log    logger.Loggable
	worker Worker
	config WorkflowConfig
}

func NewWorkflowImpl(
	worker Worker,
	config WorkflowConfig,
) *WorkflowImpl {
	return &WorkflowImpl{
		log: logger.NewLoggableImplWithServiceAndFields(
			"temporal_workflow",
			logger.Fields{
				logFieldName: config.Name,
			},
		),
		worker: worker,
		config: config,
	}
}

func (w *WorkflowImpl) Name() WorkflowName {
	return w.config.Name
}

func (w *WorkflowImpl) Apply(ctx context.Context, workflowId string, queue string, input any, options ...StartWorkflowOption) (client.WorkflowRun, error) {
	workflowOptions := client.StartWorkflowOptions{
		ID:        workflowId,
		TaskQueue: queue,
	}

	for _, option := range options {
		option(&workflowOptions)
	}

	return w.worker.ExecuteWorkflow(
		ctx,
		workflowOptions,
		w.Name(),
		input,
	)
}

func (w *WorkflowImpl) GetLoggerWithoutContext() logger.Logger {
	return w.log.GetLoggerWithoutContext()
}
