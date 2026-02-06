package proto_parser

type State struct {
	models map[string]Model
	enums  map[string]Enum
}

func NewState() *State {
	return &State{
		models: make(map[string]Model),
		enums:  make(map[string]Enum),
	}
}
