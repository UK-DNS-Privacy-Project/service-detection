package models

import (
	"testing"
	"time"
)

func TestRecord(t *testing.T) {
	now := time.Now()

	t.Run("Record with IPs and Timestamp", func(t *testing.T) {
		rec := Record{
			IPs:       []string{"1.1.1.1", "8.8.8.8"},
			Timestamp: now,
		}
		if len(rec.IPs) != 2 {
			t.Errorf("expected 2 IPs, got %d", len(rec.IPs))
		}
		if !rec.Timestamp.Equal(now) {
			t.Errorf("expected timestamp %v, got %v", now, rec.Timestamp)
		}
	})

	t.Run("Record with empty IPs", func(t *testing.T) {
		rec := Record{
			IPs:       []string{},
			Timestamp: now,
		}
		if len(rec.IPs) != 0 {
			t.Errorf("expected 0 IPs, got %d", len(rec.IPs))
		}
	})

	t.Run("Record with zero Timestamp", func(t *testing.T) {
		rec := Record{
			IPs:       []string{"127.0.0.1"},
			Timestamp: time.Time{},
		}
		if !rec.Timestamp.IsZero() {
			t.Errorf("expected zero timestamp, got %v", rec.Timestamp)
		}
	})
}
