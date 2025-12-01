package cli

import (
	"io"
	"maps"
)

type Option = func(request *Request)

func WithWorkDir(workDir string) Option {
	return func(request *Request) {
		request.workDir = workDir
	}
}

func WithStdout(stdout io.Writer) Option {
	return func(request *Request) {
		request.stdout = stdout
	}
}

func WithStderr(stderr io.Writer) Option {
	return func(request *Request) {
		request.stderr = stderr
	}
}

func WithEnv(name string, value string) Option {
	return func(request *Request) {
		request.envs[name] = value
	}
}

func WithEnvs(envs map[string]string) Option {
	return func(request *Request) {
		maps.Copy(request.envs, envs)
	}
}
