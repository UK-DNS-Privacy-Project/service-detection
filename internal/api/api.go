package api

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"

	"dnsprivacy.org.uk/resolver/internal/config"
)

func Start() {
	http.HandleFunc("/json", jsonHandler)
	http.HandleFunc("/", rootHandler)
	log.Println("Starting HTTP server on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %s", err)
	}
}

func rootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "up")
}

func jsonHandler(w http.ResponseWriter, r *http.Request) {
	host := r.Host + "."
	if host == "." {
		http.Error(w, "missing host name", http.StatusBadRequest)
		return
	}

	// Debug log all headers
	log.Println("HTTP Headers:")
	for name, values := range r.Header {
		for _, value := range values {
			log.Printf("  %s: %s", name, value)
		}
	}

	config.Mutex.Lock()
	rec, ok := config.DNSData[host]
	config.Mutex.Unlock()

	log.Println("HTTP Request Received", host, rec)

	if !ok {
		http.Error(w, "not found", http.StatusNotFound)
		return
	}

	requesterIP := r.Header.Get("X-Forwarded-For")
	if requesterIP == "" {
		requesterIP, _, _ = net.SplitHostPort(r.RemoteAddr)
	}

	geoIPCity, err := lookupGeoIPCity(requesterIP)
	if err != nil {
		log.Printf("GeoIP lookup failed for %s: %v\n", requesterIP, err)
	}

	geoIPCountry, err := lookupGeoIPCountry(requesterIP)
	if err != nil {
		log.Printf("GeoIP lookup failed for %s: %v\n", requesterIP, err)
	}

	geoIPASN, err := lookupGeoIPASN(requesterIP)
	if err != nil {
		log.Printf("ASN lookup failed for %s: %v\n", requesterIP, err)
	}

	servers := make(map[string]string)
	for _, ip := range rec.IPs {
		reverseDNS, err := lookupReverseDNS(ip)
		if err != nil {
			log.Printf("Reverse DNS lookup failed for %s: %v\n", ip, err)
		}
		servers[ip] = reverseDNS
	}

	response := map[string]interface{}{
		"domain":      host,
		"ips":         rec.IPs,
		"known":       true,
		"requesterIP": requesterIP,
		"city":        geoIPCity,
		"country":     geoIPCountry,
		"isp":         geoIPASN,
		"servers":     servers,
	}
	for _, ip := range rec.IPs {
		if !config.KnownIPs[ip] {
			response["known"] = false
			break
		}
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	json.NewEncoder(w).Encode(response)
}

func lookupReverseDNS(ip string) (string, error) {
	addrs, err := net.LookupAddr(ip)
	if err != nil {
		return "", err
	}
	if len(addrs) > 0 {
		return addrs[0], nil
	}
	return "", nil
}
