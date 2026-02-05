package proto_parser

type Results struct {
	Models map[string]Model
	Enums  map[string]Enum
}

func NewResult(
	models map[string]Model,
	enums map[string]Enum,
) *Results {
	return &Results{
		Models: models,
		Enums:  enums,
	}
}
