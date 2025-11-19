package base_env

import (
	"github.com/pixality-inc/golang-core/env"
	"github.com/pixality-inc/golang-core/logger"
)

type BaseEnv interface {
	Logger() logger.Logger
}

type BaseEnvImpl struct {
	logger logger.Logger
	appEnv env.AppEnv
}

func NewBaseEnv(
	appEnv env.AppEnv,
	loggerConfig logger.Config,
) BaseEnv {
	// Logger
	log := logger.New(loggerConfig)

	if err := logger.InitLogSpawner(log); err != nil {
		log.WithError(err).Fatal("error initializing log spawner")
	}

	// Log current version

	log.
		WithFields(map[string]any{
			"env_name":         appEnv.EnvName(),
			"git_branch":       appEnv.GitBranch(),
			"git_commit":       appEnv.GitCommit(),
			"git_commit_short": appEnv.GitCommitShort(),
			"git_tag":          appEnv.GitTag(),
			"ci_pipeline_id":   appEnv.CiPipelineId(),
		}).
		Infof("Initializing application")

	// Return

	return &BaseEnvImpl{
		logger: log,
		appEnv: appEnv,
	}
}

func (e *BaseEnvImpl) Logger() logger.Logger {
	return e.logger
}

func (e *BaseEnvImpl) AppEnv() env.AppEnv {
	return e.appEnv
}
