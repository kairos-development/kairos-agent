package journal

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"time"
)

// Entry is an append-only journal entry.
type Entry struct {
	ClientOrderID string    `json:"client_order_id"`
	Kind          string    `json:"kind"`
	Payload       string    `json:"payload"`
	TimestampUTC  time.Time `json:"timestamp_utc"`
}

// OrderIntent is the journal-layer representation of a risk-approved order intent.
type OrderIntent struct {
	ClientOrderID string    `json:"client_order_id"`
	Symbol        string    `json:"symbol"`
	Side          string    `json:"side"`
	OrderType     string    `json:"order_type"`
	Quantity      string    `json:"quantity"`
	Price         string    `json:"price"`
	TimestampUTC  time.Time `json:"timestamp_utc"`
}

// Log manages the append-only journal.
type Log struct {
	path         string
	maxSizeBytes int64
	file         *os.File
}

// Open opens the journal file for append.
func Open(path string, maxSizeBytes int64) (*Log, error) {
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return nil, fmt.Errorf("create journal dir: %w", err)
	}
	file, err := os.OpenFile(path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return nil, fmt.Errorf("open journal: %w", err)
	}
	return &Log{path: path, maxSizeBytes: maxSizeBytes, file: file}, nil
}

// Close closes the journal file.
func (l *Log) Close() error {
	if l == nil || l.file == nil {
		return nil
	}
	return l.file.Close()
}

// Append appends and fsyncs a journal entry.
func (l *Log) Append(entry Entry) error {
	body, err := json.Marshal(entry)
	if err != nil {
		return fmt.Errorf("marshal journal entry: %w", err)
	}
	if _, err := l.file.Write(append(body, '\n')); err != nil {
		return fmt.Errorf("append journal entry: %w", err)
	}
	if err := l.file.Sync(); err != nil {
		return fmt.Errorf("sync journal entry: %w", err)
	}
	return l.rotateIfNeeded()
}

func (l *Log) rotateIfNeeded() error {
	info, err := l.file.Stat()
	if err != nil {
		return fmt.Errorf("stat journal: %w", err)
	}
	if info.Size() < l.maxSizeBytes {
		return nil
	}
	rotated := fmt.Sprintf("%s.%d", l.path, time.Now().UTC().Unix())
	if err := l.file.Close(); err != nil {
		return err
	}
	if err := os.Rename(l.path, rotated); err != nil {
		return fmt.Errorf("rotate journal: %w", err)
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return fmt.Errorf("reopen journal: %w", err)
	}
	l.file = file
	return nil
}

// Replay replays and deduplicates journal entries by client_order_id.
func (l *Log) Replay() ([]Entry, error) {
	files, err := filepath.Glob(l.path + "*")
	if err != nil {
		return nil, fmt.Errorf("glob journal files: %w", err)
	}
	sort.Strings(files)
	indexes := make(map[string]int)
	var entries []Entry
	for _, path := range files {
		file, err := os.Open(path)
		if err != nil {
			return nil, fmt.Errorf("open journal file: %w", err)
		}
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			var entry Entry
			if err := json.Unmarshal(scanner.Bytes(), &entry); err != nil {
				continue
			}
			if entry.ClientOrderID != "" {
				if idx, ok := indexes[entry.ClientOrderID]; ok {
					entries[idx] = entry
					continue
				}
				indexes[entry.ClientOrderID] = len(entries)
			}
			entries = append(entries, entry)
		}
		file.Close()
		if err := scanner.Err(); err != nil {
			return nil, fmt.Errorf("scan journal file: %w", err)
		}
	}
	return entries, nil
}

// TruncateAfterReconciliation removes rotated segments and truncates the active journal.
func (l *Log) TruncateAfterReconciliation() error {
	if err := l.file.Close(); err != nil {
		return err
	}
	matches, err := filepath.Glob(l.path + "*")
	if err != nil {
		return err
	}
	for _, match := range matches {
		if err := os.Remove(match); err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	file, err := os.OpenFile(l.path, os.O_CREATE|os.O_TRUNC|os.O_APPEND|os.O_WRONLY, 0o600)
	if err != nil {
		return err
	}
	l.file = file
	return nil
}

// AppendOrderIntent records a risk-approved intent entry.
func AppendOrderIntent(log *Log, intent OrderIntent) error {
	body, err := json.Marshal(intent)
	if err != nil {
		return err
	}
	return log.Append(Entry{ClientOrderID: intent.ClientOrderID, Kind: "order_intent", Payload: string(body), TimestampUTC: time.Now().UTC()})
}
