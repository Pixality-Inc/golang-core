package json

import (
	"github.com/goccy/go-json"
)

type RawMessage = json.RawMessage

var (
	Marshal   = json.Marshal
	Unmarshal = json.Unmarshal
	Valid     = json.Valid
)
