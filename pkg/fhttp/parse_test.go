package fhttp

import (
	"bufio"
	"bytes"
	"fmt"
	"net/http"
	"testing"

	"github.com/go-test/deep"
)

func TestParsePost(t *testing.T) {
	br := &bytes.Buffer{}
	plainReq := "POST /cgi-bin/process.cgi HTTP/1.1\r\n" +
		"User-Agent: Mozilla/4.0 (compatible; MSIE5.01; Windows NT)\r\n" +
		"Host: www.smth.com\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		"Content-Length: 49\r\n" +
		"Accept-Language: en-us\r\n" +
		"Accept-Encoding: gzip, deflate\r\n" +
		"Connection: Keep-Alive\r\n\r\n" +
		"licenseID=string&content=string&/paramsXML=string"
	br.Write([]byte(plainReq))
	req := createReq()
	bbr := bufio.NewReader(br)
	err := ParseRequestHeaders(req, bufio.NewReader(bbr))

	if err != nil {
		t.Fatalf("error parsing request headers %s", err.Error())
	}

	w := &bytes.Buffer{}
	ParseAndProxyRequestBody(req, bbr, w)

	expectedBody := "licenseID=string&content=string&/paramsXML=string"
	if w.String() != expectedBody {
		t.Fatalf("body error %s != %s", w.String(), expectedBody)
	}
}

func TestParseGet(t *testing.T) {
	br := &bytes.Buffer{}
	plainReq := "GET /cgi-bin/process.cgi HTTP/1.1\r\n" +
		"User-Agent: Mozilla/4.0 (compatible; MSIE5.01; Windows NT)\r\n" +
		"Host: www.smth.com\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		"Accept-Language: en-us\r\n" +
		"Accept-Encoding: gzip, deflate\r\n" +
		"Connection: Keep-Alive\r\n\r\n"

	br.Write([]byte(plainReq))

	req := createReq()
	bbr := bufio.NewReader(br)
	err := ParseRequestHeaders(req, bufio.NewReader(bbr))

	if err != nil {
		fmt.Printf("%#v", err)
		t.Fatalf("error parsing request headers: %s", err.Error())
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)
	WriteRequestHeaders(req, bw)
	ParseAndProxyRequestBody(req, bbr, bw)
	bw.Flush()

	if w.String() != plainReq {
		t.Fatal("not equal HTTP requests")
	}
}

func TestParseChunk(t *testing.T) {
	plainReq := "POST / HTTP/1.1\r\n" +
		"Host: cdn.api.example.com\r\n" +
		"Connection: keep-alive\r\n" +
		"Accept-Encoding: gzip, deflate\r\n" +
		"Accept: */*\r\n" +
		"User-Agent: python-requests/2.18.4\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		"Transfer-Encoding: chunked\r\n\r\n" +
		"4" +
		"\r\n" +
		"data" +
		"\r\n" +
		"4" +
		"\r\n" +
		"test" +
		"\r\n" +
		"0"

	br := &bytes.Buffer{}
	br.Write([]byte(plainReq))

	req := createReq()
	bbr := bufio.NewReader(br)
	err := ParseRequestHeaders(req, bufio.NewReader(bbr))

	if err != nil {
		fmt.Printf("%#v", err)
		t.Fatalf("error parsing request headers: %s", err.Error())
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)

	WriteRequestHeaders(req, bw)
	ParseAndProxyRequestBody(req, bbr, bw)
	bw.Flush()

	rRes, err := http.ReadRequest(bufio.NewReader(w))
	rExpected, err := http.ReadRequest(bufio.NewReader(bytes.NewBufferString(plainReq)))
	if diff := deep.Equal(rRes, rExpected); diff != nil {
		t.Error(diff)
	}
}

