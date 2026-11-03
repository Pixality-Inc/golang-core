package json

import (
	"github.com/goccy/go-json"
)

type (
	RawMessage = json.RawMessage
	// Decoder reads a stream of json values; Token walks an object member by member, the way a
	// check of the raw member names needs (Unmarshal into a map drops duplicates, into a struct it
	// matches names ignoring case)
	Decoder = json.Decoder
	Token   = json.Token
	Delim   = json.Delim
)

var (
	Marshal    = json.Marshal
	Unmarshal  = json.Unmarshal
	Valid      = json.Valid
	NewDecoder = json.NewDecoder
)
