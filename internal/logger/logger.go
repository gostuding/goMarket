package logger

import (
	"fmt"

	"go.uber.org/zap"
)

func NewLogger() (*zap.SugaredLogger, error) {
	logger, err := zap.NewDevelopment()
	if err != nil {
		return nil, fmt.Errorf("logger init error: %w", err)
	}
	return logger.Sugar(), nil
}
