package main

import (
	"encoding/json"
	"fmt"
	"log"
	"net"
	"net/http"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/miekg/dns"
	"github.com/oschwald/geoip2-golang"
)

var (
	dnsData  = make(map[string]record)
	mu       sync.Mutex
	ttl      = 5 * time.Minute // Time-to-live for each DNS record
	knownIPs = map[string]bool{
		"209.250.227.42":                         true,
		"64.176.190.82":                          true,
		"2001:19f0:7400:13c7:5400:5ff:fe40:d1ad": true,
		"2a05:f480:3400:24fd:5400:5ff:fe40:e60b": true,
	}
	dnsServers = []string{
		os.Getenv("ACME_CHALLENGE_DNS_1"),
		os.Getenv("ACME_CHALLENGE_DNS_2"),
	}
	currentDNS int32
)

type record struct {
	ips       []string
	timestamp time.Time
}

func lookupGeoIPCity(ip string) (string, error) {
	db, err := geoip2.Open("/usr/local/share/GeoIP/GeoLite2-City.mmdb")
	if err != nil {
		return "", err
	}
	defer db.Close()

	parsedIP := net.ParseIP(ip)
	record, err := db.City(parsedIP)
	if err != nil {
		return "", err
	}

	return record.City.Names["en"], nil
}

func lookupGeoIPCountry(ip string) (string, error) {
	db, err := geoip2.Open("/usr/local/share/GeoIP/GeoLite2-Country.mmdb")
	if err != nil {
		return "", err
	}
	defer db.Close()

	parsedIP := net.ParseIP(ip)
	record, err := db.Country(parsedIP)
	if err != nil {
		return "", err
	}

	return record.Country.Names["en"], nil
}

func lookupGeoIPASN(ip string) (string, error) {
	db, err := geoip2.Open("/usr/local/share/GeoIP/GeoLite2-ASN.mmdb")
	if err != nil {
		return "", err
	}
	defer db.Close()

	parsedIP := net.ParseIP(ip)
	record, err := db.ASN(parsedIP)
	if err != nil {
		return "", err
	}

	return record.AutonomousSystemOrganization, nil
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

// DNSHandler handles DNS requests
func DNSHandler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		log.Println("Query Received", q.Qtype, q.Name)

		if strings.ToLower(q.Name) == os.Getenv("ACME_CHALLENGE_DOMAIN") {
			// Query upstream DNS servers for TXT record
			txtRecords, err := queryRoundRobinDNS(q.Name)
			if err != nil {
				log.Printf("Failed to query upstream servers for %s: %v\n", q.Name, err)
				// Respond with a SERVFAIL if upstream query fails
				m.Rcode = dns.RcodeServerFailure
				w.WriteMsg(m)
				return
			}

			// Add the TXT records to the DNS response
			for _, txt := range txtRecords {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 TXT \"%s\"", q.Name, txt))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
			w.WriteMsg(m)
			return
		}

		switch q.Qtype {
		case dns.TypeSOA:
			if HasSuffixCaseInsensitive(q.Name, os.Getenv("DOMAIN")) {
				rr, err := dns.NewRR(fmt.Sprintf("%s SOA %s %s 2025012400 3600 600 604800 86400", q.Name, os.Getenv("DOMAIN"), os.Getenv("SOA_ADMIN")))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		case dns.TypeNS:
			if HasSuffixCaseInsensitive(q.Name, os.Getenv("DOMAIN")) {
				rr1, err1 := dns.NewRR(fmt.Sprintf("%s 60 NS %s", q.Name, os.Getenv("TARGET_NS")))
				if err1 == nil {
					m.Answer = append(m.Answer, rr1)
				}
			}
		case dns.TypeA:
			if HasSuffixCaseInsensitive(q.Name, os.Getenv("DOMAIN")) {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 A %s", q.Name, os.Getenv("TARGET_IPV4")))
				if err == nil {
					m.Answer = append(m.Answer, rr)

					host, _, err := net.SplitHostPort(w.RemoteAddr().String())
					if err == nil {
						if err == nil {
							mu.Lock()
							rec, exists := dnsData[q.Name]
							if exists {
								ipExists := false
								for _, ip := range rec.ips {
									if ip == host {
										ipExists = true
										break
									}
								}
								if !ipExists {
									rec.ips = append(rec.ips, host)
								}
							} else {
								rec = record{ips: []string{host}, timestamp: time.Now()}
							}
							dnsData[q.Name] = rec
							mu.Unlock()
						}
					}
				}
			}
		case dns.TypeAAAA:
			if HasSuffixCaseInsensitive(q.Name, os.Getenv("DOMAIN")) {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 AAAA %s", q.Name, os.Getenv("TARGET_IPV6")))
				if err == nil {
					m.Answer = append(m.Answer, rr)

					host, _, err := net.SplitHostPort(w.RemoteAddr().String())
					if err == nil {
						if err == nil {
							mu.Lock()
							rec, exists := dnsData[q.Name]
							if exists {
								ipExists := false
								for _, ip := range rec.ips {
									if ip == host {
										ipExists = true
										break
									}
								}
								if !ipExists {
									rec.ips = append(rec.ips, host)
								}
							} else {
								rec = record{ips: []string{host}, timestamp: time.Now()}
							}
							dnsData[q.Name] = rec
							mu.Unlock()
						}
					}
				}
			}
		case dns.TypeHTTPS:
			if HasSuffixCaseInsensitive(q.Name, os.Getenv("DOMAIN")) {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 HTTPS 1 . alpn=\"h2\" port=443", q.Name))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}

	w.WriteMsg(m)
}

