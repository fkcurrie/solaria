package main

import (
	"fmt"
	"sync"
	"testing"
	"time"
)

func TestRingBuffer_PushAndCapacity(t *testing.T) {
	rb := NewRingBuffer(5)

	for i := 1; i <= 10; i++ {
		rb.Push([]SolarRecord{
			{
				Timestamp: time.Now().UTC().Format(time.RFC3339),
				Site:      fmt.Sprintf("Site %d", i),
				Telemetry: Telemetry{
					PVPowerW:       i * 10,
					ArrayCapacityW: 400,
				},
			},
		})
	}

	history := rb.GetHistory(10)
	if len(history) != 5 {
		t.Errorf("Expected history length capped at maxCap 5, got %d", len(history))
	}

	latest := rb.GetLatest()
	if latest.Site != "Site 10" {
		t.Errorf("Expected latest site 'Site 10', got '%s'", latest.Site)
	}

	if latest.Telemetry.PVPowerW != 100 {
		t.Errorf("Expected latest PV power 100W, got %dW", latest.Telemetry.PVPowerW)
	}
}

func TestRingBuffer_ConcurrentAccess(t *testing.T) {
	rb := NewRingBuffer(100)
	var wg sync.WaitGroup

	// Concurrently push 50 records from 10 goroutines
	for g := 0; g < 10; g++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			for i := 0; i < 50; i++ {
				rb.Push([]SolarRecord{
					{
						Timestamp: time.Now().UTC().Format(time.RFC3339),
						Telemetry: Telemetry{PVPowerW: id * 10},
					},
				})
				_ = rb.GetLatest()
				_ = rb.GetHistory(20)
			}
		}(g)
	}

	wg.Wait()
	history := rb.GetHistory(100)
	if len(history) != 100 {
		t.Errorf("Expected 100 records in ring buffer after concurrent pushes, got %d", len(history))
	}
}
