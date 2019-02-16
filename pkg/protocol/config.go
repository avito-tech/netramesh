package protocol

import (
	"log"
	"os"
	"strings"
)

type HttpConfig struct {
	HeadersMap map[string]string
	CookiesMap map[string]string
}

var httpConfig = HttpConfig{
	HeadersMap: map[string]string{},
	CookiesMap: map[string]string{},
}

func getHttpConfig() HttpConfig {
	return httpConfig
}

const (
	envHttpHeaderTagMap = "HTTP_HEADER_TAG_MAP"
	envHttpCookieTagMap = "HTTP_COOKIE_TAG_MAP"
)

func GlobalConfigFromENV() {
	if v := os.Getenv(envHttpHeaderTagMap); v != "" {
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
	if v := os.Getenv(envHttpCookieTagMap); v != "" {
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
}
