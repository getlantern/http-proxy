// Package geo provides functionality for looking up country codes based on
// IP addresses.
package geo

import (
	"fmt"
	"net"
	"sync/atomic"
	"time"

	"github.com/getlantern/golog"
	"github.com/getlantern/keepcurrent"
	geoip2 "github.com/oschwald/geoip2-golang"
)

var (
	log = golog.LoggerFor("http-proxy-lantern.geo")

	geolite2_url = "https://download.maxmind.com/app/geoip_download?license_key=%s&edition_id=GeoLite2-Country&suffix=tar.gz"
)

// Lookup allows looking up the country for an IP address
type Lookup interface {
	// CountryCode looks up the 2 digit ISO 3166 country code in upper case for
	// the given IP address and returns "" if there was an error.
	CountryCode(ip net.IP) string
}

// NoLookup is a Lookup implementation which always return empty country code.
type NoLookup struct{}

func (l NoLookup) CountryCode(ip net.IP) string { return "" }

type lookup struct {
	runner *keepcurrent.Runner
	db     atomic.Value
}

// New constructs a new Lookup from the MaxMind GeoLite2 Country database with
// the given license key and keeps in sync with it every day. It saves the
// database file to filePath and uses the file if available.
func New(licenseKey string, filePath string) Lookup {
	chDB := make(chan []byte)
	runner := keepcurrent.New(
		keepcurrent.FromTarGz(keepcurrent.FromWeb(
			fmt.Sprintf(geolite2_url, licenseKey)),
			"GeoLite2-Country.mmdb"),
		keepcurrent.ToFile(filePath),
		keepcurrent.ToChannel(chDB))
	runner.InitFrom(keepcurrent.FromFile(filePath))
	runner.OnSourceError = func(err error) {
		log.Errorf("Error fetching geo database: %v", err)
	}
	runner.Start(24 * time.Hour)
	v := &lookup{runner: runner}
	go func() {
		for data := range chDB {
			db, err := geoip2.FromBytes(data)
			if err != nil {
				log.Errorf("Error loading geo database: %v", err)
			} else {
				v.db.Store(db)
			}
		}
	}()
	return v
}

func (l *lookup) CountryCode(ip net.IP) string {
	if db := l.db.Load(); db != nil {
		geoData, err := db.(*geoip2.Reader).Country(ip)
		if err != nil {
			log.Debugf("Unable to look up ip address %s: %s", ip, err)
			return ""
		}
		return geoData.Country.IsoCode
	}
	return ""
}
