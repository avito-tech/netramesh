package fhttp

import (
	"bufio"
	"bytes"
	"errors"
	"fmt"
	"io"
	"sync"
)

var (
	errEmptyHexNum    = errors.New("empty hex number")
	errTooLargeHexNum = errors.New("too large hex number")
	maxIntChars       = 18
	maxHexIntChars    = 15
)

var RequestsPool = sync.Pool{
	New: func() interface{} {
		r := &Request{}
		r.Header.DisableNormalizing()
		return r
	},
}

var ResponsePool = sync.Pool{
	New: func() interface{} {
		r := &Response{}
		r.Header.DisableNormalizing()
		return r
	},
}

var bytesPool = sync.Pool{
	New: func() interface{} {
		return make([]byte, 0xfff)
	},
}

// ParseRequest parses requests from r (reset req for keep-alive support)
func ParseRequestHeaders(req *Request, r *bufio.Reader) error {
	return req.Header.Read(r)
}

// WriteRequestHeaders writes headers to w
func WriteRequestHeaders(req *Request, w io.Writer) (int64, error) {
	return req.Header.WriteTo(w)
}

// ParseAndProxyRequestBody parses and proxies request body
func ParseAndProxyRequestBody(req *Request, r *bufio.Reader, w io.Writer) error {
	b := bytesPool.Get().([]byte)
	err := readBody(r, w, req.Header.ContentLength(), b)
	bytesPool.Put(b)
	return err
}

// ParseResponseHeaders parses response from r
func ParseResponseHeaders(resp *Response, r *bufio.Reader) error {
	return resp.Header.Read(r)
}

// ParseAndProxyResponseBody parses and proxies response body
func ParseAndProxyResponseBody(resp *Response, r *bufio.Reader, w io.Writer) error {
	b := bytesPool.Get().([]byte)
	err := readBody(r, w, resp.Header.ContentLength(), b)
	bytesPool.Put(b)
	return err
}

// WriteResponseHeaders writes response headers to w
func WriteResponseHeaders(resp *Response, w io.Writer) (int64, error) {
	return resp.Header.WriteTo(w)
}

func readBody(r *bufio.Reader, w io.Writer, contentLength int, dst []byte) error {
	dst = dst[:0]
	if contentLength >= 0 {
		_, err := appendBodyFixedSize(r, w, dst, contentLength)
		return err
	}
	if contentLength == -1 {
		return readBodyChunked(r, w, dst)
	}
	return readBodyIdentity(r, w, dst)
}

func readBodyIdentity(r *bufio.Reader, w io.Writer, dst []byte) error {
	dst = dst[:cap(dst)]
	_, err := io.CopyBuffer(w, r, dst)
	return err
}

func appendBodyFixedSize(r *bufio.Reader, w io.Writer, dst []byte, n int) ([]byte, error) {
	if n == 0 {
		return dst, nil
	}
	remain := n
	dst = dst[0:cap(dst)]

	for {
		if len(dst) > remain {
			dst = dst[0:remain]
		}
		nn, err := r.Read(dst)
		if nn <= 0 {
			if err != nil {
				if err == io.EOF {
					err = io.ErrUnexpectedEOF
				}
				return dst, err
			}
			panic(fmt.Sprintf("BUG: bufio.Read() returned (%d, nil)", nn))
		}
		remain -= nn
		_, err = w.Write(dst)
		if err != nil {
			return dst, err
		}
		if remain == 0 {
			return dst, nil
		}
	}
}

func readBodyChunked(r *bufio.Reader, w io.Writer, dst []byte) error {
	if len(dst) > 0 {
		panic("BUG: expected zero-length buffer")
	}

	strCRLFLen := len(strCRLF)
	for {
		chunkSize, err := parseChunkSize(r, w)
		if err != nil {
			return err
		}
		b, err := appendBodyFixedSize(r, w, dst, chunkSize+strCRLFLen)
		if err != nil {
			return err
		}
		if !bytes.Equal(b[len(b)-strCRLFLen:], strCRLF) {
			return fmt.Errorf("cannot find crlf at the end of chunk")
		}
		dst = dst[:0]
		if chunkSize == 0 {
			return nil
		}
	}
}

func parseChunkSize(r *bufio.Reader, w io.Writer) (int, error) {
	n, err := readHexInt(r, w)
	if err != nil {
		return -1, err
	}
	c, err := r.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("cannot read '\r' char at the end of chunk size: %s", err)
	}

	_, err = w.Write([]byte{c})
	if err != nil {
		return -1, fmt.Errorf("cannot write to w: %s", err.Error())
	}

	if c != '\r' {
		return -1, fmt.Errorf("unexpected char %q at the end of chunk size. Expected %q", c, '\r')
	}
	c, err = r.ReadByte()
	if err != nil {
		return -1, fmt.Errorf("cannot read '\n' char at the end of chunk size: %s", err)
	}

	_, err = w.Write([]byte{c})
	if err != nil {
		return -1, fmt.Errorf("cannot write to w: %s", err.Error())
	}

	if c != '\n' {
		return -1, fmt.Errorf("unexpected char %q at the end of chunk size. Expected %q", c, '\n')
	}
	return n, nil
}

func round2(n int) int {
	if n <= 0 {
		return 0
	}
	n--
	x := uint(0)
	for n > 0 {
		n >>= 1
		x++
	}
	return 1 << x
}

func readHexInt(r *bufio.Reader, w io.Writer) (int, error) {
	n := 0
	i := 0
	var k int
	for {
		c, err := r.ReadByte()
		if err != nil {
			if err == io.EOF && i > 0 {
				return n, nil
			}
			return -1, err
		}
		k = int(hex2intTable[c])
		if k == 16 {
			if i == 0 {
				return -1, errEmptyHexNum
			}
			r.UnreadByte()
			return n, nil
		}

		_, err = w.Write([]byte{c})
		if err != nil {
			return -1, fmt.Errorf("cannot write to w: %s", err.Error())
		}

		if i >= maxHexIntChars {
			return -1, errTooLargeHexNum
		}
		n = (n << 4) | k
		i++
	}
}

var hex2intTable = func() []byte {
	b := make([]byte, 256)
	for i := 0; i < 256; i++ {
		c := byte(16)
		if i >= '0' && i <= '9' {
			c = byte(i) - '0'
		} else if i >= 'a' && i <= 'f' {
			c = byte(i) - 'a' + 10
		} else if i >= 'A' && i <= 'F' {
			c = byte(i) - 'A' + 10
		}
		b[i] = c
	}
	return b
}()
