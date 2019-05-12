package fhttp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"sync"

	"github.com/valyala/bytebufferpool"
)

// Request represents HTTP request.
//
// It is forbidden copying Request instances. Create new instances
// and use CopyTo instead.
//
// Request instance MUST NOT be used from concurrently running goroutines.
type Request struct {
	noCopy noCopy

	// Request header
	//
	// Copying Header by value is forbidden. Use pointer to Header instead.
	Header RequestHeader

	uri      URI
	postArgs Args

	bodyStream io.Reader
	w          requestBodyWriter
	body       *bytebufferpool.ByteBuffer

	multipartForm         *multipart.Form
	multipartFormBoundary string

	// Group bool members in order to reduce Request object size.
	parsedURI      bool
	parsedPostArgs bool

	keepBodyBuffer bool

	isTLS bool
}

// Response represents HTTP response.
//
// It is forbidden copying Response instances. Create new instances
// and use CopyTo instead.
//
// Response instance MUST NOT be used from concurrently running goroutines.
type Response struct {
	noCopy noCopy

	// Response header
	//
	// Copying Header by value is forbidden. Use pointer to Header instead.
	Header ResponseHeader

	bodyStream io.Reader
	w          responseBodyWriter
	body       *bytebufferpool.ByteBuffer

	// Response.Read() skips reading body if set to true.
	// Use it for reading HEAD responses.
	//
	// Response.Write() skips writing body if set to true.
	// Use it for writing HEAD responses.
	SkipBody bool

	keepBodyBuffer bool
}

// SetHost sets host for the request.
func (req *Request) SetHost(host string) {
	req.URI().SetHost(host)
}

// SetHostBytes sets host for the request.
func (req *Request) SetHostBytes(host []byte) {
	req.URI().SetHostBytes(host)
}

// Host returns the host for the given request.
func (req *Request) Host() []byte {
	return req.URI().Host()
}

// SetRequestURI sets RequestURI.
func (req *Request) SetRequestURI(requestURI string) {
	req.Header.SetRequestURI(requestURI)
	req.parsedURI = false
}

// SetRequestURIBytes sets RequestURI.
func (req *Request) SetRequestURIBytes(requestURI []byte) {
	req.Header.SetRequestURIBytes(requestURI)
	req.parsedURI = false
}

// RequestURI returns request's URI.
func (req *Request) RequestURI() []byte {
	if req.parsedURI {
		requestURI := req.uri.RequestURI()
		req.SetRequestURIBytes(requestURI)
	}
	return req.Header.RequestURI()
}

// StatusCode returns response status code.
func (resp *Response) StatusCode() int {
	return resp.Header.StatusCode()
}

// SetStatusCode sets response status code.
func (resp *Response) SetStatusCode(statusCode int) {
	resp.Header.SetStatusCode(statusCode)
}

// ConnectionClose returns true if 'Connection: close' header is set.
func (resp *Response) ConnectionClose() bool {
	return resp.Header.ConnectionClose()
}

// SetConnectionClose sets 'Connection: close' header.
func (resp *Response) SetConnectionClose() {
	resp.Header.SetConnectionClose()
}

// ConnectionClose returns true if 'Connection: close' header is set.
func (req *Request) ConnectionClose() bool {
	return req.Header.ConnectionClose()
}

// SetConnectionClose sets 'Connection: close' header.
func (req *Request) SetConnectionClose() {
	req.Header.SetConnectionClose()
}

// SendFile registers file on the given path to be used as response body
// when Write is called.
//
// Note that SendFile doesn't set Content-Type, so set it yourself
// with Header.SetContentType.
func (resp *Response) SendFile(path string) error {
	f, err := os.Open(path)
	if err != nil {
		return err
	}
	fileInfo, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}
	size64 := fileInfo.Size()
	size := int(size64)
	if int64(size) != size64 {
		size = -1
	}

	resp.Header.SetLastModified(fileInfo.ModTime())
	resp.SetBodyStream(f, size)
	return nil
}

