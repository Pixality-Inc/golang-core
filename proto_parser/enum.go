package proto_parser

import "strings"

type Enum interface {
	FileId() int
	Package() string
	Path() []string
	Name() string
	FullName() string
	Entries() []EnumEntry
}

type EnumImpl struct {
	fileId  int
	pkg     string
	path    []string
	name    string
	entries []EnumEntry
}

func NewEnum(
	fileId int,
	pkg string,
	path []string,
	name string,
	entries []EnumEntry,
) Enum {
	return &EnumImpl{
		fileId:  fileId,
		pkg:     pkg,
		path:    path,
		name:    name,
		entries: entries,
	}
}

func (e *EnumImpl) FileId() int {
	return e.fileId
}

func (e *EnumImpl) Package() string {
	return e.pkg
}

func (e *EnumImpl) Path() []string {
	return e.path
}

func (e *EnumImpl) Name() string {
	return e.name
}

func (e *EnumImpl) FullName() string {
	if len(e.path) > 0 {
		return strings.Join(e.path, "__") + "__" + e.Name()
	} else {
		return e.Name()
	}
}

func (e *EnumImpl) Entries() []EnumEntry {
	return e.entries
}
