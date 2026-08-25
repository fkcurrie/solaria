package main

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"sync"
	"testing"
)

func TestSpooler_AppendAndCount(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spooler-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spoolPath := filepath.Join(tmpDir, "test_spool.jsonl")
	spooler := NewSpooler(spoolPath)

	if count := spooler.Count(); count != 0 {
		t.Errorf("Expected initial count 0, got %d", count)
	}

	for i := 0; i < 5; i++ {
		rec := SolarRecord{
			Timestamp: "2026-08-25T12:00:00Z",
			Site:      "Dorset Lakehouse",
			Telemetry: &Telemetry{PVPowerW: 300 + i},
		}
		if err := spooler.Append(rec); err != nil {
			t.Fatalf("Append failed: %v", err)
		}
	}

	if count := spooler.Count(); count != 5 {
		t.Errorf("Expected count 5, got %d", count)
	}
}

func TestSpooler_Drain_PartialFailure_ZeroDataLoss(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spooler-drain-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spoolPath := filepath.Join(tmpDir, "test_drain.jsonl")
	spooler := NewSpooler(spoolPath)

	// Append 10 records
	for i := 0; i < 10; i++ {
		rec := SolarRecord{
			Timestamp: "2026-08-25T12:00:00Z",
			Site:      "Dorset Lakehouse",
			Telemetry: &Telemetry{PVPowerW: i * 10},
		}
		_ = spooler.Append(rec)
	}

	// Drain: succeed on first 4, fail on 5th
	uploadAttempts := 0
	drained, err := spooler.Drain(func(r SolarRecord) error {
		uploadAttempts++
		if uploadAttempts > 4 {
			return fmt.Errorf("network uplink dropped")
		}
		return nil
	})

	if err != nil {
		t.Fatalf("Unexpected drain error: %v", err)
	}
	if drained != 4 {
		t.Errorf("Expected 4 drained records, got %d", drained)
	}

	// Remaining count must be 6
	remainingCount := spooler.Count()
	if remainingCount != 6 {
		t.Errorf("Expected 6 remaining records in spool after partial upload, got %d", remainingCount)
	}

	// Drain remainder with 100% success
	drainedRemainder, err := spooler.Drain(func(r SolarRecord) error {
		return nil
	})
	if err != nil {
		t.Fatalf("Drain remainder failed: %v", err)
	}
	if drainedRemainder != 6 {
		t.Errorf("Expected 6 remainder records drained, got %d", drainedRemainder)
	}

	if finalCount := spooler.Count(); finalCount != 0 {
		t.Errorf("Expected 0 records after full drain, got %d", finalCount)
	}
}

func TestSpooler_CorruptionToleranceAndQuarantine(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spooler-corrupt-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spoolPath := filepath.Join(tmpDir, "test_corrupt.jsonl")
	spooler := NewSpooler(spoolPath)

	// Write valid record 1
	_ = spooler.Append(SolarRecord{Site: "Dorset Lakehouse", Telemetry: &Telemetry{PVPowerW: 100}})

	// Inject corrupt malformed line
	f, _ := os.OpenFile(spoolPath, os.O_APPEND|os.O_WRONLY, 0600)
	_, _ = f.Write([]byte("{CORRUPTED_JSON_LINE_FROM_POWER_OUTAGE\n"))
	f.Close()

	// Write valid record 2
	_ = spooler.Append(SolarRecord{Site: "Dorset Lakehouse", Telemetry: &Telemetry{PVPowerW: 200}})

	var drainedRecords []SolarRecord
	drained, err := spooler.Drain(func(r SolarRecord) error {
		drainedRecords = append(drainedRecords, r)
		return nil
	})

	if err != nil {
		t.Fatalf("Drain failed: %v", err)
	}
	if drained != 2 {
		t.Errorf("Expected 2 valid records to be recovered, got %d", drained)
	}

	// Check that quarantine log was created
	corruptLogPath := spoolPath + ".corrupt.log"
	if _, err := os.Stat(corruptLogPath); os.IsNotExist(err) {
		t.Errorf("Expected corrupt log to exist at %s", corruptLogPath)
	}
}

func TestSpooler_ConcurrentAppends(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "spooler-concurrent-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spoolPath := filepath.Join(tmpDir, "test_concurrent.jsonl")
	spooler := NewSpooler(spoolPath)

	var wg sync.WaitGroup
	workers := 10
	perWorker := 20

	for w := 0; w < workers; w++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()
			for i := 0; i < perWorker; i++ {
				_ = spooler.Append(SolarRecord{
					Site:      "Dorset",
					Telemetry: &Telemetry{PVPowerW: workerID*100 + i},
				})
			}
		}(w)
	}
	wg.Wait()

	expectedTotal := workers * perWorker
	if count := spooler.Count(); count != expectedTotal {
		t.Errorf("Expected %d records from concurrent appends, got %d", expectedTotal, count)
	}
}

func TestUploader_FlushPending_WithServer(t *testing.T) {
	receivedCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var payload BatchPayload
		if err := json.NewDecoder(r.Body).Decode(&payload); err == nil {
			receivedCount += len(payload.Batch)
			w.WriteHeader(http.StatusOK)
		} else {
			w.WriteHeader(http.StatusBadRequest)
		}
	}))
	defer server.Close()

	tmpDir, err := os.MkdirTemp("", "uploader-test-*")
	if err != nil {
		t.Fatalf("Failed to create temp dir: %v", err)
	}
	defer os.RemoveAll(tmpDir)

	spooler := NewSpooler(filepath.Join(tmpDir, "upload_spool.jsonl"))
	for i := 0; i < 5; i++ {
		_ = spooler.Append(SolarRecord{Site: "Dorset", Telemetry: &Telemetry{PVPowerW: 250}})
	}

	uploader := NewUploader(server.URL, "test-token", spooler)
	uploader.FlushPending()

	if receivedCount != 5 {
		t.Errorf("Expected server to receive 5 records, got %d", receivedCount)
	}
	if remaining := spooler.Count(); remaining != 0 {
		t.Errorf("Expected spool to be empty, got %d", remaining)
	}
}
