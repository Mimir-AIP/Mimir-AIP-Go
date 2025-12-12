package utils

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

// ExecutionLogLevel represents the log level
type ExecutionLogLevel string

const (
	LogLevelInfo  ExecutionLogLevel = "INFO"
	LogLevelWarn  ExecutionLogLevel = "WARN"
	LogLevelError ExecutionLogLevel = "ERROR"
	LogLevelDebug ExecutionLogLevel = "DEBUG"
)

// ExecutionLogEntry represents a single log entry
type ExecutionLogEntry struct {
	Timestamp time.Time         `json:"timestamp"`
	Level     ExecutionLogLevel `json:"level"`
	Message   string            `json:"message"`
	Data      map[string]any    `json:"data,omitempty"`
	StepName  string            `json:"step_name,omitempty"`
	Plugin    string            `json:"plugin,omitempty"`
}

// ExecutionLog represents the complete log for an execution
type ExecutionLog struct {
	ExecutionID string              `json:"execution_id"`
	JobID       string              `json:"job_id,omitempty"`
	PipelineID  string              `json:"pipeline_id,omitempty"`
	StartTime   time.Time           `json:"start_time"`
	EndTime     *time.Time          `json:"end_time,omitempty"`
	Status      string              `json:"status"` // running, completed, failed
	Entries     []ExecutionLogEntry `json:"entries"`
	mutex       sync.RWMutex        `json:"-"`
}

// ExecutionLogger manages logging for pipeline and job executions
type ExecutionLogger struct {
	logsDir       string
	activeLogs    map[string]*ExecutionLog
	mutex         sync.RWMutex
	maxLogSize    int // Maximum number of entries per log
	retentionDays int // Days to retain logs
}

// NewExecutionLogger creates a new execution logger
func NewExecutionLogger(logsDir string) *ExecutionLogger {
	return &ExecutionLogger{
		logsDir:       logsDir,
		activeLogs:    make(map[string]*ExecutionLog),
		maxLogSize:    10000,
		retentionDays: 30,
	}
}

// Initialize creates the logs directory if it doesn't exist
func (el *ExecutionLogger) Initialize() error {
	if err := os.MkdirAll(el.logsDir, 0755); err != nil {
		return fmt.Errorf("failed to create logs directory: %w", err)
	}
	return nil
}

// StartExecution starts logging for a new execution
func (el *ExecutionLogger) StartExecution(executionID, jobID, pipelineID string) *ExecutionLog {
	el.mutex.Lock()
	defer el.mutex.Unlock()

	log := &ExecutionLog{
		ExecutionID: executionID,
		JobID:       jobID,
		PipelineID:  pipelineID,
		StartTime:   time.Now(),
		Status:      "running",
		Entries:     make([]ExecutionLogEntry, 0),
	}

	el.activeLogs[executionID] = log
	return log
}

// EndExecution marks an execution as completed
func (el *ExecutionLogger) EndExecution(executionID string, status string) error {
	el.mutex.RLock()
	log, exists := el.activeLogs[executionID]
	el.mutex.RUnlock()

	if !exists {
		return fmt.Errorf("execution %s not found", executionID)
	}

	log.mutex.Lock()
	endTime := time.Now()
	log.EndTime = &endTime
	log.Status = status
	log.mutex.Unlock()

	// Save log to file
	if err := el.saveLogToFile(log); err != nil {
		GetLogger().Error("Failed to save execution log", err, String("execution_id", executionID))
	}

	// Remove from active logs after a delay (to allow retrieval of recent logs)
	go func() {
		time.Sleep(5 * time.Minute)
		el.mutex.Lock()
		delete(el.activeLogs, executionID)
		el.mutex.Unlock()
	}()

	return nil
}

// Log adds a log entry to an execution
func (el *ExecutionLogger) Log(executionID string, level ExecutionLogLevel, message string, data map[string]any, stepName, plugin string) {
	el.mutex.RLock()
	log, exists := el.activeLogs[executionID]
	el.mutex.RUnlock()

	if !exists {
		GetLogger().Warn("Attempted to log to non-existent execution", String("execution_id", executionID))
		return
	}

	entry := ExecutionLogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   message,
		Data:      data,
		StepName:  stepName,
		Plugin:    plugin,
	}

	log.mutex.Lock()
	log.Entries = append(log.Entries, entry)

	// Truncate if exceeding max size
	if len(log.Entries) > el.maxLogSize {
		log.Entries = log.Entries[len(log.Entries)-el.maxLogSize:]
	}
	log.mutex.Unlock()
}

