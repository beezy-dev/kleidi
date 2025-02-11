package logger

import (
	//"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func CreateLogger(debug bool) *zap.Logger {
	encoderCfg := zap.NewProductionEncoderConfig()
	encoderCfg.TimeKey = "timestamp"
	encoderCfg.EncodeTime = zapcore.ISO8601TimeEncoder
	level := zap.InfoLevel

	if debug {
		level = zap.DebugLevel
	}
	config := zap.Config{
		Level:             zap.NewAtomicLevelAt(level),
		Development:       debug,
		DisableCaller:     false,
		DisableStacktrace: false,
		Sampling:          nil,
		Encoding:          "console",
		EncoderConfig:     encoderCfg,
		OutputPaths: []string{
			"stderr",
		},
		ErrorOutputPaths: []string{
			"stderr",
		},
		InitialFields: map[string]interface{}{
			//"pid": os.Getpid(),
		},
	}

	return zap.Must(config.Build())
}
