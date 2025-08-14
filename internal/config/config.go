package config

import (
	"os"
	"sync"
	"time"

	"dnsprivacy.org.uk/resolver/internal/models"
)

var (
	DNSData    = make(map[string]models.Record)
	Mutex      sync.Mutex
	TTL        = 5 * time.Minute // Time-to-live for each DNS record
	CurrentDNS int32
	KnownIPs   = map[string]bool{
		"209.250.227.42":                         true,
		"64.176.190.82":                          true,
		"2001:19f0:7400:13c7:5400:5ff:fe40:d1ad": true,
		"2a05:f480:3400:24fd:5400:5ff:fe40:e60b": true,
	}
	DNSServers = []string{
		os.Getenv("ACME_CHALLENGE_DNS_1"),
		os.Getenv("ACME_CHALLENGE_DNS_2"),
	}
	ACMEChallengeDomain = os.Getenv("ACME_CHALLENGE_DOMAIN")
	Domain              = os.Getenv("DOMAIN")
	SOAAdmin            = os.Getenv("SOA_ADMIN")
	TargetNS            = os.Getenv("TARGET_NS")
	TargetIPV4          = os.Getenv("TARGET_IPV4")
	TargetIPV6          = os.Getenv("TARGET_IPV6")
	Now                 = func() time.Time {
		return time.Now()
	}
)

func SetupExpiredRecordCleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		CleanExpiredRecords()
	}
}

func CleanExpiredRecords() {
	Mutex.Lock()
	defer Mutex.Unlock()
	for k, v := range DNSData {
		if time.Since(v.Timestamp) > TTL {
			delete(DNSData, k)
		}
	}
}