// GetExecutionLog retrieves the log for a specific execution
func (el *ExecutionLogger) GetExecutionLog(executionID string) (*ExecutionLog, error) {
	// Check active logs first
	el.mutex.RLock()
	log, exists := el.activeLogs[executionID]
	el.mutex.RUnlock()

	if exists {
		// Return a copy to prevent external modification
		log.mutex.RLock()
		defer log.mutex.RUnlock()

		logCopy := &ExecutionLog{
			ExecutionID: log.ExecutionID,
			JobID:       log.JobID,
			PipelineID:  log.PipelineID,
			StartTime:   log.StartTime,
			EndTime:     log.EndTime,
			Status:      log.Status,
			Entries:     make([]ExecutionLogEntry, len(log.Entries)),
		}
		copy(logCopy.Entries, log.Entries)
		return logCopy, nil
	}

	// Load from file if not in active logs
	return el.loadLogFromFile(executionID)
}

// ListLogs lists all available execution logs with optional filtering
func (el *ExecutionLogger) ListLogs(jobID, pipelineID string, limit int) ([]ExecutionLog, error) {
	logs := make([]ExecutionLog, 0)

	// Add active logs
	el.mutex.RLock()
	for _, log := range el.activeLogs {
		if (jobID == "" || log.JobID == jobID) && (pipelineID == "" || log.PipelineID == pipelineID) {
			log.mutex.RLock()
			logCopy := ExecutionLog{
				ExecutionID: log.ExecutionID,
				JobID:       log.JobID,
				PipelineID:  log.PipelineID,
				StartTime:   log.StartTime,
				EndTime:     log.EndTime,
				Status:      log.Status,
				// Don't copy entries for list view (too large)
			}
			log.mutex.RUnlock()
			logs = append(logs, logCopy)
		}
	}
	el.mutex.RUnlock()

	// Load from files
	files, err := filepath.Glob(filepath.Join(el.logsDir, "*.json"))
	if err != nil {
		return logs, fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		if limit > 0 && len(logs) >= limit {
			break
		}

		log, err := el.loadLogFromFileByPath(file)
		if err != nil {
			continue
		}

		if (jobID == "" || log.JobID == jobID) && (pipelineID == "" || log.PipelineID == pipelineID) {
			// Don't include entries in list view
			log.Entries = nil
			logs = append(logs, *log)
		}
	}

	return logs, nil
}

// CleanupOldLogs removes logs older than the retention period
func (el *ExecutionLogger) CleanupOldLogs() error {
	cutoff := time.Now().AddDate(0, 0, -el.retentionDays)

	files, err := filepath.Glob(filepath.Join(el.logsDir, "*.json"))
	if err != nil {
		return fmt.Errorf("failed to list log files: %w", err)
	}

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := os.Remove(file); err != nil {
				GetLogger().Warn("Failed to remove old log file", Error(err), String("file", file))
			}
		}
	}

	return nil
}

// Helper methods

func (el *ExecutionLogger) saveLogToFile(log *ExecutionLog) error {
	log.mutex.RLock()
	defer log.mutex.RUnlock()

	data, err := json.MarshalIndent(log, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal log: %w", err)
	}

	fileName := fmt.Sprintf("%s.json", log.ExecutionID)
	filePath := filepath.Join(el.logsDir, fileName)

	if err := os.WriteFile(filePath, data, 0644); err != nil {
		return fmt.Errorf("failed to write log file: %w", err)
	}

	return nil
}

func (el *ExecutionLogger) loadLogFromFile(executionID string) (*ExecutionLog, error) {
	fileName := fmt.Sprintf("%s.json", executionID)
	filePath := filepath.Join(el.logsDir, fileName)
	return el.loadLogFromFileByPath(filePath)
}

func (el *ExecutionLogger) loadLogFromFileByPath(filePath string) (*ExecutionLog, error) {
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read log file: %w", err)
	}

	var log ExecutionLog
	if err := json.Unmarshal(data, &log); err != nil {
		return nil, fmt.Errorf("failed to unmarshal log: %w", err)
	}

	return &log, nil
}

// Global execution logger
var globalExecutionLogger *ExecutionLogger
var executionLoggerOnce sync.Once

// GetExecutionLogger returns the global execution logger instance
func GetExecutionLogger() *ExecutionLogger {
	executionLoggerOnce.Do(func() {
		globalExecutionLogger = NewExecutionLogger("./logs/executions")
		if err := globalExecutionLogger.Initialize(); err != nil {
			GetLogger().Error("Failed to initialize execution logger", err)
		}

		// Start cleanup goroutine
		go func() {
			ticker := time.NewTicker(24 * time.Hour)
			defer ticker.Stop()

			for range ticker.C {
				if err := globalExecutionLogger.CleanupOldLogs(); err != nil {
					GetLogger().Error("Failed to cleanup old logs", err)
				}
			}
		}()
	})
	return globalExecutionLogger
}
