package protocol

import (
	"io"
	"sync"
)

type NetHandler interface {
	// HandleRequest should get all data from r, process it and write result to w
	HandleRequest(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInboundConn bool)
	// HandleResponse should get all data from r, process it and write result to w
	HandleResponse(r io.ReadCloser, w io.WriteCloser, netRequest NetRequest, isInboundConn bool)
}

var bufferPool = sync.Pool{
	New: func() interface{} { return make([]byte, 0xffff) },
}
