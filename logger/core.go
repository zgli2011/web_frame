package logger

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"time"
)

// ============================================================================
// Types and Constants
// ============================================================================

type Level int

const (
	DEBUG Level = iota
	INFO
	WARN
	ERROR
	FATAL
)

var levelNames = map[Level]string{
	DEBUG: "DEBUG",
	INFO:  "INFO",
	WARN:  "WARN",
	ERROR: "ERROR",
	FATAL: "FATAL",
}

var levelMap = map[string]Level{
	"DEBUG": DEBUG,
	"INFO":  INFO,
	"WARN":  WARN,
	"ERROR": ERROR,
	"FATAL": FATAL,
}

func (l Level) String() string {
	if name, exists := levelNames[l]; exists {
		return name
	}
	return "UNKNOWN"
}

func ParseLevel(level string) (Level, bool) {
	l, exists := levelMap[level]
	return l, exists
}

type Output string

const (
	OutputConsole Output = "console"
	OutputFile    Output = "file"
	OutputBoth    Output = "both"
)

type Format string

const (
	FormatText Format = "text"
	FormatJSON Format = "json"
)

type Config struct {
	Level      string `yaml:"level"`
	Output     string `yaml:"output"`
	Format     string `yaml:"format"`
	FilePath   string `yaml:"file_path"`
	FileName   string `yaml:"file_name"`
	MaxSize    int    `yaml:"max_size"`    // MB
	MaxBackups int    `yaml:"max_backups"`
}

type Logger interface {
	Debug(ctx context.Context, msg string, fields ...Field)
	Info(ctx context.Context, msg string, fields ...Field)
	Warn(ctx context.Context, msg string, fields ...Field)
	Error(ctx context.Context, msg string, fields ...Field)
	Fatal(ctx context.Context, msg string, fields ...Field)

	DebugF(ctx context.Context, format string, args ...interface{})
	InfoF(ctx context.Context, format string, args ...interface{})
	WarnF(ctx context.Context, format string, args ...interface{})
	ErrorF(ctx context.Context, format string, args ...interface{})
	FatalF(ctx context.Context, format string, args ...interface{})

	WithField(key string, value interface{}) Logger
	WithFields(fields map[string]interface{}) Logger

	SetLevel(level Level)
	SetOutput(writer io.Writer)
}

type Field struct {
	Key   string
	Value interface{}
}

type Entry struct {
	Time    string                 `json:"time"`
	Level   string                 `json:"level"`
	PID     int                    `json:"pid"`
	TraceID string                 `json:"trace_id"`
	File    string                 `json:"file"`
	Message string                 `json:"message"`
	IP      string                 `json:"ip,omitempty"`
	Fields  map[string]interface{} `json:"fields,omitempty"`
}

const (
	TraceIDKey = "trace_id"
)

// ============================================================================
// Utility Functions
// ============================================================================

func generateTraceID() string {
	bytes := make([]byte, 16)
	if _, err := rand.Read(bytes); err != nil {
		return fmt.Sprintf("trace-%d", os.Getpid())
	}
	return fmt.Sprintf("%x", bytes)
}

func getTraceID(ctx context.Context) string {
	if ctx == nil {
		return generateTraceID()
	}

	if traceID := ctx.Value(TraceIDKey); traceID != nil {
		if id, ok := traceID.(string); ok && id != "" {
			return id
		}
	}

	return generateTraceID()
}

func getCallerInfo(skip int) string {
	_, file, line, ok := runtime.Caller(skip + 2)
	if !ok {
		return "unknown:0"
	}

	parts := strings.Split(file, "/")
	if len(parts) > 2 {
		file = strings.Join(parts[len(parts)-2:], "/")
	}

	return fmt.Sprintf("%s:%d", file, line)
}

func getLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "unknown"
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return "unknown"
}

func getPID() int {
	return os.Getpid()
}

func WithTraceID(ctx context.Context, traceID string) context.Context {
	if ctx == nil {
		ctx = context.Background()
	}
	return context.WithValue(ctx, TraceIDKey, traceID)
}

// ============================================================================
// Formatters
// ============================================================================

type Formatter interface {
	Format(entry *Entry) (string, error)
}

type TextFormatter struct{}

func NewTextFormatter() *TextFormatter {
	return &TextFormatter{}
}

func (f *TextFormatter) Format(entry *Entry) (string, error) {
	var builder strings.Builder

	builder.WriteString(fmt.Sprintf("[%s] [%s] [%d] [%s] [%s] %s",
		entry.Time,
		entry.Level,
		entry.PID,
		entry.TraceID,
		entry.File,
		entry.Message,
	))

	if len(entry.Fields) > 0 {
		for key, value := range entry.Fields {
			builder.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
	}

	builder.WriteString("\n")
	return builder.String(), nil
}

type JSONFormatter struct{}

func NewJSONFormatter() *JSONFormatter {
	return &JSONFormatter{}
}

func (f *JSONFormatter) Format(entry *Entry) (string, error) {
	data, err := json.Marshal(entry)
	if err != nil {
		return "", fmt.Errorf("failed to marshal log entry: %v", err)
	}

	return string(data) + "\n", nil
}

func formatTime() string {
	return time.Now().Format("2006-01-02 15:04:05.123")
}