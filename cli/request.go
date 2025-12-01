package cli

import "io"

type Request struct {
	workDir string
	stdout  io.Writer
	stderr  io.Writer
	envs    map[string]string
}

func NewRequest() *Request {
	return &Request{
		workDir: "",
		envs:    make(map[string]string),
	}
}
