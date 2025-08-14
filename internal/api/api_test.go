package api

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"dnsprivacy.org.uk/resolver/internal/config"
	"dnsprivacy.org.uk/resolver/internal/models"
)

func TestMain(m *testing.M) {
	geoIPCityLookup = func(ip string) (string, error) { return "TestCity", nil }
	geoIPCountryLookup = func(ip string) (string, error) { return "TestCountry", nil }
	geoIPASNLookup = func(ip string) (string, error) { return "TestISP", nil }
	reverseDNSLookup = func(ip string) (string, error) { return "test.reverse.local.", nil }
	m.Run()
}

func TestRootHandler(t *testing.T) {
	req := httptest.NewRequest("GET", "/", nil)
	w := httptest.NewRecorder()
	rootHandler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Errorf("expected 200 OK, got %d", resp.StatusCode)
	}
}

func TestJsonHandler_NotFound(t *testing.T) {
	req := httptest.NewRequest("GET", "/json", nil)
	req.Host = "missing"
	w := httptest.NewRecorder()
	config.DNSData = make(map[string]models.Record)
	jsonHandler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusNotFound {
		t.Errorf("expected 404 Not Found, got %d", resp.StatusCode)
	}
}

func TestJsonHandler_Found(t *testing.T) {
	host := "test.example.com."
	config.DNSData = map[string]models.Record{
		host: {IPs: []string{"1.2.3.4"}, Timestamp: config.Now()},
	}
	config.KnownIPs = map[string]bool{"1.2.3.4": true}
	req := httptest.NewRequest("GET", "/json", nil)
	req.Host = "test.example.com"
	req.RemoteAddr = "1.2.3.4:12345"
	w := httptest.NewRecorder()
	jsonHandler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if data["domain"] != "test.example.com." {
		t.Errorf("expected domain to be test.example.com., got %v", data["domain"])
	}
	if data["city"] != "TestCity" {
		t.Errorf("expected city to be TestCity, got %v", data["city"])
	}
	if data["country"] != "TestCountry" {
		t.Errorf("expected country to be TestCountry, got %v", data["country"])
	}
	if data["isp"] != "TestISP" {
		t.Errorf("expected isp to be TestISP, got %v", data["isp"])
	}
	if data["known"] != true {
		t.Errorf("expected known to be true, got %v", data["known"])
	}
}

func TestJsonHandler_UnknownIP(t *testing.T) {
	host := "test2.example.com."
	config.DNSData = map[string]models.Record{
		host: {IPs: []string{"5.6.7.8"}, Timestamp: config.Now()},
	}
	config.KnownIPs = map[string]bool{"1.2.3.4": true}
	req := httptest.NewRequest("GET", "/json", nil)
	req.Host = "test2.example.com"
	req.RemoteAddr = "5.6.7.8:12345"
	w := httptest.NewRecorder()
	jsonHandler(w, req)
	resp := w.Result()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("expected 200 OK, got %d", resp.StatusCode)
	}
	var data map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&data); err != nil {
		t.Fatalf("failed to decode JSON: %v", err)
	}
	if data["known"] != false {
		t.Errorf("expected known to be false, got %v", data["known"])
	}
}
