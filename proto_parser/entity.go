package proto_parser

import "strings"

type Entity interface {
	FileId() int
	Package() string
	Path() []string
	Name() string
}

func GetFullName(entity Entity, pathSeparator string) string {
	path := entity.Path()

	if len(path) > 0 {
		return strings.Join(path, pathSeparator) + pathSeparator + entity.Name()
	} else {
		return entity.Name()
	}
}
