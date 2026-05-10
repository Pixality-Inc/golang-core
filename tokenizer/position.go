package tokenizer

type Position interface {
	Offset() uint64
	Column() uint64
	Line() uint64
}

type PositionImpl struct {
	offset uint64
	column uint64
	line   uint64
}

func NewPosition(offset uint64, column uint64, line uint64) Position {
	return &PositionImpl{
		offset: offset,
		column: column,
		line:   line,
	}
}

func (p *PositionImpl) Offset() uint64 {
	return p.offset
}

func (p *PositionImpl) Column() uint64 {
	return p.column
}

func (p *PositionImpl) Line() uint64 {
	return p.line
}
