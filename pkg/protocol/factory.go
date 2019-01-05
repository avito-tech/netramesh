package protocol

import "sync"

func GetNetworkHandler(proto Proto, tracingContextMapping sync.Map) NetHandler {
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
