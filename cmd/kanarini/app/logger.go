package app

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
	"flag"
)

type LoggerOptions struct {
	LogLevel    string
	LogEncoding string
}

func BindLoggerFlags(o *LoggerOptions, fs *flag.FlagSet) {
	fs.StringVar(&o.LogLevel, "log-level", "info", `Sets the logger's output level.`)
	fs.StringVar(&o.LogEncoding, "log-encoding", "json", `Sets the logger's encoding. Valid values are "json" and "console".`)
}

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

func LoggerFromOptions(o LoggerOptions) *zap.Logger {
	var levelEnabler zapcore.Level
	switch o.LogLevel {
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
	if o.LogEncoding == "console" {
		logEncoder = zapcore.NewConsoleEncoder
	} else {
		logEncoder = zapcore.NewJSONEncoder
	}
	return Logger(levelEnabler, logEncoder)
}
