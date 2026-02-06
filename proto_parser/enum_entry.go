package proto_parser

type EnumEntry interface {
	Name() string
	Value() int
	Comment() string
}

type EnumEntryImpl struct {
	name    string
	value   int
	comment string
}

func NewEnumEntry(name string, value int, comment string) EnumEntry {
	return &EnumEntryImpl{
		name:    name,
		value:   value,
		comment: comment,
	}
}

func (e *EnumEntryImpl) Name() string {
	return e.name
}

func (e *EnumEntryImpl) Value() int {
	return e.value
}

func (e *EnumEntryImpl) Comment() string {
	return e.comment
}
