package flow

import "maps"

type EnvOption func(env *Env)

func WithEnv(name string, value string) EnvOption {
	return func(env *Env) {
		if env.Env == nil {
			env.Env = make(map[string]string)
		}

		env.Env[name] = value
	}
}

func WithEnvs(envs map[string]string) EnvOption {
	return func(env *Env) {
		if env.Env == nil {
			env.Env = make(map[string]string)
		}

		maps.Copy(env.Env, envs)
	}
}

type Env struct {
	WorkDir string
	Context map[string]any
	Env     map[string]string
}

func NewEnv(workDir string, context map[string]any, options ...EnvOption) *Env {
	if context == nil {
		context = make(map[string]any)
	}

	env := &Env{
		WorkDir: workDir,
		Context: context,
		Env:     make(map[string]string),
	}

	for _, option := range options {
		option(env)
	}

	return env
}

type Result struct {
	ActionsResponses map[string]*ActionResponse
	Data             map[string]any
}
