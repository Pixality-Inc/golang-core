package healthcheck

type Service interface {
	IsOK() bool
}
