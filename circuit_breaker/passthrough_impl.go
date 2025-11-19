package circuit_breaker

type passthroughImpl struct{}

func (p *passthroughImpl) Execute(fn func() error) error {
	return fn()
}

func (p *passthroughImpl) ExecuteWithResult(fn func() (any, error)) (any, error) {
	return fn()
}
