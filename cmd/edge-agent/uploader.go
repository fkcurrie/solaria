package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"time"
)

// Uploader handles shipping batches of solar telemetry records to Cloud Run
type Uploader struct {
	endpoint string
	token    string
	spooler  *Spooler
	client   *http.Client
}

type BatchPayload struct {
	Batch []SolarRecord `json:"batch"`
}

func NewUploader(endpoint, token string, spooler *Spooler) *Uploader {
	return &Uploader{
		endpoint: endpoint,
		token:    token,
		spooler:  spooler,
		client:   &http.Client{Timeout: 10 * time.Second},
	}
}

func (u *Uploader) FlushPending() {
	records, err := u.spooler.ReadBatch(30)
	if err != nil || len(records) == 0 {
		return
	}

	payload := BatchPayload{Batch: records}
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	req, err := http.NewRequest("POST", u.endpoint, bytes.NewBuffer(data))
	if err != nil {
		return
	}

	req.Header.Set("Content-Type", "application/json")
	if u.token != "" {
		req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", u.token))
	}

	resp, err := u.client.Do(req)
	if err != nil {
		log.Printf("[Uploader] Cloud offline: %v (data queued safely in spool)", err)
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusOK || resp.StatusCode == http.StatusCreated {
		log.Printf("[Uploader] Synced %d record(s) to Cloud Run", len(records))
		_ = u.spooler.Clear()
	} else {
		log.Printf("[Uploader] Ingest rejected status %d", resp.StatusCode)
	}
}
