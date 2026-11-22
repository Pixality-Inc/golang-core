package storage

import (
	"errors"
	"io/fs"
	"net/http"

	"github.com/minio/minio-go/v7"
	"google.golang.org/api/googleapi"
)

// IsRejected reports whether err says the backend refused the request itself rather than failed to
// serve it, whichever provider produced it: the credentials or a policy deny it (401, 403), or a
// precondition of it does not hold (412 - R2 answered every read of a gzipped object with it until
// v0.8.23). Like a missing object (IsNotFound) it answers the same on every attempt, so a caller that
// retries storage calls gives up on it at once instead of spending a ladder on it.
//
// A 400 deliberately does not count: S3 answers a client too slow to send its body with
// 400 RequestTimeout, which is worth waiting out. Neither do 408 and 429.
//
// The s3 status is read with errors.As for the reason given at IsNotFound.
func IsRejected(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrPermission) {
		return true
	}

	var response minio.ErrorResponse
	if errors.As(err, &response) {
		return isRejectedStatus(response.StatusCode)
	}

	var apiErr *googleapi.Error
	if errors.As(err, &apiErr) {
		return isRejectedStatus(apiErr.Code)
	}

	return false
}

func isRejectedStatus(status int) bool {
	return status == http.StatusUnauthorized ||
		status == http.StatusForbidden ||
		status == http.StatusPreconditionFailed
}
