package utils

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"time"
)

// LogLevel represents different logging levels
type LogLevel int

const (
	DEBUG LogLevel = iota
	INFO
	WARN
	ERROR
	FATAL
)

// String returns the string representation of LogLevel
func (l LogLevel) String() string {
	switch l {
	case DEBUG:
		return "DEBUG"
	case INFO:
		return "INFO"
	case WARN:
		return "WARN"
	case ERROR:
		return "ERROR"
	case FATAL:
		return "FATAL"
	default:
		return "UNKNOWN"
	}
}

// LogEntry represents a structured log entry
type LogEntry struct {
	Timestamp string                 `json:"timestamp"`
	Level     string                 `json:"level"`
	Message   string                 `json:"message"`
	Service   string                 `json:"service,omitempty"`
	Component string                 `json:"component,omitempty"`
	RequestID string                 `json:"request_id,omitempty"`
	UserID    string                 `json:"user_id,omitempty"`
	TraceID   string                 `json:"trace_id,omitempty"`
	SpanID    string                 `json:"span_id,omitempty"`
	Error     string                 `json:"error,omitempty"`
	Stack     string                 `json:"stack,omitempty"`
	Fields    map[string]any `json:"fields,omitempty"`
	File      string                 `json:"file,omitempty"`
	Line      int                    `json:"line,omitempty"`
}

// Logger provides structured logging capabilities
type Logger struct {
	level      LogLevel
	format     string // "json" or "text"
	output     io.Writer
	fileWriter *os.File
	mu         sync.RWMutex
	service    string
}

// NewLogger creates a new logger instance
func NewLogger() *Logger {
	return &Logger{
		level:   INFO,
		format:  "text",
		output:  os.Stdout,
		service: "mimir-aip",
	}
}

// SetLevel sets the logging level
func (l *Logger) SetLevel(level LogLevel) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.level = level
}

// SetFormat sets the logging format ("json" or "text")
func (l *Logger) SetFormat(format string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.format = strings.ToLower(format)
}

// SetOutput sets the logging output destination
func (l *Logger) SetOutput(output io.Writer) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.output = output
}

// SetFileOutput sets up file-based logging with rotation
func (l *Logger) SetFileOutput(filePath string, maxSize int64, maxBackups int, compress bool) error {
	l.mu.Lock()
	defer l.mu.Unlock()

	// Create directory if it doesn't exist
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("failed to create log directory: %w", err)
	}

	// Close existing file if open
	if l.fileWriter != nil {
		l.fileWriter.Close()
	}

	// Open new file
	file, err := os.OpenFile(filePath, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		return fmt.Errorf("failed to open log file: %w", err)
	}

	l.fileWriter = file

	// Set up multi-writer if needed
	if l.output == os.Stdout {
		l.output = io.MultiWriter(os.Stdout, file)
	} else {
		l.output = file
	}

	return nil
}

// SetService sets the service name for logging
func (l *Logger) SetService(service string) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.service = service
}

// Debug logs a debug message
func (l *Logger) Debug(msg string, fields ...Field) {
	l.log(DEBUG, msg, fields...)
}

// Info logs an info message
func (l *Logger) Info(msg string, fields ...Field) {
	l.log(INFO, msg, fields...)
}

// Warn logs a warning message
func (l *Logger) Warn(msg string, fields ...Field) {
	l.log(WARN, msg, fields...)
}

// Error logs an error message
func (l *Logger) Error(msg string, err error, fields ...Field) {
	if err != nil {
		fields = append(fields, Error(err))
	}
	l.log(ERROR, msg, fields...)
}

// Fatal logs a fatal message and exits
func (l *Logger) Fatal(msg string, err error, fields ...Field) {
	if err != nil {
		fields = append(fields, Error(err))
	}
	l.log(FATAL, msg, fields...)
	os.Exit(1)
}

// WithContext returns a logger with context fields
func (l *Logger) WithContext(ctx context.Context) *ContextLogger {
	return &ContextLogger{
		logger: l,
		ctx:    ctx,
	}
}

// WithFields returns a logger with additional fields
func (l *Logger) WithFields(fields ...Field) *FieldLogger {
	return &FieldLogger{
		logger: l,
		fields: fields,
	}
}

