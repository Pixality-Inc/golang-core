package util

import (
	"os"

	"github.com/pixality-inc/golang-core/logger"
)

func FileExists(filename string) (os.FileInfo, bool) {
	fileInfo, err := os.Stat(filename)

	return fileInfo, err == nil
}

func CopyFile(source string, dest string) error {
	r, err := os.Open(source)
	if err != nil {
		return err
	}

	defer func() {
		fErr := r.Close()
		if fErr != nil {
			logger.GetLoggerWithoutContext().WithError(err).Errorf("failed to close file %s", source)
		}
	}()

	w, err := os.Create(dest)
	if err != nil {
		return err
	}

	defer func() {
		fErr := w.Close()
		if fErr != nil {
			logger.GetLoggerWithoutContext().WithError(err).Errorf("failed to close file %s", dest)
		}
	}()

	if _, err = w.ReadFrom(r); err != nil {
		return err
	}

	return nil
}
