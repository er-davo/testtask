package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func NewLogger() *zap.Logger {
	var cfg zapcore.EncoderConfig
	var encoder zapcore.Encoder

	cfg = zap.NewDevelopmentEncoderConfig()
	cfg.EncodeTime = zapcore.TimeEncoderOfLayout("15:04:05")

	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	encoder = zapcore.NewConsoleEncoder(cfg)

	core := zapcore.NewCore(
		encoder,
		zapcore.AddSync(os.Stdout),
		zapcore.DebugLevel,
	)

	return zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))
}
