package geoip

import (
	"log"
	"net/netip"

	"github.com/oschwald/maxminddb-golang/v2"
)

/*
	{
	  "city": {
	    "geoname_id": 5368361,
	    "names": {
	      "de": "Los Angeles",
	      "en": "Los Angeles",
	      "es": "Los Ángeles",
	      "fr": "Los Angeles",
	      "ja": "ロサンゼルス",
	      "pt-BR": "Los Angeles",
	      "ru": "Лос-Анджелес",
	      "zh-CN": "洛杉矶"
	    }
	  },
	  "continent": {
	    "code": "NA",
	    "geoname_id": 6255149,
	    "names": {
	      "de": "Nordamerika",
	      "en": "North America",
	      "es": "Norteamérica",
	      "fr": "Amérique du Nord",
	      "ja": "北アメリカ",
	      "pt-BR": "América do Norte",
	      "ru": "Северная Америка",
	      "zh-CN": "北美洲"
	    }
	  },
	  "country": {
	    "geoname_id": 6252001,
	    "iso_code": "US",
	    "names": {
	      "de": "USA",
	      "en": "United States",
	      "es": "Estados Unidos",
	      "fr": "États Unis",
	      "ja": "アメリカ",
	      "pt-BR": "EUA",
	      "ru": "США",
	      "zh-CN": "美国"
	    }
	  },
	  "location": {
	    "accuracy_radius": 20,
	    "latitude": 34.0481,
	    "longitude": -118.2531,
	    "metro_code": 803,
	    "time_zone": "America/Los_Angeles"
	  },
	  "postal": {
	    "code": "90014"
	  },
	  "registered_country": {
	    "geoname_id": 6251999,
	    "iso_code": "CA",
	    "names": {
	      "de": "Kanada",
	      "en": "Canada",
	      "es": "Canadá",
	      "fr": "Canada",
	      "ja": "カナダ",
	      "pt-BR": "Canadá",
	      "ru": "Канада",
	      "zh-CN": "加拿大"
	    }
	  },
	  "subdivisions": [
	    {
	      "geoname_id": 5332921,
	      "iso_code": "CA",
	      "names": {
	        "de": "Kalifornien",
	        "en": "California",
	        "es": "California",
	        "fr": "Californie",
	        "ja": "カリフォルニア州",
	        "pt-BR": "Califórnia",
	        "ru": "Калифорния",
	        "zh-CN": "加州"
	      }
	    }
	  ]
	}
*/
type IpLookUp struct {
	City struct {
		GeoNameID uint              `maxminddb:"geoname_id"`
		Names     map[string]string `maxminddb:"names"`
	} `maxminddb:"city"`
	Country struct {
		IsoCode string `maxminddb:"iso_code"`
	} `maxminddb:"country"`
	Location struct {
		AccuracyRadius uint16 `maxminddb:"accuracy_radius"`
	} `maxminddb:"location"`
}

var geoIPDb *maxminddb.Reader

func Init(filePath string) *maxminddb.Reader {
	if filePath == "" {
		filePath = "GeoLite2-City.mmdb"
	}
	var err error
	geoIPDb, err = maxminddb.Open(filePath)
	if err != nil {
		log.Println("Unable to load '", filePath, "'.")
	}
	log.Println("Inited GeoIP")
	return geoIPDb
}

func Close() {
	if geoIPDb != nil {
		geoIPDb.Close()
	}
}

func GeoIp(ip netip.Addr) any {
	var record any
	err := geoIPDb.Lookup(ip).Decode(&record)
	if err != nil {
		log.Println(err)
		return nil
	}
	return record
}
