package cli

type Result interface {
	ExitCode() int
	Stdout() []byte
	Stderr() []byte
}

type ResultImpl struct {
	exitCode int
	stdout   []byte
	stderr   []byte
}

func NewResult(exitCode int, stdout []byte, stderr []byte) *ResultImpl {
	return &ResultImpl{
		exitCode: exitCode,
		stdout:   stdout,
		stderr:   stderr,
	}
}

func (r *ResultImpl) ExitCode() int {
	return r.exitCode
}

func (r *ResultImpl) Stdout() []byte {
	return r.stdout
}

func (r *ResultImpl) Stderr() []byte {
	return r.stderr
}
