package main

import (
	"dnsprivacy.org.uk/resolver/internal/api"
	"dnsprivacy.org.uk/resolver/internal/config"
	"dnsprivacy.org.uk/resolver/internal/resolver"
)

func main() {
	go resolver.Start()
	go config.SetupExpiredRecordCleanup()
	api.Start()
}
