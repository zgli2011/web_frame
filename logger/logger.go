package logger

import (
	"context"
	"fmt"
	"io"
	"os"
	"sync"
)

// ============================================================================
// StandardLogger Implementation
// ============================================================================

type StandardLogger struct {
	level     Level
	formatter Formatter
	writer    io.Writer
	fields    map[string]interface{}
	mu        sync.RWMutex
	localIP   string
}

func NewLogger(config Config) (*StandardLogger, error) {
	level, exists := ParseLevel(config.Level)
	if !exists {
		level = INFO
	}

	var formatter Formatter
	switch Format(config.Format) {
	case FormatJSON:
		formatter = NewJSONFormatter()
	case FormatText:
		formatter = NewTextFormatter()
	default:
		formatter = NewTextFormatter()
	}

	var writer io.Writer
	switch Output(config.Output) {
	case OutputConsole:
		writer = os.Stdout
	case OutputFile:
		if config.FilePath == "" {
			config.FilePath = "./logs"
		}
		if config.FileName == "" {
			config.FileName = "app.log"
		}
		if config.MaxSize <= 0 {
			config.MaxSize = 100
		}
		if config.MaxBackups <= 0 {
			config.MaxBackups = 7
		}

		fileWriter, err := NewRotatingFileWriter(
			config.FilePath,
			config.FileName,
			config.MaxSize,
			config.MaxBackups,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %v", err)
		}
		writer = fileWriter

	case OutputBoth:
		if config.FilePath == "" {
			config.FilePath = "./logs"
		}
		if config.FileName == "" {
			config.FileName = "app.log"
		}
		if config.MaxSize <= 0 {
			config.MaxSize = 100
		}
		if config.MaxBackups <= 0 {
			config.MaxBackups = 7
		}

		fileWriter, err := NewRotatingFileWriter(
			config.FilePath,
			config.FileName,
			config.MaxSize,
			config.MaxBackups,
		)
		if err != nil {
			return nil, fmt.Errorf("failed to create file writer: %v", err)
		}
		writer = NewMultiWriter(os.Stdout, fileWriter)

	default:
		writer = os.Stdout
	}

	logger := &StandardLogger{
		level:     level,
		formatter: formatter,
		writer:    writer,
		fields:    make(map[string]interface{}),
		localIP:   getLocalIP(),
	}

	return logger, nil
}

func (l *StandardLogger) log(ctx context.Context, level Level, msg string, fields ...Field) {
	l.mu.RLock()
	if level < l.level {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	entry := &Entry{
		Time:    formatTime(),
		Level:   level.String(),
		PID:     getPID(),
		TraceID: getTraceID(ctx),
		File:    getCallerInfo(1),
		Message: msg,
		IP:      l.localIP,
		Fields:  make(map[string]interface{}),
	}

	l.mu.RLock()
	for k, v := range l.fields {
		entry.Fields[k] = v
	}
	l.mu.RUnlock()

	for _, field := range fields {
		entry.Fields[field.Key] = field.Value
	}

	if len(entry.Fields) == 0 {
		entry.Fields = nil
	}

	formatted, err := l.formatter.Format(entry)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Failed to format log entry: %v\n", err)
		return
	}

	l.writer.Write([]byte(formatted))

	if level == FATAL {
		os.Exit(1)
	}
}

func (l *StandardLogger) Debug(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, DEBUG, msg, fields...)
}

func (l *StandardLogger) Info(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, INFO, msg, fields...)
}

func (l *StandardLogger) Warn(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, WARN, msg, fields...)
}

func (l *StandardLogger) Error(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, ERROR, msg, fields...)
}

func (l *StandardLogger) Fatal(ctx context.Context, msg string, fields ...Field) {
	l.log(ctx, FATAL, msg, fields...)
}

func (l *StandardLogger) DebugF(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, DEBUG, fmt.Sprintf(format, args...))
}

func (l *StandardLogger) InfoF(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, INFO, fmt.Sprintf(format, args...))
}

func (l *StandardLogger) WarnF(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, WARN, fmt.Sprintf(format, args...))
}

