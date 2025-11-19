package env

import "time"

type AppEnv interface {
	EnvName() string
	CiPipelineId() string
	GitTag() string
	GitBranch() string
	GitCommitShort() string
	GitCommit() string
	StartedAt() time.Time
}

type Impl struct {
	envName        string
	ciPipelineId   string
	gitTag         string
	gitBranch      string
	gitCommit      string
	gitCommitShort string
	startedAt      time.Time
}

func New(
	envName string,
	ciPipelineId string,
	gitTag string,
	gitBranch string,
	gitCommit string,
	gitCommitShort string,
	startedAt time.Time,
) *Impl {
	return &Impl{
		envName:        envName,
		ciPipelineId:   ciPipelineId,
		gitTag:         gitTag,
		gitBranch:      gitBranch,
		gitCommit:      gitCommit,
		gitCommitShort: gitCommitShort,
		startedAt:      startedAt,
	}
}

func (a *Impl) EnvName() string {
	return a.envName
}

func (a *Impl) CiPipelineId() string {
	return a.ciPipelineId
}

func (a *Impl) GitTag() string {
	return a.gitTag
}

func (a *Impl) GitBranch() string {
	return a.gitBranch
}

func (a *Impl) GitCommit() string {
	return a.gitCommit
}

func (a *Impl) GitCommitShort() string {
	return a.gitCommitShort
}

func (a *Impl) StartedAt() time.Time {
	return a.startedAt
}