func JSONHandler(w http.ResponseWriter, r *http.Request) {
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

	mu.Lock()
	rec, ok := dnsData[host]
	mu.Unlock()

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
	for _, ip := range rec.ips {
		reverseDNS, err := lookupReverseDNS(ip)
		if err != nil {
			log.Printf("Reverse DNS lookup failed for %s: %v\n", ip, err)
		}
		servers[ip] = reverseDNS
	}

	response := map[string]interface{}{
		"domain":      host,
		"ips":         rec.ips,
		"known":       true,
		"requesterIP": requesterIP,
		"city":        geoIPCity,
		"country":     geoIPCountry,
		"isp":         geoIPASN,
		"servers":     servers,
	}
	for _, ip := range rec.ips {
		if !knownIPs[ip] {
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

func RootHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	fmt.Fprintln(w, "up")
}

func queryRoundRobinDNS(qName string) ([]string, error) {
	for i := 0; i < len(dnsServers); i++ {
		// Select the current DNS server using round-robin
		serverIndex := atomic.AddInt32(&currentDNS, 1) % int32(len(dnsServers))
		dnsServer := dnsServers[serverIndex]

		log.Printf("Querying DNS server: %s for %s", dnsServer, qName)

		// Query the selected DNS server
		client := new(dns.Client)
		message := new(dns.Msg)
		message.SetQuestion(dns.Fqdn(qName), dns.TypeTXT)

		response, _, err := client.Exchange(message, dnsServer)
		if err != nil {
			log.Printf("Failed to query %s: %v\n", dnsServer, err)
			continue // Try the next server
		}

		log.Printf("DNS Response from %s: %+v", dnsServer, response)

		// Extract TXT records from the response
		var txtRecords []string
		for _, answer := range response.Answer {
			if txt, ok := answer.(*dns.TXT); ok {
				txtRecords = append(txtRecords, txt.Txt...)
			}
		}

		// If successful, return the records
		if len(txtRecords) > 0 {
			return txtRecords, nil
		}
	}
	return nil, fmt.Errorf("all upstream DNS servers failed")
}

func startDNSServer() {
	dns.HandleFunc(".", DNSHandler)
	server := &dns.Server{Addr: ":53", Net: "udp"}
	log.Println("Starting DNS server on port 53")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start DNS server: %s", err)
	}
}

func startHTTPServer() {
	http.HandleFunc("/json", JSONHandler)
	http.HandleFunc("/", RootHandler)
	log.Println("Starting HTTP server on port 8080")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		log.Fatalf("Failed to start HTTP server: %s", err)
	}
}

func startCleanupRoutine() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		cleanupExpiredRecords()
	}
}

func cleanupExpiredRecords() {
	mu.Lock()
	defer mu.Unlock()
	for k, v := range dnsData {
		if time.Since(v.timestamp) > ttl {
			delete(dnsData, k)
		}
	}
}

func HasSuffixCaseInsensitive(name, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix))
}

func main() {
	go startDNSServer()
	go startCleanupRoutine()
	startHTTPServer()
}