// log performs the actual logging
func (l *Logger) log(level LogLevel, msg string, fields ...Field) {
	l.mu.RLock()
	if level < l.level {
		l.mu.RUnlock()
		return
	}
	l.mu.RUnlock()

	entry := l.createLogEntry(level, msg, fields...)

	var output string
	if l.format == "json" {
		if jsonBytes, err := json.Marshal(entry); err == nil {
			output = string(jsonBytes)
		} else {
			output = fmt.Sprintf("Failed to marshal log entry: %v", err)
		}
	} else {
		output = l.formatTextEntry(entry)
	}

	l.mu.RLock()
	defer l.mu.RUnlock()
	fmt.Fprintln(l.output, output)
}

// createLogEntry creates a structured log entry
func (l *Logger) createLogEntry(level LogLevel, msg string, fields ...Field) *LogEntry {
	entry := &LogEntry{
		Timestamp: time.Now().UTC().Format(time.RFC3339),
		Level:     level.String(),
		Message:   msg,
		Service:   l.service,
		Fields:    make(map[string]any),
	}

	// Add caller information
	if _, file, line, ok := runtime.Caller(3); ok {
		entry.File = filepath.Base(file)
		entry.Line = line
	}

	// Process fields
	for _, field := range fields {
		field.Apply(entry)
	}

	return entry
}

// formatTextEntry formats a log entry as text
func (l *Logger) formatTextEntry(entry *LogEntry) string {
	var builder strings.Builder

	// Timestamp and level
	builder.WriteString(fmt.Sprintf("%s [%s] %s",
		entry.Timestamp,
		entry.Level,
		entry.Message))

	// Add component if present
	if entry.Component != "" {
		builder.WriteString(fmt.Sprintf(" component=%s", entry.Component))
	}

	// Add request ID if present
	if entry.RequestID != "" {
		builder.WriteString(fmt.Sprintf(" request_id=%s", entry.RequestID))
	}

	// Add error if present
	if entry.Error != "" {
		builder.WriteString(fmt.Sprintf(" error=%s", entry.Error))
	}

	// Add other fields
	for key, value := range entry.Fields {
		if str, ok := value.(string); ok {
			builder.WriteString(fmt.Sprintf(" %s=%s", key, str))
		} else {
			builder.WriteString(fmt.Sprintf(" %s=%v", key, value))
		}
	}

	// Add file and line
	if entry.File != "" && entry.Line != 0 {
		builder.WriteString(fmt.Sprintf(" (%s:%d)", entry.File, entry.Line))
	}

	return builder.String()
}

// Field represents a log field
type Field interface {
	Apply(entry *LogEntry)
}

// StringField represents a string field
type StringField struct {
	Key   string
	Value string
}

// Apply applies the field to a log entry
func (f StringField) Apply(entry *LogEntry) {
	entry.Fields[f.Key] = f.Value
}

// IntField represents an integer field
type IntField struct {
	Key   string
	Value int
}

// Apply applies the field to a log entry
func (f IntField) Apply(entry *LogEntry) {
	entry.Fields[f.Key] = f.Value
}

// FloatField represents a float field
type FloatField struct {
	Key   string
	Value float64
}

// Apply applies the field to a log entry
func (f FloatField) Apply(entry *LogEntry) {
	entry.Fields[f.Key] = f.Value
}

// BoolField represents a boolean field
type BoolField struct {
	Key   string
	Value bool
}

// Apply applies the field to a log entry
func (f BoolField) Apply(entry *LogEntry) {
	entry.Fields[f.Key] = f.Value
}

// ErrorField represents an error field
type ErrorField struct {
	Err error
}

// Apply applies the field to a log entry
func (f ErrorField) Apply(entry *LogEntry) {
	entry.Error = f.Err.Error()
	// Add stack trace for errors
	buf := make([]byte, 4096)
	if n := runtime.Stack(buf, false); n > 0 {
		entry.Stack = string(buf[:n])
	}
}

// ComponentField represents a component field
type ComponentField struct {
	Component string
}

// Apply applies the field to a log entry
func (f ComponentField) Apply(entry *LogEntry) {
	entry.Component = f.Component
}

// RequestIDField represents a request ID field
type RequestIDField struct {
	RequestID string
}

// Apply applies the field to a log entry
func (f RequestIDField) Apply(entry *LogEntry) {
	entry.RequestID = f.RequestID
}