func (l *StandardLogger) ErrorF(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, ERROR, fmt.Sprintf(format, args...))
}

func (l *StandardLogger) FatalF(ctx context.Context, format string, args ...interface{}) {
	l.log(ctx, FATAL, fmt.Sprintf(format, args...))
}

func (l *StandardLogger) WithField(key string, value interface{}) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &StandardLogger{
		level:     l.level,
		formatter: l.formatter,
		writer:    l.writer,
		fields:    make(map[string]interface{}),
		localIP:   l.localIP,
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}
	newLogger.fields[key] = value

	return newLogger
}

func (l *StandardLogger) WithFields(fields map[string]interface{}) Logger {
	l.mu.Lock()
	defer l.mu.Unlock()

	newLogger := &StandardLogger{
		level:     l.level,
		formatter: l.formatter,
		writer:    l.writer,
		fields:    make(map[string]interface{}),
		localIP:   l.localIP,
	}

	for k, v := range l.fields {
		newLogger.fields[k] = v
	}

	for k, v := range fields {
		newLogger.fields[k] = v
	}

	return newLogger
}

func (l *StandardLogger) SetLevel(level Level) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

func (l *StandardLogger) SetOutput(writer io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.writer = writer
}

// ============================================================================
// Global Logger Functions
// ============================================================================

var (
	globalLogger Logger
	globalMutex  sync.RWMutex
)

func SetGlobalLogger(logger Logger) {
	globalMutex.Lock()
	defer globalMutex.Unlock()
	globalLogger = logger
}

func GetGlobalLogger() Logger {
	globalMutex.RLock()
	defer globalMutex.RUnlock()
	return globalLogger
}

// Simple global functions (most commonly used)
func Debug(msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Debug(context.Background(), msg, fields...)
	}
}

func Info(msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Info(context.Background(), msg, fields...)
	}
}

func Warn(msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Warn(context.Background(), msg, fields...)
	}
}

func Error(msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Error(context.Background(), msg, fields...)
	}
}

func Fatal(msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Fatal(context.Background(), msg, fields...)
	}
}

// Formatted global functions
func DebugF(format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.DebugF(context.Background(), format, args...)
	}
}

func InfoF(format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.InfoF(context.Background(), format, args...)
	}
}

func WarnF(format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.WarnF(context.Background(), format, args...)
	}
}

func ErrorF(format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.ErrorF(context.Background(), format, args...)
	}
}

func FatalF(format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.FatalF(context.Background(), format, args...)
	}
}

// Field functions
func WithField(key string, value interface{}) Logger {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		return logger.WithField(key, value)
	}
	return nil
}

func WithFields(fields map[string]interface{}) Logger {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		return logger.WithFields(fields)
	}
	return nil
}

// Context-aware global functions (for trace_id support)
func DebugCtx(ctx context.Context, msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Debug(ctx, msg, fields...)
	}
}

func InfoCtx(ctx context.Context, msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Info(ctx, msg, fields...)
	}
}

func WarnCtx(ctx context.Context, msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Warn(ctx, msg, fields...)
	}
}

func ErrorCtx(ctx context.Context, msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Error(ctx, msg, fields...)
	}
}

func FatalCtx(ctx context.Context, msg string, fields ...Field) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.Fatal(ctx, msg, fields...)
	}
}

// Context-aware formatted functions
func DebugFCtx(ctx context.Context, format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.DebugF(ctx, format, args...)
	}
}

func InfoFCtx(ctx context.Context, format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.InfoF(ctx, format, args...)
	}
}

func WarnFCtx(ctx context.Context, format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.WarnF(ctx, format, args...)
	}
}

func ErrorFCtx(ctx context.Context, format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.ErrorF(ctx, format, args...)
	}
}

func FatalFCtx(ctx context.Context, format string, args ...interface{}) {
	globalMutex.RLock()
	logger := globalLogger
	globalMutex.RUnlock()
	if logger != nil {
		logger.FatalF(ctx, format, args...)
	}
}