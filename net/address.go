package net

import (
	stdNet "net"
	"strconv"
)

type Address interface {
	Host() string
	Port() int
}

type AddressImpl struct {
	host string
	port int
}

func NewAddress(host string, port int) Address {
	return &AddressImpl{
		host: host,
		port: port,
	}
}

func NewAddressFromNet(address stdNet.Addr) Address {
	if address == nil {
		return NewAddress("", 0)
	}

	return NewAddressFromString(address.String())
}

func NewAddressFromString(address string) Address {
	host, portValue, err := stdNet.SplitHostPort(address)
	if err != nil {
		return NewAddress(address, 0)
	}

	port, err := strconv.Atoi(portValue)
	if err != nil {
		return NewAddress(host, 0)
	}

	return NewAddress(host, port)
}

func (a *AddressImpl) Host() string {
	return a.host
}

func (a *AddressImpl) Port() int {
	return a.port
}

type Addresses interface {
	Local() Address
	Remote() Address
}

type AddressesImpl struct {
	local  Address
	remote Address
}

func NewAddresses(local Address, remote Address) Addresses {
	return &AddressesImpl{
		local:  local,
		remote: remote,
	}
}

func NewAddressesFromNet(local stdNet.Addr, remote stdNet.Addr) Addresses {
	return NewAddresses(
		NewAddressFromNet(local),
		NewAddressFromNet(remote),
	)
}

func (a *AddressesImpl) Local() Address {
	return a.local
}

func (a *AddressesImpl) Remote() Address {
	return a.remote
}