func TestRequestKeepAlive(t *testing.T) {
	br := &bytes.Buffer{}
	plainReq := "POST /cgi-bin/process.cgi HTTP/1.1\r\n" +
		"User-Agent: Mozilla/4.0 (compatible; MSIE5.01; Windows NT)\r\n" +
		"Host: www.smth.com\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		"Content-Length: 49\r\n" +
		"Accept-Language: en-us\r\n" +
		"Accept-Encoding: gzip, deflate\r\n" +
		"Connection: Keep-Alive\r\n\r\n" +
		"licenseID=string&content=string&/paramsXML=stringPOST /cgi-bin/process.cgi HTTP/1.1\r\n" +
		"User-Agent: Mozilla/4.0 (compatible; MSIE5.01; Windows NT)\r\n" +
		"Host: www.smth.com\r\n" +
		"Content-Type: application/x-www-form-urlencoded\r\n" +
		"Content-Length: 49\r\n" +
		"Accept-Language: en-us\r\n" +
		"Accept-Encoding: gzip, deflate\r\n" +
		"Connection: Keep-Alive\r\n\r\n" +
		"licenseID=string&content=string&/paramsXML=string"

	br.Write([]byte(plainReq))
	req := createReq()
	bbr := bufio.NewReader(br)
	err := ParseRequestHeaders(req, bufio.NewReader(bbr))

	if err != nil {
		t.Fatalf("error parsing request headers %s", err.Error())
	}

	w := &bytes.Buffer{}
	ParseAndProxyRequestBody(req, bbr, w)

	expectedBody := "licenseID=string&content=string&/paramsXML=string"
	if w.String() != expectedBody {
		t.Fatalf("body error %s != %s", w.String(), expectedBody)
	}

	req.Reset()
	err = ParseRequestHeaders(req, bufio.NewReader(bbr))

	if err != nil {
		t.Fatalf("error parsing request headers %s", err.Error())
	}

	w = &bytes.Buffer{}
	ParseAndProxyRequestBody(req, bbr, w)

	if w.String() != expectedBody {
		t.Fatalf("body error %s != %s", w.String(), expectedBody)
	}
}

func TestResponse(t *testing.T) {
	plainReq := "HTTP/1.1 404 Not Found\r\n" +
		"Date: Sun, 18 Oct 2012 10:36:20 GMT\r\n" +
		"Server: Apache/2.2.14 (Win32)\r\n" +
		"Content-Length: 215\r\n" +
		"Connection: Closed\r\n" +
		"Content-Type: text/html; charset=iso-8859-1\r\n\r\n" +
		"<!DOCTYPE HTML PUBLIC \"-//IETF//DTD HTML 2.0//EN\">\r\n" +
		"<html>\r\n" +
		"<head>\r\n" +
		"   <title>404 Not Found</title>\r\n" +
		"</head>\r\n" +
		"<body>\r\n" +
		"   <h1>Not Found</h1>\r\n" +
		"   <p>The requested URL /t.html was not found on this server.</p>\r\n" +
		"</body>\r\n" +
		"</html>"

	br := &bytes.Buffer{}
	br.Write([]byte(plainReq))

	resp := createResp()
	bbr := bufio.NewReader(br)
	err := ParseResponseHeaders(resp, bufio.NewReader(bbr))

	if err != nil {
		t.Fatalf("error parsing request headers %s", err.Error())
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)

	WriteResponseHeaders(resp, bw)
	ParseAndProxyResponseBody(resp, bbr, bw)
	bw.Flush()

	rRes, err := http.ReadResponse(bufio.NewReader(w), nil)
	rExpected, err := http.ReadResponse(bufio.NewReader(bytes.NewBufferString(plainReq)), nil)
	if diff := deep.Equal(rRes, rExpected); diff != nil {
		t.Error(diff)
	}
}

