package main

import (
	"bufio"
	"bytes"
	"encoding/json"
	"os"
	"sync"
)

// SolarRecord encapsulates a single snapshot of telemetry + weather + sun classification
type SolarRecord struct {
	Timestamp         string             `json:"timestamp"`
	Site              string             `json:"site"`
	Location          map[string]float64 `json:"location"`
	Telemetry         *Telemetry         `json:"telemetry"`
	Weather           WeatherMetrics     `json:"weather"`
	SunClassification SunCondition       `json:"sun_classification"`
}

// Spooler manages persistent append-only disk spooling for zero data loss
type Spooler struct {
	filePath string
	mu       sync.Mutex
}

func NewSpooler(filePath string) *Spooler {
	return &Spooler{filePath: filePath}
}

// MaxSpoolBytes limits spool file size to 25MB to protect flash storage (SD cards)
const MaxSpoolBytes int64 = 25 * 1024 * 1024

func (s *Spooler) Append(record SolarRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Check file size before appending and rotate safely on line boundaries if exceeding limit
	if fi, err := os.Stat(s.filePath); err == nil && fi.Size() > MaxSpoolBytes {
		if data, rErr := os.ReadFile(s.filePath); rErr == nil {
			// Find newline near the midpoint to preserve valid JSON lines
			mid := len(data) / 2
			if idx := bytes.IndexByte(data[mid:], '\n'); idx != -1 {
				tmpPath := s.filePath + ".rotate.tmp"
				if err := os.WriteFile(tmpPath, data[mid+idx+1:], 0600); err == nil {
					_ = os.Rename(tmpPath, s.filePath)
				}
			}
		}
	}

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer func() { _ = f.Close() }()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	if _, err := f.Write(append(data, '\n')); err != nil {
		return err
	}
	return f.Sync()
}

func (s *Spooler) ReadBatch(limit int) ([]SolarRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.Open(s.filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	defer func() { _ = f.Close() }()

	var records []SolarRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() && len(records) < limit {
		line := scanner.Bytes()
		if len(line) == 0 {
			continue
		}
		var r SolarRecord
		if err := json.Unmarshal(line, &r); err == nil {
			records = append(records, r)
		}
	}
	return records, scanner.Err()
}

func (s *Spooler) Count() int {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return 0
	}

	f, err := os.Open(s.filePath)
	if err != nil {
		return 0
	}
	defer func() { _ = f.Close() }()

	count := 0
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) > 0 {
			count++
		}
	}
	return count
}

func (s *Spooler) Drain(uploader func(record SolarRecord) error) (int, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, err := os.Stat(s.filePath); os.IsNotExist(err) {
		return 0, nil
	}

	f, err := os.Open(s.filePath)
	if err != nil {
		return 0, err
	}

	var validRecords []SolarRecord
	scanner := bufio.NewScanner(f)
	scanner.Buffer(make([]byte, 64*1024), 1024*1024)
	for scanner.Scan() {
		line := bytes.TrimSpace(scanner.Bytes())
		if len(line) == 0 {
			continue
		}
		var r SolarRecord
		if err := json.Unmarshal(line, &r); err == nil {
			validRecords = append(validRecords, r)
		} else {
			// Quarantine corrupted line to dead-letter log
			qf, qErr := os.OpenFile(s.filePath+".corrupt.log", os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
			if qErr == nil {
				_, _ = qf.Write(append(line, '\n'))
				_ = qf.Close()
			}
		}
	}
	f.Close()

	if len(validRecords) == 0 {
		_ = os.Remove(s.filePath)
		return 0, nil
	}

	var remaining []SolarRecord
	drainedCount := 0

	for i, rec := range validRecords {
		if err := uploader(rec); err != nil {
			remaining = validRecords[i:]
			break
		}
		drainedCount++
	}

	if len(remaining) == 0 {
		_ = os.Remove(s.filePath)
	} else {
		tmpPath := s.filePath + ".tmp"
		tf, err := os.OpenFile(tmpPath, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0600)
		if err == nil {
			for _, r := range remaining {
				d, _ := json.Marshal(r)
				_, _ = tf.Write(append(d, '\n'))
			}
			_ = tf.Sync()
			tf.Close()
			_ = os.Rename(tmpPath, s.filePath)
		}
	}

	return drainedCount, nil
}

func (s *Spooler) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.filePath)
}
