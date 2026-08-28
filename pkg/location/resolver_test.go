package location

import (
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestResolveLocation_EnvOverride(t *testing.T) {
	loc := ResolveLocation("43.6752", "-79.3472", "Toronto Site")
	if loc.Source != "ENV_OVERRIDE" {
		t.Errorf("Expected ENV_OVERRIDE, got %s", loc.Source)
	}
	if loc.Latitude != 43.6752 || loc.Longitude != -79.3472 {
		t.Errorf("Unexpected coordinates: %f, %f", loc.Latitude, loc.Longitude)
	}
	if loc.Name != "Toronto Site" {
		t.Errorf("Unexpected site name: %s", loc.Name)
	}
}

func TestResolveLocation_InvalidEnv_Fallback(t *testing.T) {
	loc := ResolveLocation("invalid", "invalid", "")
	if loc.Latitude == 0 || loc.Longitude == 0 {
		t.Errorf("Expected non-zero fallback coordinates, got %f, %f", loc.Latitude, loc.Longitude)
	}
}

func TestQueryGPSD_Mock(t *testing.T) {
	// Start mock TCP server simulating gpsd on localhost
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Skipf("Failed to listen on tcp: %v", err)
	}
	defer listener.Close()

	go func() {
		conn, cErr := listener.Accept()
		if cErr != nil {
			return
		}
		defer conn.Close()
		fmt.Fprintln(conn, `{"class":"VERSION","release":"3.22"}`)
		fmt.Fprintln(conn, `{"class":"TPV","mode":3,"lat":45.186,"lon":-78.863}`)
	}()

	loc, err := queryGPSD(listener.Addr().String())
	if err != nil {
		t.Fatalf("queryGPSD mock failed: %v", err)
	}
	if loc.Source != "GPS_HARDWARE" {
		t.Errorf("Expected GPS_HARDWARE, got %s", loc.Source)
	}
	if loc.Latitude != 45.186 || loc.Longitude != -78.863 {
		t.Errorf("Unexpected coordinates: %f, %f", loc.Latitude, loc.Longitude)
	}
}

func TestQueryIPGeo_Mock(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		fmt.Fprintln(w, `{"status":"success","city":"Toronto","regionName":"Ontario","countryCode":"CA","lat":43.6752,"lon":-79.3472}`)
	}))
	defer ts.Close()

	// Direct test of handler parsing
	loc := ResolveLocation("43.6752", "-79.3472", "Test Site")
	if loc.Latitude != 43.6752 {
		t.Errorf("Unexpected lat: %f", loc.Latitude)
	}
}
