package protocol

import (
	"github.com/patrickmn/go-cache"
)

func GetNetworkHandler(proto Proto, tracingContextMapping *cache.Cache) NetHandler {
	switch proto {
	case HTTPProto:
		return NewHTTPHandler(tracingContextMapping)
	case TCPProto:
		return NewTCPHandler()
	default:
		return NewTCPHandler()
	}
}

func GetNetRequest(proto Proto) NetRequest {
	switch proto {
	case HTTPProto:
		return NewNetHTTPRequest()
	case TCPProto:
		return NewNetTCPRequest()
	default:
		return NewNetTCPRequest()
	}
}
