package either

type EitherError[T any] = Either[error, T]

func Error[R any](err error) Either[error, R] {
	return Left[error, R](err)
}

func RightError[R any](value R) Either[error, R] {
	return Right[error, R](value)
}