// Field constructors

// String creates a string field
func String(key, value string) Field {
	return StringField{Key: key, Value: value}
}

// Int creates an integer field
func Int(key string, value int) Field {
	return IntField{Key: key, Value: value}
}

// Float creates a float field
func Float(key string, value float64) Field {
	return FloatField{Key: key, Value: value}
}

// Bool creates a boolean field
func Bool(key string, value bool) Field {
	return BoolField{Key: key, Value: value}
}

// Error creates an error field
func Error(err error) Field {
	return ErrorField{Err: err}
}

// Component creates a component field
func Component(component string) Field {
	return ComponentField{Component: component}
}

// RequestID creates a request ID field
func RequestID(requestID string) Field {
	return RequestIDField{RequestID: requestID}
}

// ContextLogger provides context-aware logging
type ContextLogger struct {
	logger *Logger
	ctx    context.Context
}

// Debug logs a debug message with context
func (cl *ContextLogger) Debug(msg string, fields ...Field) {
	cl.logger.Debug(msg, fields...)
}

// Info logs an info message with context
func (cl *ContextLogger) Info(msg string, fields ...Field) {
	cl.logger.Info(msg, fields...)
}

// Warn logs a warning message with context
func (cl *ContextLogger) Warn(msg string, fields ...Field) {
	cl.logger.Warn(msg, fields...)
}

// Error logs an error message with context
func (cl *ContextLogger) Error(msg string, err error, fields ...Field) {
	cl.logger.Error(msg, err, fields...)
}

// WithFields returns a logger with additional fields
func (cl *ContextLogger) WithFields(fields ...Field) *FieldLogger {
	return &FieldLogger{
		logger: cl.logger,
		fields: fields,
	}
}

// FieldLogger provides field-aware logging
type FieldLogger struct {
	logger *Logger
	fields []Field
}

// Debug logs a debug message with fields
func (fl *FieldLogger) Debug(msg string, fields ...Field) {
	allFields := append(fl.fields, fields...)
	fl.logger.Debug(msg, allFields...)
}

// Info logs an info message with fields
func (fl *FieldLogger) Info(msg string, fields ...Field) {
	allFields := append(fl.fields, fields...)
	fl.logger.Info(msg, allFields...)
}

// Warn logs a warning message with fields
func (fl *FieldLogger) Warn(msg string, fields ...Field) {
	allFields := append(fl.fields, fields...)
	fl.logger.Warn(msg, allFields...)
}

// Error logs an error message with fields
func (fl *FieldLogger) Error(msg string, err error, fields ...Field) {
	allFields := append(fl.fields, fields...)
	if err != nil {
		allFields = append(allFields, Error(err))
	}
	fl.logger.log(ERROR, msg, allFields...)
}

// Global logger instance
var globalLogger *Logger
var loggerOnce sync.Once

// GetLogger returns the global logger instance
func GetLogger() *Logger {
	loggerOnce.Do(func() {
		globalLogger = NewLogger()
	})
	return globalLogger
}

// InitLogger initializes the global logger with configuration
func InitLogger(config LoggingConfig) error {
	logger := GetLogger()

	// Set log level
	switch strings.ToLower(config.Level) {
	case "debug":
		logger.SetLevel(DEBUG)
	case "info":
		logger.SetLevel(INFO)
	case "warn":
		logger.SetLevel(WARN)
	case "error":
		logger.SetLevel(ERROR)
	case "fatal":
		logger.SetLevel(FATAL)
	default:
		logger.SetLevel(INFO)
	}

	// Set format
	logger.SetFormat(config.Format)

	// Set output
	switch strings.ToLower(config.Output) {
	case "stdout":
		logger.SetOutput(os.Stdout)
	case "file":
		if err := logger.SetFileOutput(config.FilePath, int64(config.MaxSize)*1024*1024, config.MaxBackups, config.Compress); err != nil {
			return fmt.Errorf("failed to set file output: %w", err)
		}
	case "both":
		if err := logger.SetFileOutput(config.FilePath, int64(config.MaxSize)*1024*1024, config.MaxBackups, config.Compress); err != nil {
			return fmt.Errorf("failed to set file output: %w", err)
		}
		// Multi-writer is handled in SetFileOutput
	default:
		logger.SetOutput(os.Stdout)
	}

	return nil
}
