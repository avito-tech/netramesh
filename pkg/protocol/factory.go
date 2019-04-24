package protocol

import (
	"github.com/patrickmn/go-cache"

	"github.com/Lookyan/netramesh/pkg/log"
)

var httpHandler *HTTPHandler
var tcpHandler *TCPHandler
var netTCPRequest *NetTCPRequest

func InitHandlerRequest(logger *log.Logger, tracingContextMapping *cache.Cache) {
	httpHandler = NewHTTPHandler(logger, tracingContextMapping)
	tcpHandler = NewTCPHandler(logger)
	netTCPRequest = NewNetTCPRequest(logger)
}

func GetNetworkHandler(proto Proto, logger *log.Logger, tracingContextMapping *cache.Cache) NetHandler {
	switch proto {
	case HTTPProto:
		return httpHandler
	case TCPProto:
		return tcpHandler
	default:
		return tcpHandler
	}
}

func GetNetRequest(proto Proto, isInbound bool, logger *log.Logger, tracingContextMapping *cache.Cache) NetRequest {
	switch proto {
	case HTTPProto:
		return NewNetHTTPRequest(logger, isInbound, tracingContextMapping)
	case TCPProto:
		return netTCPRequest
	default:
		return netTCPRequest
	}
}
