package logger

import (
	"context"

	"github.com/sirupsen/logrus"
)

const ctxLogger = "logger"

func New(ctx context.Context) logrus.FieldLogger {
	defaultLogger := logrus.StandardLogger()

	if ctx == nil {
		return defaultLogger
	}

	lgr := ctx.Value(ctxLogger)

	logger, ok := lgr.(logrus.FieldLogger)
	if !ok {
		return defaultLogger
	}

	return logger
}
