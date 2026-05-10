package net

type Address interface {
	Local() string
	Remote() string
}

type AddressImpl struct {
	local  string
	remote string
}

func NewAddress(local string, remote string) Address {
	return &AddressImpl{
		local:  local,
		remote: remote,
	}
}

func (a *AddressImpl) Local() string {
	return a.local
}

func (a *AddressImpl) Remote() string {
	return a.remote
}
