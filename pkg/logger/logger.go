package logger

import (
	"sync"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type Logger struct {
	*zap.SugaredLogger
}

var (
	globalLogger *Logger
	globalMu     sync.RWMutex
)

func NewLogger(debug bool) *Logger {
	var config zap.Config

	if debug {
		config = zap.NewDevelopmentConfig()
		config.EncoderConfig.EncodeLevel = zapcore.CapitalColorLevelEncoder
		// 在开发模式下，禁用时间戳和调用者信息，使输出更简洁
		config.EncoderConfig.TimeKey = ""
		config.EncoderConfig.CallerKey = ""
	} else {
		config = zap.NewProductionConfig()
		config.EncoderConfig.TimeKey = "timestamp"
		config.EncoderConfig.EncodeTime = zapcore.ISO8601TimeEncoder
	}

	logger, err := config.Build()
	if err != nil {
		panic(err)
	}

	return &Logger{
		SugaredLogger: logger.Sugar(),
	}
}

// SetGlobal sets the process-wide logger used by utility/client packages.
func SetGlobal(l *Logger) {
	globalMu.Lock()
	defer globalMu.Unlock()
	globalLogger = l
}

// L returns the process-wide logger. Falls back to a default non-debug logger.
func L() *Logger {
	globalMu.RLock()
	if globalLogger != nil {
		l := globalLogger
		globalMu.RUnlock()
		return l
	}
	globalMu.RUnlock()

	globalMu.Lock()
	defer globalMu.Unlock()
	if globalLogger == nil {
		globalLogger = NewLogger(false)
	}
	return globalLogger
}
