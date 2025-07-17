package models

import "time"

type Record struct {
	IPs       []string
	Timestamp time.Time
}
