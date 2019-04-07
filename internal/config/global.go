package config

import (
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/Lookyan/netramesh/pkg/log"
)

const (
	defaultRequestIdHeaderName = "X-Request-Id"
	defaultXSourceName         = "X-Source"
	defaultXSourceValue        = "netra"
)

type NetraConfig struct {
	Port                          int
	PprofPort                     int
	ServiceName                   string
	TracingContextExpiration      time.Duration
	TracingContextCleanupInterval time.Duration
	LoggerLevel                   log.Level
	HTTPProtoPorts                map[string]struct{}
}

var netraConfig = NetraConfig{
	Port:                          14956,
	PprofPort:                     14957,
	TracingContextExpiration:      5 * time.Second,
	TracingContextCleanupInterval: 1 * time.Second,
	HTTPProtoPorts:                make(map[string]struct{}),
}

func GetNetraConfig() NetraConfig {
	return netraConfig
}

func SetServiceName(serviceName string) {
	netraConfig.ServiceName = serviceName
}

type HTTPConfig struct {
	HeadersMap          map[string]string
	CookiesMap          map[string]string
	RequestIdHeaderName string
	XSourceHeaderName   string
	XSourceValue        string
}

var httpConfig = HTTPConfig{
	HeadersMap:          map[string]string{},
	CookiesMap:          map[string]string{},
	RequestIdHeaderName: defaultRequestIdHeaderName,
	XSourceHeaderName:   defaultXSourceName,
	XSourceValue:        defaultXSourceValue,
}

func GetHTTPConfig() HTTPConfig {
	return httpConfig
}

const (
	envNetraPort                          = "NETRA_PORT"
	envNetraPprofPort                     = "NETRA_PPROF_PORT"
	envNetraTracingContextExpiration      = "NETRA_TRACING_CONTEXT_EXPIRATION_MILLISECONDS"
	envNetraTracingContextCleanupInterval = "NETRA_TRACING_CONTEXT_CLEANUP_INTERVAL"
	envNetraHTTPPorts                     = "NETRA_HTTP_PORTS"
	envHttpHeaderTagMap                   = "HTTP_HEADER_TAG_MAP"
	envHttpCookieTagMap                   = "HTTP_COOKIE_TAG_MAP"
	envHttpRequestIdHeaderName            = "NETRA_HTTP_REQUEST_ID_HEADER_NAME"
	envHttpXSourceHeaderName              = "NETRA_HTTP_X_SOURCE_HEADER_NAME"
	envHTTPXSourceValue                   = "NETRA_HTTP_X_SOURCE_VALUE"
)

func GlobalConfigFromENV(logger *log.Logger) error {
	if v := os.Getenv(envHttpHeaderTagMap); v != "" {
		pairs := strings.Split(v, ",")
		for _, pair := range pairs {
			kv := strings.SplitN(pair, ":", 2)
			if len(kv) < 2 {
				continue
			}
			httpConfig.HeadersMap[kv[0]] = kv[1]
			logger.Infof("loaded header to tag mapping: %s => %s", kv[0], kv[1])
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
			logger.Infof("loaded cookie to tag mapping: %s => %s", kv[0], kv[1])
		}
	}
	if v := os.Getenv(envNetraPort); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		netraConfig.Port = p
	}
	if v := os.Getenv(envNetraPprofPort); v != "" {
		p, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		netraConfig.PprofPort = p
	}
	if v := os.Getenv(envNetraTracingContextExpiration); v != "" {
		exp, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		netraConfig.TracingContextExpiration = time.Duration(exp) * time.Millisecond
	}
	if v := os.Getenv(envNetraTracingContextCleanupInterval); v != "" {
		c, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		netraConfig.TracingContextCleanupInterval = time.Duration(c) * time.Millisecond
	}
	if v := os.Getenv(envNetraHTTPPorts); v != "" {
		ports := strings.Split(v, ",")
		for _, port := range ports {
			// check that port is int
			_, err := strconv.Atoi(port)
			if err != nil {
				return err
			}
			netraConfig.HTTPProtoPorts[port] = struct{}{}
		}
	}
	if v := os.Getenv(envHttpRequestIdHeaderName); v != "" {
		httpConfig.RequestIdHeaderName = v
	}
	if v := os.Getenv(envHttpXSourceHeaderName); v != "" {
		httpConfig.XSourceHeaderName = v
	}
	if v := os.Getenv(envHTTPXSourceValue); v != "" {
		httpConfig.XSourceValue = v
	}

	return nil
}
