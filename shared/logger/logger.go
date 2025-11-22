package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Config holds logger configuration
type Config struct {
	Level      string
	Encoding   string // json or console
	OutputPath string
}

// New creates a new logger with the given configuration
func New(cfg Config) (*zap.Logger, error) {
	// Parse log level
	level := zapcore.InfoLevel
	if cfg.Level != "" {
		if err := level.UnmarshalText([]byte(cfg.Level)); err != nil {
			return nil, err
		}
	}

	// Configure encoder
	encoderConfig := zap.NewProductionEncoderConfig()
	encoderConfig.TimeKey = "timestamp"
	encoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	encoderConfig.EncodeLevel = zapcore.CapitalLevelEncoder

	// Create encoder
	var encoder zapcore.Encoder
	if cfg.Encoding == "console" {
		encoder = zapcore.NewConsoleEncoder(encoderConfig)
	} else {
		encoder = zapcore.NewJSONEncoder(encoderConfig)
	}

	// Create writer
	writer := zapcore.AddSync(os.Stdout)
	if cfg.OutputPath != "" && cfg.OutputPath != "stdout" {
		file, err := os.OpenFile(cfg.OutputPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, err
		}
		writer = zapcore.AddSync(file)
	}

	// Create core
	core := zapcore.NewCore(encoder, writer, level)

	// Create logger
	logger := zap.New(core, zap.AddCaller(), zap.AddStacktrace(zapcore.ErrorLevel))

	return logger, nil
}

// NewProduction creates a production logger
func NewProduction() (*zap.Logger, error) {
	return New(Config{
		Level:      "info",
		Encoding:   "json",
		OutputPath: "stdout",
	})
}

// NewDevelopment creates a development logger
func NewDevelopment() (*zap.Logger, error) {
	return New(Config{
		Level:      "debug",
		Encoding:   "console",
		OutputPath: "stdout",
	})
}
