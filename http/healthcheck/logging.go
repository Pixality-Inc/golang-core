package healthcheck

type NamedService interface {
	Service
	Name() string
}

type namedService struct {
	name  string
	inner Service
}

func Named(name string, service Service) NamedService {
	return &namedService{name: name, inner: service}
}

func (n *namedService) Name() string {
	return n.name
}

func (n *namedService) IsOK() bool {
	return n.inner.IsOK()
}
