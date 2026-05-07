package either

type Either[L any, R any] interface {
	Left() L
	Right() R
	IsLeft() bool
	IsRight() bool
	Value() (R, L)
}
type Impl[L any, R any] struct {
	left  *L
	right *R
}

func Left[L any, R any](left L) Either[L, R] {
	return &Impl[L, R]{
		left:  &left,
		right: nil,
	}
}

func Right[L any, R any](right R) Either[L, R] {
	return &Impl[L, R]{
		left:  nil,
		right: &right,
	}
}

func (e *Impl[L, R]) Left() L {
	return *e.left
}

func (e *Impl[L, R]) Right() R {
	return *e.right
}

func (e *Impl[L, R]) IsLeft() bool {
	return e.left != nil
}

func (e *Impl[L, R]) IsRight() bool {
	return e.right != nil
}

func (e *Impl[L, R]) Value() (R, L) {
	var (
		leftDefault  L
		rightDefault R
	)

	if e.IsLeft() {
		return rightDefault, e.Left()
	} else if e.IsRight() {
		return e.Right(), leftDefault
	}

	return rightDefault, leftDefault
}
