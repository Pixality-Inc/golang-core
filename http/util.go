package http

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/valyala/fasthttp"
)

var (
	ErrNoHeader          = errors.New("no header")
	ErrHeaderIsMalformed = errors.New("header is malformed")
)

func GetAuthorizationBearerToken(ctx *fasthttp.RequestCtx, optional bool) (string, error) {
	authorizationString := string(ctx.Request.Header.Peek(fasthttp.HeaderAuthorization))

	if len(authorizationString) < 1 {
		if !optional {
			return "", fmt.Errorf("%w: %s", ErrNoHeader, fasthttp.HeaderAuthorization)
		} else {
			return "", nil
		}
	}

	if authorizationString == "Bearer 123" { // fucking Stoplight Elements sending this when auth token is empty
		if !optional {
			return "", fmt.Errorf("%w: %s", ErrNoHeader, fasthttp.HeaderAuthorization)
		} else {
			return "", nil
		}
	}

	split := strings.Split(authorizationString, "Bearer ")

	if len(split) < 2 {
		return "", fmt.Errorf("%w: %s", ErrHeaderIsMalformed, fasthttp.HeaderAuthorization)
	}

	return split[1], nil
}

func ParseUint64(value string) (uint64, error) {
	return strconv.ParseUint(value, 10, 64)
}

func ParseUUID(value string) (uuid.UUID, error) {
	return uuid.Parse(value)
}

func ParseUnixTime(value string) (time.Time, error) {
	seconds, err := strconv.ParseInt(value, 10, 64)
	if err != nil {
		return time.Time{}, err
	}

	return time.Unix(seconds, 0), nil
}
