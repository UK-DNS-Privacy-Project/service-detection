package resolver

import (
	"fmt"
	"log"
	"net"
	"strings"
	"sync/atomic"
	"time"

	"dnsprivacy.org.uk/resolver/internal/config"
	"dnsprivacy.org.uk/resolver/internal/models"

	"github.com/miekg/dns"
)

func Start() {
	dns.HandleFunc(".", dnsHandler)
	server := &dns.Server{Addr: ":53", Net: "udp"}
	log.Println("Starting DNS server on port 53")
	err := server.ListenAndServe()
	if err != nil {
		log.Fatalf("Failed to start DNS server: %s", err)
	}
}

func dnsHandler(w dns.ResponseWriter, r *dns.Msg) {
	m := new(dns.Msg)
	m.SetReply(r)
	m.Authoritative = true

	for _, q := range r.Question {
		log.Println("Query Received", q.Qtype, q.Name)

		if strings.ToLower(q.Name) == config.ACMEChallengeDomain {
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
			if hasSuffixCaseInsensitive(q.Name, config.Domain) {
				rr, err := dns.NewRR(fmt.Sprintf("%s SOA %s %s 2025012400 3600 600 604800 86400", q.Name, config.Domain, config.SOAAdmin))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		case dns.TypeNS:
			if hasSuffixCaseInsensitive(q.Name, config.Domain) {
				rr1, err1 := dns.NewRR(fmt.Sprintf("%s 60 NS %s", q.Name, config.TargetNS))
				if err1 == nil {
					m.Answer = append(m.Answer, rr1)
				}
			}
		case dns.TypeA:
			if hasSuffixCaseInsensitive(q.Name, config.Domain) {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 A %s", q.Name, config.TargetIPV4))
				if err == nil {
					m.Answer = append(m.Answer, rr)

					host, _, err := net.SplitHostPort(w.RemoteAddr().String())
					if err == nil {
						config.Mutex.Lock()
						rec, exists := config.DNSData[q.Name]
						if exists {
							ipExists := false
							for _, ip := range rec.IPs {
								if ip == host {
									ipExists = true
									break
								}
							}
							if !ipExists {
								rec.IPs = append(rec.IPs, host)
							}
						} else {
							rec = models.Record{IPs: []string{host}, Timestamp: time.Now()}
						}
						config.DNSData[q.Name] = rec
						config.Mutex.Unlock()
					}
				}
			}
		case dns.TypeAAAA:
			if hasSuffixCaseInsensitive(q.Name, config.Domain) {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 AAAA %s", q.Name, config.TargetIPV6))
				if err == nil {
					m.Answer = append(m.Answer, rr)

					host, _, err := net.SplitHostPort(w.RemoteAddr().String())
					if err == nil {
						config.Mutex.Lock()
						rec, exists := config.DNSData[q.Name]
						if exists {
							ipExists := false
							for _, ip := range rec.IPs {
								if ip == host {
									ipExists = true
									break
								}
							}
							if !ipExists {
								rec.IPs = append(rec.IPs, host)
							}
						} else {
							rec = models.Record{IPs: []string{host}, Timestamp: time.Now()}
						}
						config.DNSData[q.Name] = rec
						config.Mutex.Unlock()
					}
				}
			}
		case dns.TypeHTTPS:
			if hasSuffixCaseInsensitive(q.Name, config.Domain) {
				rr, err := dns.NewRR(fmt.Sprintf("%s 60 HTTPS 1 . alpn=\"h2\" port=443", q.Name))
				if err == nil {
					m.Answer = append(m.Answer, rr)
				}
			}
		}
	}

	w.WriteMsg(m)
}

func queryRoundRobinDNS(qName string) ([]string, error) {
	for i := 0; i < len(config.DNSServers); i++ {
		// Select the current DNS server using round-robin
		serverIndex := atomic.AddInt32(&config.CurrentDNS, 1) % int32(len(config.DNSServers))
		dnsServer := config.DNSServers[serverIndex]

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

func hasSuffixCaseInsensitive(name, suffix string) bool {
	return strings.HasSuffix(strings.ToLower(name), strings.ToLower(suffix))
}
