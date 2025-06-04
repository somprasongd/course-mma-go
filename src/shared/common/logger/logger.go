package logger

import (
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
)

type closeLog func() error

var Log *zap.Logger

func Init() (closeLog, error) {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig = ecszap.ECSCompatibleEncoderConfig(config.EncoderConfig)

	var err error
	Log, err = config.Build(ecszap.WrapCoreOption())

	if err != nil {
		return nil, err
	}

	return func() error {
		return Log.Sync()
	}, nil
}

func With(fields ...zap.Field) *zap.Logger {
	return Log.With(fields...)
}
