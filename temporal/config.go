package temporal

import "time"

type (
	QueueName    string
	WorkflowName string
	ActivityName string
)

type ActivityConfig struct {
	Name                    ActivityName  `env:"NAME"                      yaml:"name"`
	Queue                   QueueName     `env:"QUEUE"                     yaml:"queue"`
	Timeout                 time.Duration `env:"TIMEOUT"                   yaml:"timeout"`
	MaxAttempts             int           `env:"MAX_ATTEMPTS"              yaml:"max_attempts"`
	RetryInitialInterval    time.Duration `env:"RETRY_INITIAL_INTERVAL"    yaml:"retry_initial_interval"`
	RetryBackoffCoefficient float64       `env:"RETRY_BACKOFF_COEFFICIENT" yaml:"retry_backoff_coefficient"`
	RetryMaximumInterval    time.Duration `env:"RETRY_MAX_INTERVAL"        yaml:"retry_maximum_interval"`
}

type QueueConfig struct {
	Name QueueName `env:"NAME" yaml:"name"`
}

type WorkflowConfig struct {
	Name WorkflowName `env:"NAME" yaml:"name"`
}

type WorkerConfig struct {
	Name                               string        `env:"NAME"                                   yaml:"name"`
	MaxConcurrentActivityExecutionSize int           `env:"MAX_CONCURRENT_ACTIVITY_EXECUTION_SIZE" yaml:"max_concurrent_activity_execution_size"`
	MaxActivitiesPerSecond             float64       `env:"MAX_ACTIVITIES_PER_SECOND"              yaml:"max_activities_per_second"`
	WorkerStopTimeout                  time.Duration `env:"WORKER_STOP_TIMEOUT"                    yaml:"worker_stop_timeout"`
}
