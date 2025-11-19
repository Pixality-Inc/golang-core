package util

func TernaryFunc[R any](whatIf bool, then func() R, other func() R) R {
	if whatIf {
		return then()
	} else {
		return other()
	}
}

func Ternary[T any](whatIf bool, then T, other T) T {
	if whatIf {
		return then
	} else {
		return other
	}
}