func TestChunkedResponse(t *testing.T) {
	plainResp := "HTTP/1.1 200 OK\r\n" +
		"Transfer-Encoding: chunked\r\n" +
		"Content-Type: application/json; charset=utf-8\r\n" +
		"Date: Tue, 14 May 2019 05:58:55 GMT\r\n" +
		"\r\n" +
		"e88\r\n" +
		"{\"result\":{\"lastActionTimes\":[{\"id\":109654091,\"lastActionTime\":1557805845,\"timeDiff\":7690},{\"id\":117873152,\"lastActionTime\":1557810019,\"timeDiff\":3516},{\"id\":26826205,\"lastActionTime\":1557761405,\"timeDiff\":52130},{\"id\":161977027,\"lastActionTime\":1557772098,\"timeDiff\":41437},{\"id\":74470985,\"lastActionTime\":1557808434,\"timeDiff\":5101},{\"id\":116997508,\"lastActionTime\":1557318749,\"timeDiff\":494786},{\"id\":77539574,\"lastActionTime\":1557480342,\"timeDiff\":333193},{\"id\":144986387,\"lastActionTime\":1557766220,\"timeDiff\":47315},{\"id\":35694984,\"lastActionTime\":1557807629,\"timeDiff\":5906},{\"id\":29901815,\"lastActionTime\":1557673022,\"timeDiff\":140513},{\"id\":148202369,\"lastActionTime\":1557782597,\"timeDiff\":30938},{\"id\":73739521,\"lastActionTime\":1556708846,\"timeDiff\":1104689},{\"id\":155406164,\"lastActionTime\":1556947402,\"timeDiff\":866133},{\"id\":157642736,\"lastActionTime\":1557772138,\"timeDiff\":41397},{\"id\":158132735,\"lastActionTime\":1557679768,\"timeDiff\":133767},{\"id\":148834492,\"lastActionTime\":1557583102,\"timeDiff\":230433},{\"id\":16012340,\"lastActionTime\":1557813419,\"timeDiff\":116},{\"id\":120813338,\"lastActionTime\":1557249642,\"timeDiff\":563893},{\"id\":81826991,\"lastActionTime\":1557811081,\"timeDiff\":2454},{\"id\":1928287,\"lastActionTime\":1557783357,\"timeDiff\":30178},{\"id\":138595677,\"lastActionTime\":1557812510,\"timeDiff\":1025},{\"id\":113485727,\"lastActionTime\":1557439902,\"timeDiff\":373633},{\"id\":153020853,\"lastActionTime\":1557750197,\"timeDiff\":63338},{\"id\":126313560,\"lastActionTime\":1557754458,\"timeDiff\":59077},{\"id\":70004606,\"lastActionTime\":1557780373,\"timeDiff\":33162},{\"id\":157388620,\"lastActionTime\":1557759389,\"timeDiff\":54146},{\"id\":74702661,\"lastActionTime\":1557813535,\"timeDiff\":0},{\"id\":162116480,\"lastActionTime\":1557758339,\"timeDiff\":55196},{\"id\":162503484,\"lastActionTime\":1557811760,\"timeDiff\":1775},{\"id\":134305051,\"lastActionTime\":1557812969,\"timeDiff\":566},{\"id\":151014373,\"lastActionTime\":1557809540,\"timeDiff\":3995},{\"id\":79279803,\"lastActionTime\":1557767155,\"timeDiff\":46380},{\"id\":127922269,\"lastActionTime\":1556368755,\"timeDiff\":1444780},{\"id\":105220719,\"lastActionTime\":1557806314,\"timeDiff\":7221},{\"id\":113426567,\"lastActionTime\":1557388251,\"timeDiff\":425284},{\"id\":161477091,\"lastActionTime\":1557675387,\"timeDiff\":138148},{\"id\":97900160,\"lastActionTime\":1557787479,\"timeDiff\":26056},{\"id\":109825878,\"lastActionTime\":1557810232,\"timeDiff\":3303},{\"id\":97598994,\"lastActionTime\":1557756237,\"timeDiff\":57298},{\"id\":103207971,\"lastActionTime\":1557812754,\"timeDiff\":781},{\"id\":66608622,\"lastActionTime\":1557779803,\"timeDiff\":33732},{\"id\":122820245,\"lastActionTime\":1557804299,\"timeDiff\":9236},{\"id\":143367362,\"lastActionTime\":1557774677,\"timeDiff\":38858},{\"id\":107672522,\"lastActionTime\":1557810900,\"timeDiff\":2635},{\"id\":118322523,\"lastActionTime\":1557808647,\"timeDiff\":4888},{\"id\":118186840,\"lastActionTime\":1557769540,\"timeDiff\":43995},{\"id\":2630563,\"lastActionTime\":1557578960,\"timeDiff\":234575},{\"id\":89249889,\"lastActionTime\":1557805372,\"timeDiff\":8163},{\"id\":20572629,\"lastActionTime\":1557794761,\"timeDiff\":18774},{\"id\":133071738,\"lastActionTime\":1557770325,\"timeDiff\":43210},{\"id\":110291574,\"lastActionTime\":1557787516,\"timeDiff\":26019},{\"id\":145703023,\"lastActionTime\":1557770775,\"timeDiff\":42760},{\"id\":150796414,\"lastActionTime\":1557805431,\"timeDiff\":8104},{\"id\":161921823,\"lastActionTime\":1557758642,\"timeDiff\":54893},{\"id\":161549159,\"lastActionTime\":1557784117,\"timeDiff\":29418},{\"id\":138361510,\"lastActionTime\":1557781192,\"timeDiff\":32343},{\"id\":120276732,\"lastActionTime\":1557720803,\"timeDiff\":92732},{\"id\":116094341,\"lastActionTime\":1557812606,\"timeDiff\":929},{\"id\":148983792,\"lastActionTime\":1557767054,\"timeDiff\":46481},{\"id\":92461201,\"lastActionTime\":1557264759,\"timeDiff\":548776}]}}"

	br := &bytes.Buffer{}
	br.Write([]byte(plainResp))

	resp := createResp()
	bbr := bufio.NewReader(br)
	err := ParseResponseHeaders(resp, bufio.NewReader(bbr))

	if err != nil {
		t.Fatalf("error parsing request headers %s", err.Error())
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)

	WriteResponseHeaders(resp, bw)
	ParseAndProxyResponseBody(resp, bbr, bw)
	bw.Flush()

	rRes, err := http.ReadResponse(bufio.NewReader(w), nil)
	rExpected, err := http.ReadResponse(bufio.NewReader(bytes.NewBufferString(plainResp)), nil)
	if diff := deep.Equal(rRes, rExpected); diff != nil {
		t.Error(diff)
	}
}

