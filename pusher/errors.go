package pusher

import "errors"

// ErrTemporary marks a provider failure that may succeed when repeated later: a network error,
// a timeout, an unresolvable host, a 5xx, 408 or 429 response of the provider. errors that stay the
// same on every attempt (bad credentials, invalid arguments, a malformed url, a tls failure) do not
// carry it.
// it says nothing about whether a repeat is safe: a timed out request may have reached the
// provider, so only idempotent calls (token registration) are safe to repeat blindly, a repeated
// send may deliver the message twice
var ErrTemporary = errors.New("temporary pusher failure")
