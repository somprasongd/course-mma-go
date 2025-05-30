package logger

import (
	"go.elastic.co/ecszap"
	"go.uber.org/zap"
)

var Log *zap.Logger

func Init() {
	config := zap.NewDevelopmentConfig()
	config.EncoderConfig = ecszap.ECSCompatibleEncoderConfig(config.EncoderConfig)

	var err error
	Log, err = config.Build(ecszap.WrapCoreOption())

	if err != nil {
		panic(err)
	}
}
