package http_client

type Request struct {
	QueryParams QueryParams
	Headers     Headers
}

func NewRequest() *Request {
	return &Request{
		QueryParams: make(QueryParams),
		Headers:     make(Headers),
	}
}
