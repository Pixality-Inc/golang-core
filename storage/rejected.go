package storage

import (
	"errors"
	"io/fs"
	"net/http"

	"github.com/minio/minio-go/v7"
	"google.golang.org/api/googleapi"
)

// IsRejected reports whether err says the backend refused the request itself rather than failed to
// serve it, whichever provider produced it: the credentials or a policy deny it (401, 403, a file
// the os provider may not open), or a precondition of it does not hold (412). It exists for the
// same callers as IsNotFound: a refusal is not worth a retry ladder.
//
// A 412 is not always permanent: the s3 provider stats an object before reading it and minio-go
// reads it with If-Match of the etag the stat saw, so an object rewritten between the two fails
// once and passes on a fresh read. It still counts, because a caller reading objects written once
// meets the deterministic one instead - R2 compressed plain json and txt on the fly and answered
// every such read with 412 until the client stopped asking it for gzip
// (WithForceGzipAcceptEncoding) - and a retry ladder only delays that failure by its whole budget.
//
// A 400 does not count: S3 answers a client too slow to send its body with 400 RequestTimeout,
// which is worth waiting out. Neither do 408 and 429, nor a 403 RequestTimeTooSkewed, which a node
// with a drifting clock gets and a later attempt does not. A permission error of a socket
// (connect: operation not permitted, a dropped connection) is a network failure, not a refusal.
//
// The s3 status is read with errors.AsType for the reason given at IsNotFound.
func IsRejected(err error) bool {
	if err == nil {
		return false
	}

	if response, ok := errors.AsType[minio.ErrorResponse](err); ok {
		return response.Code != minio.RequestTimeTooSkewed && isRejectedStatus(response.StatusCode)
	}

	if apiErr, ok := errors.AsType[*googleapi.Error](err); ok {
		return isRejectedStatus(apiErr.Code)
	}

	// the os provider fails an open or a create with a path error, while a socket fails with an
	// op error around the same errno
	pathErr, ok := errors.AsType[*fs.PathError](err)

	return ok && errors.Is(pathErr.Err, fs.ErrPermission)
}

func isRejectedStatus(status int) bool {
	return status == http.StatusUnauthorized ||
		status == http.StatusForbidden ||
		status == http.StatusPreconditionFailed
}
