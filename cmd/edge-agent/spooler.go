package main

import (
	"bufio"
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

func (s *Spooler) Append(record SolarRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	f, err := os.OpenFile(s.filePath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0600)
	if err != nil {
		return err
	}
	defer f.Close()

	data, err := json.Marshal(record)
	if err != nil {
		return err
	}

	_, err = f.Write(append(data, '\n'))
	return err
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
	defer f.Close()

	var records []SolarRecord
	scanner := bufio.NewScanner(f)
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

func (s *Spooler) Clear() error {
	s.mu.Lock()
	defer s.mu.Unlock()
	return os.Remove(s.filePath)
}
