package api

import (
	"net"

	"github.com/oschwald/geoip2-golang"
)

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
