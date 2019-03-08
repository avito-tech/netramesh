package protocol

import (
	"github.com/patrickmn/go-cache"

	"github.com/Lookyan/netramesh/pkg/log"
)

func GetNetworkHandler(proto Proto, logger *log.Logger, tracingContextMapping *cache.Cache) NetHandler {
	switch proto {
	case HTTPProto:
		return NewHTTPHandler(logger, tracingContextMapping)
	case TCPProto:
		return NewTCPHandler(logger)
	default:
		return NewTCPHandler(logger)
	}
}

func GetNetRequest(proto Proto, logger *log.Logger) NetRequest {
	switch proto {
	case HTTPProto:
		return NewNetHTTPRequest(logger)
	case TCPProto:
		return NewNetTCPRequest(logger)
	default:
		return NewNetTCPRequest(logger)
	}
}
