package logger

import (
	"os"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

// Logger defines the logger interface
type Logger interface {
	Debug(args ...interface{})
	Debugf(format string, args ...interface{})
	Info(args ...interface{})
	Infof(format string, args ...interface{})
	Warn(args ...interface{})
	Warnf(format string, args ...interface{})
	Error(args ...interface{})
	Errorf(format string, args ...interface{})
	Fatal(args ...interface{})
	Fatalf(format string, args ...interface{})
}

// zapLogger implements Logger interface using zap
type zapLogger struct {
	logger *zap.SugaredLogger
}

// NewLogger creates a new logger instance
func NewLogger() Logger {
	format := getLogFormat()

	// Выбрать encoder в зависимости от формата
	var levelEncoder zapcore.LevelEncoder
	if format == "json" {
		levelEncoder = zapcore.CapitalLevelEncoder // Без цветов для JSON
	} else {
		levelEncoder = zapcore.CapitalColorLevelEncoder // С цветами для console
	}

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "time",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		MessageKey:     "msg",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    levelEncoder,
		EncodeTime:     zapcore.TimeEncoderOfLayout("2006-01-02 15:04:05"),
		EncodeDuration: zapcore.StringDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	cfg := zap.Config{
		Level:            zap.NewAtomicLevelAt(getLogLevel()),
		Development:      false,
		Encoding:         format,
		EncoderConfig:    encoderConfig,
		OutputPaths:      []string{"stdout"},
		ErrorOutputPaths: []string{"stderr"},
	}

	logger, err := cfg.Build()
	if err != nil {
		panic(err)
	}

	sugar := logger.Sugar()

	return &zapLogger{
		logger: sugar,
	}
}

// getLogLevel returns the log level based on the environment
func getLogLevel() zapcore.Level {
	level := os.Getenv("LogLevel")
	switch level {
	case "debug":
		return zap.DebugLevel
	case "info":
		return zap.InfoLevel
	case "warn":
		return zap.WarnLevel
	case "error":
		return zap.ErrorLevel
	default:
		return zap.InfoLevel
	}
}

// getLogFormat returns the log format based on the environment
func getLogFormat() string {
	format := os.Getenv("LogFormat")
	if format == "json" {
		return "json"
	}
	return "console"
}

// Debug logs a debug message
func (l *zapLogger) Debug(args ...interface{}) {
	l.logger.Debugf("%#v", args...)
}

// Debugf logs a formatted debug message
func (l *zapLogger) Debugf(format string, args ...interface{}) {
	l.logger.Debugf(format, args...)
}

// Info logs an info message
func (l *zapLogger) Info(args ...interface{}) {
	l.logger.Infof("%#v", args...)
}

// Infof logs a formatted info message
func (l *zapLogger) Infof(format string, args ...interface{}) {
	l.logger.Infof(format, args...)
}

// Warn logs a warning message
func (l *zapLogger) Warn(args ...interface{}) {
	l.logger.Warnf("%#v", args...)
}

// Warnf logs a formatted warning message
func (l *zapLogger) Warnf(format string, args ...interface{}) {
	l.logger.Warnf(format, args...)
}

// Error logs an error message
func (l *zapLogger) Error(args ...interface{}) {
	l.logger.Errorf("%#v", args...)
}

// Errorf logs a formatted error message
func (l *zapLogger) Errorf(format string, args ...interface{}) {
	l.logger.Errorf(format, args...)
}

// Fatal logs a fatal message and terminates the program
func (l *zapLogger) Fatal(args ...interface{}) {
	l.logger.Fatalf("%#v", args...)
}

// Fatalf logs a formatted fatal message and terminates the program
func (l *zapLogger) Fatalf(format string, args ...interface{}) {
	l.logger.Fatalf(format, args...)
}
