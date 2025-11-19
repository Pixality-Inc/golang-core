package temporal

import (
	"context"

	"github.com/pixality-inc/golang-core/logger"

	"go.temporal.io/sdk/client"
)

type Workflow interface {
	Name() WorkflowName
	Apply(ctx context.Context, workflowId string, queue string, input any) (client.WorkflowRun, error)
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
				"name": config.Name,
			},
		),
		worker: worker,
		config: config,
	}
}

func (w *WorkflowImpl) Name() WorkflowName {
	return w.config.Name
}

func (w *WorkflowImpl) Apply(ctx context.Context, workflowId string, queue string, input any) (client.WorkflowRun, error) {
	return w.worker.ExecuteWorkflow(
		ctx,
		client.StartWorkflowOptions{
			ID:        workflowId,
			TaskQueue: queue,
		},
		w.Name(),
		input,
	)
}

func (w *WorkflowImpl) GetLoggerWithoutContext() logger.Logger {
	return w.log.GetLoggerWithoutContext()
}
