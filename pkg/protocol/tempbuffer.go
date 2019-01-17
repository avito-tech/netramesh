package protocol

import (
	"bytes"
	"sync"
)

var tempWriterBufferPool = sync.Pool{
	New: func() interface{} { return &bytes.Buffer{} },
}

// TempWriter allows write to temp buffer and retrieve it in case we need it
type TempWriter struct {
	released bool
	buf      *bytes.Buffer
}

// NewTempWriter creates new instance of TempWriter
func NewTempWriter() *TempWriter {
	return &TempWriter{
		released: false,
		buf:      tempWriterBufferPool.Get().(*bytes.Buffer),
	}
}

// Write writes bytes into temp buffer if it
func (tw *TempWriter) Write(b []byte) (n int, err error) {
	if !tw.released {
		return tw.buf.Write(b)
	}
	return len(b), nil
}

// Read reads bytes from temp buffer
func (tw *TempWriter) Read(p []byte) (n int, err error) {
	return tw.buf.Read(p)
}

// Release releases writer from writing to cache
func (tw *TempWriter) Release() {
	tw.released = true
}

// Close stub
func (tw *TempWriter) Close() error {
	tw.buf.Truncate(0)
	tempWriterBufferPool.Put(tw.buf)
	return nil
}