// SetBodyStream sets request body stream and, optionally body size.
//
// If bodySize is >= 0, then the bodyStream must provide exactly bodySize bytes
// before returning io.EOF.
//
// If bodySize < 0, then bodyStream is read until io.EOF.
//
// bodyStream.Close() is called after finishing reading all body data
// if it implements io.Closer.
//
// Note that GET and HEAD requests cannot have body.
//
// See also SetBodyStreamWriter.
func (req *Request) SetBodyStream(bodyStream io.Reader, bodySize int) {
	req.ResetBody()
	req.bodyStream = bodyStream
	req.Header.SetContentLength(bodySize)
}

// SetBodyStream sets response body stream and, optionally body size.
//
// If bodySize is >= 0, then the bodyStream must provide exactly bodySize bytes
// before returning io.EOF.
//
// If bodySize < 0, then bodyStream is read until io.EOF.
//
// bodyStream.Close() is called after finishing reading all body data
// if it implements io.Closer.
//
// See also SetBodyStreamWriter.
func (resp *Response) SetBodyStream(bodyStream io.Reader, bodySize int) {
	resp.ResetBody()
	resp.bodyStream = bodyStream
	resp.Header.SetContentLength(bodySize)
}

// IsBodyStream returns true if body is set via SetBodyStream*
func (req *Request) IsBodyStream() bool {
	return req.bodyStream != nil
}

// IsBodyStream returns true if body is set via SetBodyStream*
func (resp *Response) IsBodyStream() bool {
	return resp.bodyStream != nil
}

// BodyWriter returns writer for populating response body.
//
// If used inside RequestHandler, the returned writer must not be used
// after returning from RequestHandler. Use RequestCtx.Write
// or SetBodyStreamWriter in this case.
func (resp *Response) BodyWriter() io.Writer {
	resp.w.r = resp
	return &resp.w
}

// BodyWriter returns writer for populating request body.
func (req *Request) BodyWriter() io.Writer {
	req.w.r = req
	return &req.w
}

type responseBodyWriter struct {
	r *Response
}

func (w *responseBodyWriter) Write(p []byte) (int, error) {
	w.r.AppendBody(p)
	return len(p), nil
}

type requestBodyWriter struct {
	r *Request
}

func (w *requestBodyWriter) Write(p []byte) (int, error) {
	w.r.AppendBody(p)
	return len(p), nil
}

// Body returns response body.
//
// The returned body is valid until the response modification.
func (resp *Response) Body() []byte {
	if resp.bodyStream != nil {
		bodyBuf := resp.bodyBuffer()
		bodyBuf.Reset()
		_, err := copyZeroAlloc(bodyBuf, resp.bodyStream)
		resp.closeBodyStream()
		if err != nil {
			bodyBuf.SetString(err.Error())
		}
	}
	return resp.bodyBytes()
}

func (resp *Response) bodyBytes() []byte {
	if resp.body == nil {
		return nil
	}
	return resp.body.B
}

func (req *Request) bodyBytes() []byte {
	if req.body == nil {
		return nil
	}
	return req.body.B
}

func (resp *Response) bodyBuffer() *bytebufferpool.ByteBuffer {
	if resp.body == nil {
		resp.body = responseBodyPool.Get()
	}
	return resp.body
}

func (req *Request) bodyBuffer() *bytebufferpool.ByteBuffer {
	if req.body == nil {
		req.body = requestBodyPool.Get()
	}
	return req.body
}

var (
	responseBodyPool bytebufferpool.Pool
	requestBodyPool  bytebufferpool.Pool
)

// BodyWriteTo writes request body to w.
func (req *Request) BodyWriteTo(w io.Writer) error {
	if req.bodyStream != nil {
		_, err := copyZeroAlloc(w, req.bodyStream)
		req.closeBodyStream()
		return err
	}
	if req.onlyMultipartForm() {
		return WriteMultipartForm(w, req.multipartForm, req.multipartFormBoundary)
	}
	_, err := w.Write(req.bodyBytes())
	return err
}

// BodyWriteTo writes response body to w.
func (resp *Response) BodyWriteTo(w io.Writer) error {
	if resp.bodyStream != nil {
		_, err := copyZeroAlloc(w, resp.bodyStream)
		resp.closeBodyStream()
		return err
	}
	_, err := w.Write(resp.bodyBytes())
	return err
}

