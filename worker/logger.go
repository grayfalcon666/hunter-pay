package worker

import (
	"fmt"
	"log/slog"
)

type Logger struct{}

func NewLogger() *Logger {
	return &Logger{}
}

func (logger *Logger) Print(level slog.Level, args ...interface{}) {
	switch level {
	case slog.LevelDebug:
		slog.Debug(fmt.Sprint(args...))
	case slog.LevelInfo:
		slog.Info(fmt.Sprint(args...))
	case slog.LevelWarn:
		slog.Warn(fmt.Sprint(args...))
	case slog.LevelError:
		slog.Error(fmt.Sprint(args...))
	default:
		slog.Info("not known level", "level", level, "args", args)
	}
}

func (logger *Logger) Debug(args ...interface{}) {
	logger.Print(slog.LevelDebug, args...)
}

func (logger *Logger) Info(args ...interface{}) {
	logger.Print(slog.LevelInfo, args...)
}

func (logger *Logger) Warn(args ...interface{}) {
	logger.Print(slog.LevelWarn, args...)
}

func (logger *Logger) Error(args ...interface{}) {
	logger.Print(slog.LevelError, args...)
}

// slog没有fatal级别 我暂时用error替代
func (logger *Logger) Fatal(args ...interface{}) {
	logger.Print(slog.LevelError, args...)
}
