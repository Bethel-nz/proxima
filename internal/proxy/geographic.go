package proxy

var countryToIP = map[string]string{
	"US": "204.79.197.200", // USA
	"GB": "176.227.219.0",  // UK
	"JP": "103.152.220.0",  // Japan
	"DE": "85.214.132.117", // Germany
	"FR": "91.189.92.20",   // France
	"CA": "99.79.177.77",   // Canada
	"AU": "1.1.1.1",        // Australia
	"BR": "200.160.2.3",    // Brazil
	// Add more countries as needed
}

type GeoConfig struct {
	CountryCode string
	ForwardedIP string
}

func getForwardedIP(countryCode string) string {
	if ip, ok := countryToIP[countryCode]; ok {
		return ip
	}
	return ""
}

var countryToLanguage = map[string]string{
	"US": "en-US,en;q=0.9",
	"GB": "en-GB,en;q=0.9",
	"JP": "ja-JP,ja;q=0.9,en;q=0.8",
	"DE": "de-DE,de;q=0.9,en;q=0.8",
	"FR": "fr-FR,fr;q=0.9,en;q=0.8",
	// Add more as needed
}

func getLanguageForCountry(countryCode string) string {
	if lang, ok := countryToLanguage[countryCode]; ok {
		return lang
	}
	return "en-US,en;q=0.9"
}
