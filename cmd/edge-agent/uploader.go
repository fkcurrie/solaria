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
	drained, err := u.spooler.Drain(func(rec SolarRecord) error {
		payload := BatchPayload{Batch: []SolarRecord{rec}}
		data, err := json.Marshal(payload)
		if err != nil {
			return err
		}

		req, err := http.NewRequest("POST", u.endpoint, bytes.NewBuffer(data))
		if err != nil {
			return err
		}

		req.Header.Set("Content-Type", "application/json")
		if u.token != "" {
			req.Header.Set("Authorization", fmt.Sprintf("Bearer %s", u.token))
			req.Header.Set("X-API-Key", u.token)
		}

		resp, err := u.client.Do(req)
		if err != nil {
			return err
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusCreated {
			return fmt.Errorf("ingest rejected with status %d", resp.StatusCode)
		}
		return nil
	})

	if err != nil {
		log.Printf("[Uploader] Cloud offline: %v (remaining spooled safely)", err)
	} else if drained > 0 {
		log.Printf("[Uploader] Synced %d record(s) to Cloud Run", drained)
	}
}
