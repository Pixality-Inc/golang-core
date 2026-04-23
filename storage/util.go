package storage

import (
	"context"
	"fmt"

	"github.com/pixality-inc/golang-core/logger"
)

func Copy(ctx context.Context, dst Storage, dstFilename string, src Storage, srcFilename string) error {
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
	err := Copy(ctx, dst, dstFilename, src, srcFilename)
	if err != nil {
		return fmt.Errorf("copy file %s to %s: %w", srcFilename, dstFilename, err)
	}

	return src.DeleteFile(ctx, srcFilename)
}
