package protocol

type NetRequest interface {
	StartRequest()
	StopRequest()
}
