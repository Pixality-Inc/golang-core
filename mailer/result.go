package mailer

type Result interface {
	Id() string
}

type ResultImpl struct {
	id string
}

func NewResult(id string) *ResultImpl {
	return &ResultImpl{
		id: id,
	}
}

func (r *ResultImpl) Id() string {
	return r.id
}
