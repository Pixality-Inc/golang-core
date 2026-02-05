package proto_parser

type Entity interface {
	FileId() int
	Package() string
	Path() []string
	Name() string
	FullName() string
}
