package storage

import (
	"errors"
	"io/fs"

	gcs "cloud.google.com/go/storage"
	"github.com/minio/minio-go/v7"
)

// notFoundCode is what some S3 backends answer with instead of NoSuchKey, and what a HEAD
// returns when there is no body to carry a code. minio-go exports no constant for it.
const notFoundCode = "NotFound"

// IsNotFound reports whether err says the object does not exist, whichever provider produced
// it. It exists for callers that retry a storage call: a missing object answers the same on
// every attempt, so retrying it spends the whole ladder to fail exactly as it failed the
// first time, while the other backend failures are the ones worth waiting out.
//
// The providers speak different dialects - the os provider returns fs.ErrNotExist, gcs its
// own sentinel, s3 an opaque minio error - and all of them wrap with %w, so errors.Is and
// errors.As see through the wrapping. minio.ToErrorResponse does not: it type switches
// without unwrapping and answers a wrapped error with a zero value.
//
// A bare 404 deliberately does not count on its own: minio-go labels a bodyless 404 as
// NoSuchBucket whenever the request carries no object name, which is every listing, and a
// gateway answering a listing with 404 is a failure to wait out rather than an empty folder.
// A caller that knows it asked about one object (s3.FileExists on a HEAD) reads 404 broadly
// on its own.
func IsNotFound(err error) bool {
	if err == nil {
		return false
	}

	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, gcs.ErrObjectNotExist) {
		return true
	}

	var response minio.ErrorResponse
	if !errors.As(err, &response) {
		return false
	}

	return response.Code == minio.NoSuchKey ||
		response.Code == minio.NoSuchUpload ||
		response.Code == notFoundCode
}