func TestGetResponse(t *testing.T) {
	plainResp := "HTTP/1.0 200 OK\r\n" +
		"Connection: close\r\n" +
		"Content-Length: 110\r\n" +
		"Content-Type: text/html\r\n" +
		"Date: Tue, 14 May 2019 19:52:01 GMT\r\n" +
		"Last-Modified: Tue, 14 May 2019 10:25:35 GMT\r\n" +
		"Server: SimpleHTTP/0.6 Python/2.7.9\r\n" +
		"\r\n" +
		"<html><head><title>HTTP Hello World</title></head><body><h1>Hello from app-7578dcbcb9-w2rbr</h1></body></html"

	br := &bytes.Buffer{}
	br.Write([]byte(plainResp))

	resp := createResp()
	bbr := bufio.NewReader(br)
	err := ParseResponseHeaders(resp, bufio.NewReader(bbr))

	if err != nil {
		t.Fatalf("error parsing request headers %s", err.Error())
	}

	w := &bytes.Buffer{}
	bw := bufio.NewWriter(w)

	WriteResponseHeaders(resp, bw)
	ParseAndProxyResponseBody(resp, bbr, bw)
	bw.Flush()

	rRes, err := http.ReadResponse(bufio.NewReader(w), nil)
	rExpected, err := http.ReadResponse(bufio.NewReader(bytes.NewBufferString(plainResp)), nil)
	if diff := deep.Equal(rRes, rExpected); diff != nil {
		t.Error(diff)
	}
}

func createReq() *Request {
	req := &Request{}
	req.Header.DisableNormalizing()
	return req
}

func createResp() *Response {
	resp := &Response{}
	resp.Header.DisableNormalizing()
	return resp
}
