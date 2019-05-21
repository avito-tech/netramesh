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
	defaultRoutingHeaderName   = "X-Route"
	defaultXSourceValue        = "netra"
)

type NetraConfig struct {
	Port                          uint16
	PprofPort                     uint16
	PrometheusPort                uint16
	ServiceName                   string
	TracingContextExpiration      time.Duration
	TracingContextCleanupInterval time.Duration
	RoutingContextExpiration      time.Duration
	RoutingContextCleanupInterval time.Duration
	LoggerLevel                   log.Level
	HTTPProtoPorts                map[string]struct{}
}

var netraConfig = NetraConfig{
	Port:                          14956,
	PprofPort:                     14957,
	PrometheusPort:                14958,
	TracingContextExpiration:      5 * time.Second,
	TracingContextCleanupInterval: 1 * time.Second,
	RoutingContextExpiration:      5 * time.Second,
	RoutingContextCleanupInterval: 1 * time.Second,
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
	RoutingEnabled      bool
	RoutingHeaderName   string
}

var httpConfig = HTTPConfig{
	HeadersMap:          map[string]string{},
	CookiesMap:          map[string]string{},
	RequestIdHeaderName: defaultRequestIdHeaderName,
	XSourceHeaderName:   defaultXSourceName,
	XSourceValue:        defaultXSourceValue,
	RoutingEnabled:      false,
	RoutingHeaderName:   defaultRoutingHeaderName,
}

func GetHTTPConfig() HTTPConfig {
	return httpConfig
}

const (
	envNetraPort                          = "NETRA_PORT"
	envNetraPprofPort                     = "NETRA_PPROF_PORT"
	envNetraPrometheusPort                = "NETRA_PROMETHEUS_PORT"
	envNetraTracingContextExpiration      = "NETRA_TRACING_CONTEXT_EXPIRATION_MILLISECONDS"
	envNetraTracingContextCleanupInterval = "NETRA_TRACING_CONTEXT_CLEANUP_INTERVAL"
	envNetraRoutingContextExpiration      = "NETRA_ROUTING_CONTEXT_EXPIRATION_MILLISECONDS"
	envNetraRoutingContextCleanupInterval = "NETRA_ROUTING_CONTEXT_CLEANUP_INTERVAL"
	envNetraHTTPPorts                     = "NETRA_HTTP_PORTS"
	envHttpHeaderTagMap                   = "HTTP_HEADER_TAG_MAP"
	envHttpCookieTagMap                   = "HTTP_COOKIE_TAG_MAP"
	envHttpRequestIdHeaderName            = "NETRA_HTTP_REQUEST_ID_HEADER_NAME"
	envHttpXSourceHeaderName              = "NETRA_HTTP_X_SOURCE_HEADER_NAME"
	envHTTPXSourceValue                   = "NETRA_HTTP_X_SOURCE_VALUE"
	envHTTPRoutingEnabled                 = "NETRA_HTTP_ROUTING_ENABLED"
	envHTTPRoutingHeader                  = "NETRA_HTTP_ROUTING_HEADER_NAME"
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
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return err
		}
		netraConfig.Port = uint16(p)
	}
	if v := os.Getenv(envNetraPprofPort); v != "" {
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return err
		}
		netraConfig.PprofPort = uint16(p)
	}
	if v := os.Getenv(envNetraPrometheusPort); v != "" {
		p, err := strconv.ParseUint(v, 10, 16)
		if err != nil {
			return err
		}
		netraConfig.PrometheusPort = uint16(p)
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
	if v := os.Getenv(envNetraRoutingContextExpiration); v != "" {
		exp, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		netraConfig.RoutingContextExpiration = time.Duration(exp) * time.Millisecond
	}
	if v := os.Getenv(envNetraRoutingContextCleanupInterval); v != "" {
		c, err := strconv.Atoi(v)
		if err != nil {
			return err
		}
		netraConfig.RoutingContextCleanupInterval = time.Duration(c) * time.Millisecond
	}
	if v := os.Getenv(envNetraHTTPPorts); v != "" {
		ports := strings.Split(v, ",")
		for _, port := range ports {
			// check whether port is valid
			_, err := strconv.ParseUint(port, 10, 16)
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
	if v := os.Getenv(envHTTPRoutingEnabled); v != "" {
		if v == "true" {
			httpConfig.RoutingEnabled = true
		}
	}
	if v := os.Getenv(envHTTPRoutingHeader); v != "" {
		httpConfig.RoutingHeaderName = v
	}

	return nil
}
