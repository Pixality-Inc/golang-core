package http_client

type Response struct {
	StatusCode int
	Headers    Headers
	Body       []byte
}

type TypedResponse[OUT any] struct {
	StatusCode int
	Headers    Headers
	Body       []byte
	Entity     OUT
}
