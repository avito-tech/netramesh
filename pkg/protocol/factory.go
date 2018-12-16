package protocol

func GetNetworkHandler(proto Proto) NetHandler {
	switch proto {
	case HTTPProto:
		return NewHTTPHandler()
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
