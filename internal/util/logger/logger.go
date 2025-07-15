package logger

import (
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var Log *zap.Logger

func Init(level string) error {
	logLevel := zapcore.InfoLevel
	if err := logLevel.UnmarshalText([]byte(level)); err != nil {
		return err
	}

	config := zap.NewProductionConfig()
	config.Level = zap.NewAtomicLevelAt(logLevel)
	config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder

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