// AppendBody appends p to response body.
//
// It is safe re-using p after the function returns.
func (resp *Response) AppendBody(p []byte) {
	resp.AppendBodyString(b2s(p))
}

// AppendBodyString appends s to response body.
func (resp *Response) AppendBodyString(s string) {
	resp.closeBodyStream()
	resp.bodyBuffer().WriteString(s)
}

// SetBody sets response body.
//
// It is safe re-using body argument after the function returns.
func (resp *Response) SetBody(body []byte) {
	resp.SetBodyString(b2s(body))
}

// SetBodyString sets response body.
func (resp *Response) SetBodyString(body string) {
	resp.closeBodyStream()
	bodyBuf := resp.bodyBuffer()
	bodyBuf.Reset()
	bodyBuf.WriteString(body)
}

// ResetBody resets response body.
func (resp *Response) ResetBody() {
	resp.closeBodyStream()
	if resp.body != nil {
		if resp.keepBodyBuffer {
			resp.body.Reset()
		} else {
			responseBodyPool.Put(resp.body)
			resp.body = nil
		}
	}
}

// ReleaseBody retires the response body if it is greater than "size" bytes.
//
// This permits GC to reclaim the large buffer.  If used, must be before
// ReleaseResponse.
//
// Use this method only if you really understand how it works.
// The majority of workloads don't need this method.
func (resp *Response) ReleaseBody(size int) {
	if cap(resp.body.B) > size {
		resp.closeBodyStream()
		resp.body = nil
	}
}

// ReleaseBody retires the request body if it is greater than "size" bytes.
//
// This permits GC to reclaim the large buffer.  If used, must be before
// ReleaseRequest.
//
// Use this method only if you really understand how it works.
// The majority of workloads don't need this method.
func (req *Request) ReleaseBody(size int) {
	if cap(req.body.B) > size {
		req.closeBodyStream()
		req.body = nil
	}
}

// SwapBody swaps response body with the given body and returns
// the previous response body.
//
// It is forbidden to use the body passed to SwapBody after
// the function returns.
func (resp *Response) SwapBody(body []byte) []byte {
	bb := resp.bodyBuffer()

	if resp.bodyStream != nil {
		bb.Reset()
		_, err := copyZeroAlloc(bb, resp.bodyStream)
		resp.closeBodyStream()
		if err != nil {
			bb.Reset()
			bb.SetString(err.Error())
		}
	}

	oldBody := bb.B
	bb.B = body
	return oldBody
}

// SwapBody swaps request body with the given body and returns
// the previous request body.
//
// It is forbidden to use the body passed to SwapBody after
// the function returns.
func (req *Request) SwapBody(body []byte) []byte {
	bb := req.bodyBuffer()

	if req.bodyStream != nil {
		bb.Reset()
		_, err := copyZeroAlloc(bb, req.bodyStream)
		req.closeBodyStream()
		if err != nil {
			bb.Reset()
			bb.SetString(err.Error())
		}
	}

	oldBody := bb.B
	bb.B = body
	return oldBody
}

// Body returns request body.
//
// The returned body is valid until the request modification.
func (req *Request) Body() []byte {
	if req.bodyStream != nil {
		bodyBuf := req.bodyBuffer()
		bodyBuf.Reset()
		_, err := copyZeroAlloc(bodyBuf, req.bodyStream)
		req.closeBodyStream()
		if err != nil {
			bodyBuf.SetString(err.Error())
		}
	} else if req.onlyMultipartForm() {
		body, err := marshalMultipartForm(req.multipartForm, req.multipartFormBoundary)
		if err != nil {
			return []byte(err.Error())
		}
		return body
	}
	return req.bodyBytes()
}

// AppendBody appends p to request body.
//
// It is safe re-using p after the function returns.
func (req *Request) AppendBody(p []byte) {
	req.AppendBodyString(b2s(p))
}

// AppendBodyString appends s to request body.
func (req *Request) AppendBodyString(s string) {
	req.RemoveMultipartFormFiles()
	req.closeBodyStream()
	req.bodyBuffer().WriteString(s)
}

