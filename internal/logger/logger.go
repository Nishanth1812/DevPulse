package logger

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"
)

const (
	filePermission = 0o600
	retention      = 7 * 24 * time.Hour
)

var defaultLogger *Logger

type Logger struct {
	mu   sync.Mutex
	file *os.File
}

func Init(logDir string) error {
	if err := os.MkdirAll(logDir, 0o700); err != nil {
		return fmt.Errorf("create log directory: %w", err)
	}

	if err := purgeOldLogs(logDir, time.Now()); err != nil {
		return err
	}

	logPath := filepath.Join(logDir, time.Now().Format("2006-01-02")+".log")
	file, err := os.OpenFile(logPath, os.O_CREATE|os.O_APPEND|os.O_WRONLY, filePermission)
	if err != nil {
		return fmt.Errorf("open log file: %w", err)
	}

	if defaultLogger != nil && defaultLogger.file != nil {
		_ = defaultLogger.file.Close()
	}
	defaultLogger = &Logger{file: file}

	return nil
}

func Log(level string, command string, message string) {
	if defaultLogger == nil {
		return
	}
	defaultLogger.write(level, command, message)
}

func LogError(command string, err error) {
	if err == nil {
		return
	}
	Log("ERROR", command, "error="+sanitize(err.Error()))
}

func LogAPICall(command string, keyIndex int, model string, tokens int, latency time.Duration) {
	Log("INFO", command, fmt.Sprintf(
		"api_call key_index=%d model=%s tokens=%d latency=%s",
		keyIndex,
		sanitize(model),
		tokens,
		latency.String(),
	))
}

func LogCacheEvent(command string, repo string, event string) {
	Log("INFO", command, fmt.Sprintf(
		"cache_event repo=%s event=%s",
		sanitize(repo),
		sanitize(event),
	))
}

func (l *Logger) write(level string, command string, message string) {
	l.mu.Lock()
	defer l.mu.Unlock()

	line := fmt.Sprintf(
		"%s %s %s %s\n",
		time.Now().Format(time.RFC3339),
		strings.ToUpper(sanitizeToken(level)),
		sanitizeToken(command),
		sanitize(message),
	)
	_, _ = l.file.WriteString(line)
}

func purgeOldLogs(logDir string, now time.Time) error {
	entries, err := os.ReadDir(logDir)
	if err != nil {
		return fmt.Errorf("read log directory: %w", err)
	}

	for _, entry := range entries {
		if entry.IsDir() || filepath.Ext(entry.Name()) != ".log" {
			continue
		}

		path := filepath.Join(logDir, entry.Name())
		info, err := entry.Info()
		if err != nil {
			return fmt.Errorf("stat log file %q: %w", path, err)
		}
		if now.Sub(info.ModTime()) <= retention {
			continue
		}
		if err := os.Remove(path); err != nil && !errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("remove old log file %q: %w", path, err)
		}
	}

	return nil
}

var secretPatterns []*regexp.Regexp

func init() {
	secretPatterns = []*regexp.Regexp{
		regexp.MustCompile(`-----BEGIN\s+(?:RSA|DSA|EC|OPENSSH|PGP)\s+PRIVATE\s+KEY-----`),
		regexp.MustCompile(`(?:ghp|gho|ghu|ghs|ghr)_[0-9a-zA-Z]{36,}`),
		regexp.MustCompile(`sk-[a-zA-Z0-9]{20,}`),
		regexp.MustCompile(`AKIA[0-9A-Z]{16}`),
		regexp.MustCompile(`xox[baprs]-[0-9a-zA-Z-]{10,}`),
		regexp.MustCompile(`eyJ[a-zA-Z0-9_-]{10,}\.(?:[a-zA-Z0-9_-]{10,}\.){1,2}[a-zA-Z0-9_-]{10,}`),
		regexp.MustCompile(`(?i)(api[_-]?key|apikey|secret|token|password|passwd)\s*[:=]\s*\S+`),
		regexp.MustCompile(`(?i)[a-z][a-z0-9+.\-]*://[^\s:@/]+:[^\s@/]+@`),
	}
}

func sanitize(value string) string {
	value = strings.ReplaceAll(value, "\r", " ")
	value = strings.ReplaceAll(value, "\n", " ")
	for _, re := range secretPatterns {
		value = re.ReplaceAllString(value, "[REDACTED]")
	}
	return strings.TrimSpace(value)
}

func sanitizeToken(value string) string {
	value = sanitize(value)
	if value == "" {
		return "-"
	}
	return strings.Join(strings.Fields(value), ".")
}
