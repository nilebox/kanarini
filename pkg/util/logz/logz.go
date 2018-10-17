package logz

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func Logger(level zapcore.Level, encoder func(zapcore.EncoderConfig) zapcore.Encoder) *zap.Logger {
	cfg := zap.NewProductionEncoderConfig()
	cfg.EncodeTime = zapcore.ISO8601TimeEncoder
	cfg.TimeKey = "time"
	lockedSyncer := zapcore.Lock(zapcore.AddSync(os.Stderr))
	return zap.New(
		zapcore.NewCore(
			encoder(cfg),
			lockedSyncer,
			level,
		),
		zap.ErrorOutput(lockedSyncer),
	)
}

func LoggerStr(loggingLevel, logEncoding string) *zap.Logger {
	var levelEnabler zapcore.Level
	switch loggingLevel {
	case "debug":
		levelEnabler = zap.DebugLevel
	case "warn":
		levelEnabler = zap.WarnLevel
	case "error":
		levelEnabler = zap.ErrorLevel
	default:
		levelEnabler = zap.InfoLevel
	}
	var logEncoder func(zapcore.EncoderConfig) zapcore.Encoder
	if logEncoding == "console" {
		logEncoder = zapcore.NewConsoleEncoder
	} else {
		logEncoder = zapcore.NewJSONEncoder
	}
	return Logger(levelEnabler, logEncoder)
}