// SetBody sets request body.
//
// It is safe re-using body argument after the function returns.
func (req *Request) SetBody(body []byte) {
	req.SetBodyString(b2s(body))
}

// SetBodyString sets request body.
func (req *Request) SetBodyString(body string) {
	req.RemoveMultipartFormFiles()
	req.closeBodyStream()
	req.bodyBuffer().SetString(body)
}

// ResetBody resets request body.
func (req *Request) ResetBody() {
	req.RemoveMultipartFormFiles()
	req.closeBodyStream()
	if req.body != nil {
		if req.keepBodyBuffer {
			req.body.Reset()
		} else {
			requestBodyPool.Put(req.body)
			req.body = nil
		}
	}
}

// CopyTo copies req contents to dst except of body stream.
func (req *Request) CopyTo(dst *Request) {
	req.copyToSkipBody(dst)
	if req.body != nil {
		dst.bodyBuffer().Set(req.body.B)
	} else if dst.body != nil {
		dst.body.Reset()
	}
}

func (req *Request) copyToSkipBody(dst *Request) {
	dst.Reset()
	req.Header.CopyTo(&dst.Header)

	req.uri.CopyTo(&dst.uri)
	dst.parsedURI = req.parsedURI

	req.postArgs.CopyTo(&dst.postArgs)
	dst.parsedPostArgs = req.parsedPostArgs
	dst.isTLS = req.isTLS

	// do not copy multipartForm - it will be automatically
	// re-created on the first call to MultipartForm.
}

// CopyTo copies resp contents to dst except of body stream.
func (resp *Response) CopyTo(dst *Response) {
	resp.copyToSkipBody(dst)
	if resp.body != nil {
		dst.bodyBuffer().Set(resp.body.B)
	} else if dst.body != nil {
		dst.body.Reset()
	}
}

func (resp *Response) copyToSkipBody(dst *Response) {
	dst.Reset()
	resp.Header.CopyTo(&dst.Header)
	dst.SkipBody = resp.SkipBody
}

func swapRequestBody(a, b *Request) {
	a.body, b.body = b.body, a.body
	a.bodyStream, b.bodyStream = b.bodyStream, a.bodyStream
}

func swapResponseBody(a, b *Response) {
	a.body, b.body = b.body, a.body
	a.bodyStream, b.bodyStream = b.bodyStream, a.bodyStream
}

// URI returns request URI
func (req *Request) URI() *URI {
	req.parseURI()
	return &req.uri
}

func (req *Request) parseURI() {
	if req.parsedURI {
		return
	}
	req.parsedURI = true

	req.uri.parseQuick(req.Header.RequestURI(), &req.Header, req.isTLS)
}

// PostArgs returns POST arguments.
func (req *Request) PostArgs() *Args {
	req.parsePostArgs()
	return &req.postArgs
}

func (req *Request) parsePostArgs() {
	if req.parsedPostArgs {
		return
	}
	req.parsedPostArgs = true

	if !bytes.HasPrefix(req.Header.ContentType(), strPostArgsContentType) {
		return
	}
	req.postArgs.ParseBytes(req.bodyBytes())
}

// ErrNoMultipartForm means that the request's Content-Type
// isn't 'multipart/form-data'.
var ErrNoMultipartForm = errors.New("request has no multipart/form-data Content-Type")

// MultipartForm returns requests's multipart form.
//
// Returns ErrNoMultipartForm if request's Content-Type
// isn't 'multipart/form-data'.
//
// RemoveMultipartFormFiles must be called after returned multipart form
// is processed.
//func (req *Request) MultipartForm() (*multipart.Form, error) {
//	if req.multipartForm != nil {
//		return req.multipartForm, nil
//	}
//
//	req.multipartFormBoundary = string(req.Header.MultipartFormBoundary())
//	if len(req.multipartFormBoundary) == 0 {
//		return nil, ErrNoMultipartForm
//	}
//
//	ce := req.Header.peek(strContentEncoding)
//	body := req.bodyBytes()
//	if bytes.Equal(ce, strGzip) {
//		// Do not care about memory usage here.
//		var err error
//		if body, err = AppendGunzipBytes(nil, body); err != nil {
//			return nil, fmt.Errorf("cannot gunzip request body: %s", err)
//		}
//	} else if len(ce) > 0 {
//		return nil, fmt.Errorf("unsupported Content-Encoding: %q", ce)
//	}
//
//	f, err := readMultipartForm(bytes.NewReader(body), req.multipartFormBoundary, len(body), len(body))
//	if err != nil {
//		return nil, err
//	}
//	req.multipartForm = f
//	return f, nil
//}

