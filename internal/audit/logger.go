package audit

import (
	"bufio"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const (
	ActionAcceptedFinancialRiskDisclaimer = "accepted_financial_risk_disclaimer"
)

// Logger appends immutable audit records.
type Logger struct {
	path string
	file *os.File
}

// Open opens the append-only audit log.
func Open(path string) (*Logger, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create audit dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open audit log: %w", err)
	}
	return &Logger{path: path, file: file}, nil
}

// Close closes the audit log.
func (l *Logger) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Write appends an audit record.
func (l *Logger) Write(record Record) error {
	body, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("marshal audit record: %w", err)
	}
	if _, err := l.file.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("append audit record: %w", err)
	}
	return l.file.Sync()
}

// HashState computes a stable state hash.
func HashState(payload []byte) string {
	digest := sha256.Sum256(payload)
	return hex.EncodeToString(digest[:])
}

// HasAction reports whether a matching audit action exists.
func (l *Logger) HasAction(action string) (bool, error) {
	file, err := os.Open(l.path)
	if err != nil {
		return false, fmt.Errorf("open audit log for read: %w", err)
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		var record Record
		if err := json.Unmarshal(scanner.Bytes(), &record); err != nil {
			continue
		}
		if record.Action == action {
			return true, nil
		}
	}
	if err := scanner.Err(); err != nil {
		return false, fmt.Errorf("scan audit log: %w", err)
	}
	return false, nil
}

// DisclaimerRecord builds the disclaimer acceptance audit record.
func DisclaimerRecord(operatorSource string, text string, agentVersion string) Record {
	payload := []byte(text + "|" + agentVersion)
	hash := HashState(payload)
	return Record{
		TimestampUTC:    time.Now().UTC(),
		OperatorSource:  operatorSource,
		Action:          ActionAcceptedFinancialRiskDisclaimer,
		CorrelationID:   "financial-risk-disclaimer",
		BeforeStateHash: hash,
		AfterStateHash:  hash,
	}
}
