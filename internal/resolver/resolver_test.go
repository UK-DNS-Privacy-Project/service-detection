package resolver

import (
	"net"
	"sync"
	"testing"
	"time"

	"dnsprivacy.org.uk/resolver/internal/config"
	"dnsprivacy.org.uk/resolver/internal/models"

	"github.com/miekg/dns"
)

// --- hasSuffixCaseInsensitive tests ---

func TestHasSuffixCaseInsensitive(t *testing.T) {
	tests := []struct {
		name   string
		suffix string
		want   bool
	}{
		{"example.com.", "example.com.", true},
		{"EXAMPLE.COM.", "example.com.", true},
		{"test.example.com.", "example.com.", true},
		{"test.example.com.", "EXAMPLE.COM.", true},
		{"example.org.", "example.com.", false},
	}
	for _, tt := range tests {
		got := hasSuffixCaseInsensitive(tt.name, tt.suffix)
		if got != tt.want {
			t.Errorf("hasSuffixCaseInsensitive(%q, %q) = %v, want %v", tt.name, tt.suffix, got, tt.want)
		}
	}
}

// --- queryRoundRobinDNS tests ---

func TestQueryRoundRobinDNS(t *testing.T) {
	// Start a test DNS server that returns a TXT record
	addr := "127.0.0.1:15353"
	server := &dns.Server{Addr: addr, Net: "udp"}
	dns.HandleFunc("acme.test.", func(w dns.ResponseWriter, r *dns.Msg) {
		m := new(dns.Msg)
		m.SetReply(r)
		m.Authoritative = true
		m.Answer = append(m.Answer, &dns.TXT{
			Hdr: dns.RR_Header{Name: "acme.test.", Rrtype: dns.TypeTXT, Class: dns.ClassINET, Ttl: 60},
			Txt: []string{"challenge-token"},
		})
		w.WriteMsg(m)
	})
	go server.ListenAndServe()
	defer server.Shutdown()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Mock config
	config.DNSServers = []string{addr}
	config.CurrentDNS = 0

	txts, err := queryRoundRobinDNS("acme.test.")
	if err != nil {
		t.Fatalf("expected no error, got %v", err)
	}
	if len(txts) != 1 || txts[0] != "challenge-token" {
		t.Errorf("expected [challenge-token], got %v", txts)
	}

	// Test with no servers
	config.DNSServers = []string{}
	_, err = queryRoundRobinDNS("acme.test.")
	if err == nil {
		t.Error("expected error when no DNS servers configured")
	}
}

// --- dnsHandler tests ---

type mockResponseWriter struct {
	msg    *dns.Msg
	remote net.Addr
}

func (m *mockResponseWriter) WriteMsg(msg *dns.Msg) error {
	m.msg = msg
	return nil
}
func (m *mockResponseWriter) RemoteAddr() net.Addr      { return m.remote }
func (m *mockResponseWriter) LocalAddr() net.Addr       { return &net.UDPAddr{IP: net.IPv4zero, Port: 53} }
func (m *mockResponseWriter) Write([]byte) (int, error) { return 0, nil }
func (m *mockResponseWriter) Close() error              { return nil }
func (m *mockResponseWriter) TsigStatus() error         { return nil }
func (m *mockResponseWriter) TsigTimersOnly(bool)       {}
func (m *mockResponseWriter) Hijack()                   {}
func (m *mockResponseWriter) Network() string           { return "udp" }

func TestDNSHandler_ARecord(t *testing.T) {
	// Setup config
	config.Domain = "example.com."
	config.TargetIPV4 = "1.2.3.4"
	config.DNSData = make(map[string]models.Record)
	config.Mutex = sync.Mutex{}

	q := dns.Question{Name: "test.example.com.", Qtype: dns.TypeA, Qclass: dns.ClassINET}
	msg := new(dns.Msg)
	msg.Question = []dns.Question{q}

	remote := &net.UDPAddr{IP: net.ParseIP("5.6.7.8"), Port: 12345}
	w := &mockResponseWriter{remote: remote}

	dnsHandler(w, msg)

	if w.msg == nil || len(w.msg.Answer) == 0 {
		t.Fatal("expected an answer in DNS response")
	}
	found := false
	for _, rr := range w.msg.Answer {
		if a, ok := rr.(*dns.A); ok && a.A.String() == "1.2.3.4" {
			found = true
		}
	}
	if !found {
		t.Error("expected A record with 1.2.3.4 in answer")
	}
}

func TestDNSHandler_HTTPSRecord(t *testing.T) {
	config.Domain = "example.com."
	q := dns.Question{Name: "test.example.com.", Qtype: dns.TypeHTTPS, Qclass: dns.ClassINET}
	msg := new(dns.Msg)
	msg.Question = []dns.Question{q}
	w := &mockResponseWriter{remote: &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}}
	dnsHandler(w, msg)
	if w.msg == nil || len(w.msg.Answer) == 0 {
		t.Fatal("expected an answer in DNS response")
	}
}

func TestDNSHandler_SOARecord(t *testing.T) {
	config.Domain = "example.com."
	config.SOAAdmin = "admin.example.com."
	q := dns.Question{Name: "test.example.com.", Qtype: dns.TypeSOA, Qclass: dns.ClassINET}
	msg := new(dns.Msg)
	msg.Question = []dns.Question{q}
	w := &mockResponseWriter{remote: &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}}
	dnsHandler(w, msg)
	if w.msg == nil || len(w.msg.Answer) == 0 {
		t.Fatal("expected SOA record in DNS response")
	}
}

func TestDNSHandler_NSRecord(t *testing.T) {
	config.Domain = "example.com."
	config.TargetNS = "ns1.example.com."
	q := dns.Question{Name: "test.example.com.", Qtype: dns.TypeNS, Qclass: dns.ClassINET}
	msg := new(dns.Msg)
	msg.Question = []dns.Question{q}
	w := &mockResponseWriter{remote: &net.UDPAddr{IP: net.IPv4(1, 1, 1, 1), Port: 1234}}
	dnsHandler(w, msg)
	if w.msg == nil || len(w.msg.Answer) == 0 {
		t.Fatal("expected NS record in DNS response")
	}
}