func marshalMultipartForm(f *multipart.Form, boundary string) ([]byte, error) {
	var buf ByteBuffer
	if err := WriteMultipartForm(&buf, f, boundary); err != nil {
		return nil, err
	}
	return buf.B, nil
}

// WriteMultipartForm writes the given multipart form f with the given
// boundary to w.
func WriteMultipartForm(w io.Writer, f *multipart.Form, boundary string) error {
	// Do not care about memory allocations here, since multipart
	// form processing is slooow.
	if len(boundary) == 0 {
		panic("BUG: form boundary cannot be empty")
	}

	mw := multipart.NewWriter(w)
	if err := mw.SetBoundary(boundary); err != nil {
		return fmt.Errorf("cannot use form boundary %q: %s", boundary, err)
	}

	// marshal values
	for k, vv := range f.Value {
		for _, v := range vv {
			if err := mw.WriteField(k, v); err != nil {
				return fmt.Errorf("cannot write form field %q value %q: %s", k, v, err)
			}
		}
	}

	// marshal files
	for k, fvv := range f.File {
		for _, fv := range fvv {
			vw, err := mw.CreateFormFile(k, fv.Filename)
			if err != nil {
				return fmt.Errorf("cannot create form file %q (%q): %s", k, fv.Filename, err)
			}
			fh, err := fv.Open()
			if err != nil {
				return fmt.Errorf("cannot open form file %q (%q): %s", k, fv.Filename, err)
			}
			if _, err = copyZeroAlloc(vw, fh); err != nil {
				return fmt.Errorf("error when copying form file %q (%q): %s", k, fv.Filename, err)
			}
			if err = fh.Close(); err != nil {
				return fmt.Errorf("cannot close form file %q (%q): %s", k, fv.Filename, err)
			}
		}
	}

	if err := mw.Close(); err != nil {
		return fmt.Errorf("error when closing multipart form writer: %s", err)
	}

	return nil
}

func readMultipartForm(r io.Reader, boundary string, size, maxInMemoryFileSize int) (*multipart.Form, error) {
	// Do not care about memory allocations here, since they are tiny
	// compared to multipart data (aka multi-MB files) usually sent
	// in multipart/form-data requests.

	if size <= 0 {
		panic(fmt.Sprintf("BUG: form size must be greater than 0. Given %d", size))
	}
	lr := io.LimitReader(r, int64(size))
	mr := multipart.NewReader(lr, boundary)
	f, err := mr.ReadForm(int64(maxInMemoryFileSize))
	if err != nil {
		return nil, fmt.Errorf("cannot read multipart/form-data body: %s", err)
	}
	return f, nil
}

// Reset clears request contents.
func (req *Request) Reset() {
	req.Header.Reset()
	req.resetSkipHeader()
}

func (req *Request) resetSkipHeader() {
	req.ResetBody()
	req.uri.Reset()
	req.parsedURI = false
	req.postArgs.Reset()
	req.parsedPostArgs = false
	req.isTLS = false
}

// RemoveMultipartFormFiles removes multipart/form-data temporary files
// associated with the request.
func (req *Request) RemoveMultipartFormFiles() {
	if req.multipartForm != nil {
		// Do not check for error, since these files may be deleted or moved
		// to new places by user code.
		req.multipartForm.RemoveAll()
		req.multipartForm = nil
	}
	req.multipartFormBoundary = ""
}

// Reset clears response contents.
func (resp *Response) Reset() {
	resp.Header.Reset()
	resp.resetSkipHeader()
	resp.SkipBody = false
}

func (resp *Response) resetSkipHeader() {
	resp.ResetBody()
}

