package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func Init(level string) error {
	logLevel := zapcore.DebugLevel
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		return err
	}

	config := zap.NewDevelopmentConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	config.OutputPaths = []string{"stdout"}
	config.ErrorOutputPaths = []string{"stderr"}

	var err error
	Log, err = config.Build()
	if err != nil {
		return err
	}

	return nil
}

func Sync() error {
	if Log != nil {
		return Log.Sync()
	}
	return nil
}
