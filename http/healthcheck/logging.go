package healthcheck

type namedService struct {
	name  string
	inner Service
}

func Named(name string, service Service) Service {
	return &namedService{name: name, inner: service}
}

func (n *namedService) IsOK() bool {
	return n.inner.IsOK()
}