// MayContinue returns true if the request contains
// 'Expect: 100-continue' header.
//
// The caller must do one of the following actions if MayContinue returns true:
//
//     - Either send StatusExpectationFailed response if request headers don't
//       satisfy the caller.
//     - Or send StatusContinue response before reading request body
//       with ContinueReadBody.
//     - Or close the connection.
func (req *Request) MayContinue() bool {
	return bytes.Equal(req.Header.peek(strExpect), str100Continue)
}

func (resp *Response) mustSkipBody() bool {
	return resp.SkipBody || resp.Header.mustSkipContentLength()
}

var errRequestHostRequired = errors.New("missing required Host header in request")

// WriteTo writes request to w. It implements io.WriterTo.
//func (req *Request) WriteTo(w io.Writer) (int64, error) {
//	return writeBufio(req, w)
//}
//
//// WriteTo writes response to w. It implements io.WriterTo.
//func (resp *Response) WriteTo(w io.Writer) (int64, error) {
//	return writeBufio(resp, w)
//}

func writeBufio(hw httpWriter, w io.Writer) (int64, error) {
	sw := acquireStatsWriter(w)
	bw := acquireBufioWriter(sw)
	err1 := hw.Write(bw)
	err2 := bw.Flush()
	releaseBufioWriter(bw)
	n := sw.bytesWritten
	releaseStatsWriter(sw)

	err := err1
	if err == nil {
		err = err2
	}
	return n, err
}

type statsWriter struct {
	w            io.Writer
	bytesWritten int64
}

func (w *statsWriter) Write(p []byte) (int, error) {
	n, err := w.w.Write(p)
	w.bytesWritten += int64(n)
	return n, err
}

func acquireStatsWriter(w io.Writer) *statsWriter {
	v := statsWriterPool.Get()
	if v == nil {
		return &statsWriter{
			w: w,
		}
	}
	sw := v.(*statsWriter)
	sw.w = w
	return sw
}

func releaseStatsWriter(sw *statsWriter) {
	sw.w = nil
	sw.bytesWritten = 0
	statsWriterPool.Put(sw)
}

var statsWriterPool sync.Pool

func acquireBufioWriter(w io.Writer) *bufio.Writer {
	v := bufioWriterPool.Get()
	if v == nil {
		return bufio.NewWriter(w)
	}
	bw := v.(*bufio.Writer)
	bw.Reset(w)
	return bw
}

func releaseBufioWriter(bw *bufio.Writer) {
	bufioWriterPool.Put(bw)
}

var bufioWriterPool sync.Pool

func (req *Request) onlyMultipartForm() bool {
	return req.multipartForm != nil && (req.body == nil || len(req.body.B) == 0)
}

type writeFlusher interface {
	io.Writer
	Flush() error
}

type flushWriter struct {
	wf writeFlusher
	bw *bufio.Writer
}

func (w *flushWriter) Write(p []byte) (int, error) {
	n, err := w.wf.Write(p)
	if err != nil {
		return 0, err
	}
	if err = w.wf.Flush(); err != nil {
		return 0, err
	}
	if err = w.bw.Flush(); err != nil {
		return 0, err
	}
	return n, nil
}

func (req *Request) closeBodyStream() error {
	if req.bodyStream == nil {
		return nil
	}
	var err error
	if bsc, ok := req.bodyStream.(io.Closer); ok {
		err = bsc.Close()
	}
	req.bodyStream = nil
	return err
}

func (resp *Response) closeBodyStream() error {
	if resp.bodyStream == nil {
		return nil
	}
	var err error
	if bsc, ok := resp.bodyStream.(io.Closer); ok {
		err = bsc.Close()
	}
	resp.bodyStream = nil
	return err
}

type httpWriter interface {
	Write(w *bufio.Writer) error
}

func limitedReaderSize(r io.Reader) int64 {
	lr, ok := r.(*io.LimitedReader)
	if !ok {
		return -1
	}
	return lr.N
}

func copyZeroAlloc(w io.Writer, r io.Reader) (int64, error) {
	vbuf := copyBufPool.Get()
	buf := vbuf.([]byte)
	n, err := io.CopyBuffer(w, r, buf)
	copyBufPool.Put(vbuf)
	return n, err
}

var copyBufPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 4096)
	},
}
