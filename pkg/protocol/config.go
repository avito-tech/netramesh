package protocol

import (
	"log"
	"os"
	"strings"
)

type HTTPConfig struct {
	HeadersMap             map[string]string
	CookiesMap             map[string]string
	HostSubstitutionHeader string
}

var httpConfig = HTTPConfig{
	HeadersMap:             map[string]string{},
	CookiesMap:             map[string]string{},
	HostSubstitutionHeader: "",
}

func getHTTPConfig() HTTPConfig {
	return httpConfig
}

const (
	envHTTPHeaderTagMap           = "HTTP_HEADER_TAG_MAP"
	envHTTPCookieTagMap           = "HTTP_COOKIE_TAG_MAP"
	envHTTPHostSubstitutionHeader = "HTTP_HOST_SUBSTITUTION_HEADER"
)

func GlobalConfigFromENV() {
	if v := os.Getenv(envHTTPHeaderTagMap); v != "" {
		pairs := strings.Split(v, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) < 2 {
				continue
			}
			httpConfig.HeadersMap[kv[0]] = kv[1]
			log.Printf("loaded header to tag mapping: %s => %s", kv[0], kv[1])
		}
	}
	if v := os.Getenv(envHTTPCookieTagMap); v != "" {
		pairs := strings.Split(v, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) < 2 {
				continue
			}
			httpConfig.CookiesMap[kv[0]] = kv[1]
			log.Printf("loaded cookie to tag mapping: %s => %s", kv[0], kv[1])
		}
	}
	if v := os.Getenv(envHTTPHostSubstitutionHeader); v != "" {
		// TODO: parse to map
		httpConfig.HostSubstitutionHeader = v
	}
}
