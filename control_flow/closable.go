package control_flow

type Closable interface {
	Close()
}

type ClosableImpl struct {
	service Closable
}

func NewClosable(service Closable) *ClosableImpl {
	return &ClosableImpl{
		service: service,
	}
}

func (c *ClosableImpl) Stop() error {
	c.service.Close()

	return nil
}

type ClosableWithError interface {
	Close() error
}

type ClosableWithErrorImpl struct {
	service ClosableWithError
}

func NewClosableWithError(service ClosableWithError) *ClosableWithErrorImpl {
	return &ClosableWithErrorImpl{
		service: service,
	}
}

func (c *ClosableWithErrorImpl) Stop() error {
	return c.service.Close()
}

type Stoppable interface {
	Stop()
}

type StoppableImpl struct {
	service Stoppable
}

func NewStoppable(service Stoppable) *StoppableImpl {
	return &StoppableImpl{
		service: service,
	}
}

func (s *StoppableImpl) Stop() error {
	s.service.Stop()

	return nil
}
