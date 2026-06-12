package storage

import (
	"context"
	"fmt"
	"reflect"

	"github.com/pixality-inc/golang-core/logger"
)

// sameStorage reports whether dst and src are the same storage instance, so the
// caller can take the backend-native copy/move fast path. Comparing interface
// values with == panics when their (shared) concrete type is not comparable —
// e.g. a struct holding a map or slice — so guard on Comparable and otherwise
// fall back to streaming rather than crash a public caller.
func sameStorage(dst, src Storage) bool {
	t := reflect.TypeOf(dst)
	if t == nil || t != reflect.TypeOf(src) || !t.Comparable() {
		return false
	}

	return dst == src
}

func Copy(ctx context.Context, dst Storage, dstFilename string, src Storage, srcFilename string) error {
	// same storage: use the backend-native server-side copy instead of
	// streaming the object through the application
	if sameStorage(dst, src) {
		err := dst.Copy(ctx, srcFilename, dstFilename)
		if err == nil {
			return nil
		}

		logger.GetLogger(ctx).WithError(err).Warnf(
			"native storage copy failed from %s to %s, falling back to streaming copy",
			srcFilename,
			dstFilename,
		)
	}

	return copyStreaming(ctx, dst, dstFilename, src, srcFilename)
}

func copyStreaming(ctx context.Context, dst Storage, dstFilename string, src Storage, srcFilename string) error {
	log := logger.GetLogger(ctx)

	srcFile, err := src.ReadFile(ctx, srcFilename)
	if err != nil {
		return fmt.Errorf("read source file %s: %w", srcFilename, err)
	}

	defer func() {
		if fErr := srcFile.Close(); fErr != nil {
			log.WithError(fErr).Errorf("failed to close source file %s", srcFilename)
		}
	}()

	return dst.Write(
		ctx,
		dstFilename,
		srcFile,
	)
}

func Move(ctx context.Context, dst Storage, dstFilename string, src Storage, srcFilename string) error {
	// same storage: use the backend-native server-side move
	if sameStorage(dst, src) {
		return dst.Move(ctx, srcFilename, dstFilename)
	}

	err := Copy(ctx, dst, dstFilename, src, srcFilename)
	if err != nil {
		return fmt.Errorf("copy file %s to %s: %w", srcFilename, dstFilename, err)
	}

	return src.DeleteFile(ctx, srcFilename)
}
