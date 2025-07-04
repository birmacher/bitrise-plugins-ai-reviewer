package logger

import (
	"os"
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

var (
	// Global logger instance
	logger *zap.Logger
	// Global sugared logger instance
	sugar *zap.SugaredLogger
	// Ensure initialization happens only once
	once sync.Once
)

// Init initializes the logger with the given log level
// Valid levels: debug, info, warn, error, dpanic, panic, fatal
func Init(level string) {
	once.Do(func() {
		// Parse log level
		var zapLevel zapcore.Level
		if err := zapLevel.UnmarshalText([]byte(level)); err != nil {
			zapLevel = zap.InfoLevel // Default to info level
		}

		// Create encoder config
		encoderConfig := zapcore.EncoderConfig{
			TimeKey:        "ts",
			LevelKey:       "level",
			NameKey:        "logger",
			CallerKey:      "caller",
			FunctionKey:    zapcore.OmitKey,
			MessageKey:     "msg",
			StacktraceKey:  "stacktrace",
			LineEnding:     zapcore.DefaultLineEnding,
			EncodeLevel:    zapcore.CapitalLevelEncoder,
			EncodeTime:     zapcore.ISO8601TimeEncoder,
			EncodeDuration: zapcore.SecondsDurationEncoder,
			EncodeCaller:   zapcore.ShortCallerEncoder,
		}

		// Create core
		core := zapcore.NewCore(
			zapcore.NewJSONEncoder(encoderConfig),
			zapcore.AddSync(os.Stdout),
			zapLevel,
		)

		// Create logger
		logger = zap.New(core, zap.AddCaller(), zap.AddCallerSkip(1))
		sugar = logger.Sugar()
	})
}

// Sugar returns the global sugared logger
func Sugar() *zap.SugaredLogger {
	if sugar == nil {
		// If logger hasn't been initialized, initialize with info level
		Init("info")
	}
	return sugar
}

// GetLogger returns the global zap logger
func GetLogger() *zap.Logger {
	if logger == nil {
		// If logger hasn't been initialized, initialize with info level
		Init("info")
	}
	return logger
}

// Sync flushes any buffered log entries
func Sync() {
	if logger != nil {
		_ = logger.Sync()
	}
}

// Debug logs a message at debug level
func Debug(args ...interface{}) {
	Sugar().Debug(args...)
}

// Info logs a message at info level
func Info(args ...interface{}) {
	Sugar().Info(args...)
}

// Warn logs a message at warn level
func Warn(args ...interface{}) {
	Sugar().Warn(args...)
}

// Error logs a message at error level
func Error(args ...interface{}) {
	Sugar().Error(args...)
}

// Fatal logs a message at fatal level and then calls os.Exit(1)
func Fatal(args ...interface{}) {
	Sugar().Fatal(args...)
}

// Debugf logs a formatted message at debug level
func Debugf(template string, args ...interface{}) {
	Sugar().Debugf(template, args...)
}

// Infof logs a formatted message at info level
func Infof(template string, args ...interface{}) {
	Sugar().Infof(template, args...)
}

// Warnf logs a formatted message at warn level
func Warnf(template string, args ...interface{}) {
	Sugar().Warnf(template, args...)
}

// Errorf logs a formatted message at error level
func Errorf(template string, args ...interface{}) {
	Sugar().Errorf(template, args...)
}

// Fatalf logs a formatted message at fatal level and then calls os.Exit(1)
func Fatalf(template string, args ...interface{}) {
	Sugar().Fatalf(template, args...)
}
