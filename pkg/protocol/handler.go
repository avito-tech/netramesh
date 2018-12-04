package protocol

import "io"

type NetHandler interface {
	HandleRequest(pr *io.PipeReader, pw *io.PipeWriter, netRequest NetRequest)
	HandleResponse(pr *io.PipeReader, pw *io.PipeWriter, netRequest NetRequest)
}
