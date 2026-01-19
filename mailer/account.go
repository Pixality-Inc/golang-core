package mailer

type Account interface {
	Email() string
	Name() string
	String() string
}

type AccountImpl struct {
	email string
	name  string
}

func NewAccount(email string) *AccountImpl {
	return &AccountImpl{
		email: email,
		name:  "",
	}
}

func (a *AccountImpl) WithName(name string) *AccountImpl {
	a.name = name

	return a
}

func (a *AccountImpl) Name() string {
	return a.name
}

func (a *AccountImpl) Email() string {
	return a.email
}

func (a *AccountImpl) String() string {
	if a.name != "" {
		return a.name + " <" + a.email + ">"
	}

	return a.email
}
