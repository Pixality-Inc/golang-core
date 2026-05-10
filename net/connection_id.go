package net

import "github.com/google/uuid"

type ConnectionId uuid.UUID

func NewConnectionId() ConnectionId {
	return NewConnectionIdFromUUID(uuid.New())
}

func NewConnectionIdFromUUID(uuid uuid.UUID) ConnectionId {
	return ConnectionId(uuid)
}

func (c *ConnectionId) UUID() uuid.UUID {
	return uuid.UUID(*c)
}

func (c *ConnectionId) String() string {
	return c.UUID().String()
}

func ParseConnectionId(value string) (ConnectionId, error) {
	id, err := uuid.Parse(value)
	if err != nil {
		return ConnectionId{}, err
	}

	return ConnectionId(id), nil
}
